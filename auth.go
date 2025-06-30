package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

var (
	clientID     = os.Getenv("CLIENT_ID")
	clientSecret = os.Getenv("CLIENT_SECRET")
	tenantID     = os.Getenv("TENANT_ID")
	redirectURL  = "http://localhost:8080/callback"
	provider     *oidc.Provider
	oauth2Config oauth2.Config
)

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		token := session.Get("user_id")
		if token == nil {
			c.Abort()
			c.Redirect(http.StatusFound, "/login")
		}
		c.Next()
	}
}

func candidateAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		token := session.Get("user_id")
		if token == nil {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		groups := session.Get("groups").([]string)
		if !slices.Contains(groups, configEditor.GetString("candidategroup")) {
			c.String(http.StatusUnauthorized, "Unauthorized: you are not a candidate")
			c.Abort()
			return
		}
		c.Next()
	}
}

func adminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		token := session.Get("user_id")
		if token == nil {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		groups := session.Get("groups").([]string)
		if !slices.Contains(groups, configEditor.GetString("admingroup")) {
			c.String(http.StatusUnauthorized, "Unauthorized: you are not an admin")
			c.Abort()
			return
		}
		c.Next()
	}
}

func authSetup() {
	var err error
	provider, err = oidc.NewProvider(context.Background(), "https://login.microsoftonline.com/"+tenantID+"/v2.0")
	if err != nil {
		log.Fatal(err)
	}
	oauth2Config = oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "offline_access"},
	}
}

func fetchUserGroups(accessToken string) ([]string, error) {
	url := "https://graph.microsoft.com/v1.0/me/memberOf"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch groups, status code: %d", resp.StatusCode)
	}

	var result struct {
		Value []struct {
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var groups []string
	for _, group := range result.Value {
		groups = append(groups, group.DisplayName)
	}
	return groups, nil
}

func loginRoutes() {
	r.GET("/login", func(c *gin.Context) {
		authCodeURL := oauth2Config.AuthCodeURL("state", oauth2.AccessTypeOffline)
		c.Redirect(http.StatusFound, authCodeURL)
	})
	r.GET("/callback", func(c *gin.Context) {
		session := sessions.Default(c)
		state := c.Query("state")
		if state != "state" {
			c.String(http.StatusBadRequest, "Invalid state")
			return
		}
		code := c.Query("code")
		if code == "" {
			c.String(http.StatusBadRequest, "Code not found")
			return
		}
		token, err := oauth2Config.Exchange(context.Background(), code)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to exchange token: %v", err)
			return
		}
		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok {
			c.String(http.StatusInternalServerError, "No ID token found")
			return
		}

		verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
		idToken, err := verifier.Verify(context.Background(), rawIDToken)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to verify ID token: %v", err)
			return
		}
		var claims map[string]interface{}
		if err := idToken.Claims(&claims); err != nil {
			c.String(http.StatusInternalServerError, "Failed to extract claims: %v", err)
			return
		}

		pfpURL := "https://graph.microsoft.com/v1.0/me/photo/$value"
		req, _ := http.NewRequest("GET", pfpURL, nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		pfpclient := &http.Client{}
		resp, err := pfpclient.Do(req)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to fetch profile picture: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			profilePictureData, err := io.ReadAll(resp.Body)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to read profile picture: %v", err)
				return
			}
			fileName := "pfp/" + claims["sub"].(string) + ".jpg"
			pfp, err := cropToSquare(profilePictureData)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to crop profile picture: %v", err)
				return
			}
			saveProfilePicture(claims["sub"].(string), pfp)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to save profile picture: %v", err)
				return
			}
			session.Set("pfp", fileName)
		} else {
			log.Println("Warning: no profile picture found", resp)
			session.AddFlash("Warning: no profile picture found")
			session.Set("pfp", "./pfp/default_pfp.jpg")
		}

		emailUrl := "https://graph.microsoft.com/v1.0/me"
		req, _ = http.NewRequest("GET", emailUrl, nil)
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		emailclient := &http.Client{}
		resp, err = emailclient.Do(req)
		var userEmail string
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to fetch email: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				c.String(http.StatusInternalServerError, "Failed to decode email: %v", err)
				return
			}
			if email, ok := result["mail"].(string); ok && email != "" {
				userEmail = email
			} else if upn, ok := result["userPrincipalName"].(string); ok && upn != "" {
				userEmail = upn
			} else {
				c.String(http.StatusInternalServerError, "Failed to extract email")
				return
			}
		} else {
			c.String(http.StatusInternalServerError, "Failed to fetch email: %v", err)
			return
		}

		groups, err := fetchUserGroups(token.AccessToken)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to extract groups: %v", err)
			return
		}
		session.Set("name", claims["name"])
		session.Set("user_id", claims["sub"])
		session.Set("groups", groups)
		session.Set("email", userEmail)
		if err := session.Save(); err != nil {
			c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
			return
		}
		fmt.Println(session.Get("user_id"), session.Get("groups"), session.Get("pfp"))
		c.Redirect(http.StatusFound, "/")
	})
	r.GET("/logout", authMiddleware(), func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
		session.Save()
		logoutURL := fmt.Sprintf(
			"https://login.microsoftonline.com/%s/oauth2/v2.0/logout?post_logout_redirect_uri=%s",
			tenantID,
			"http://localhost:8080/login",
		)
		c.Redirect(http.StatusFound, logoutURL)
	})
}
