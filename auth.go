package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"os"
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
		if !contains(groups, configEditor.GetString("candidategroup")) {
			c.String(http.StatusUnauthorized, "Unauthorized")
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
		if !contains(groups, configEditor.GetString("admingroup")) {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		c.Next()
	}
}

func contains(groups []string, s string) bool {
	for _, group := range groups {
		if group == s {
			return true
		}
	}
	return false
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
			fileName := "./pfp/" + claims["sub"].(string) + ".jpg"
			err = os.WriteFile(fileName, profilePictureData, 0644)
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to save profile picture: %v", err)
				return
			}
			session.Set("pfp", fileName)
		} else {
			c.String(http.StatusInternalServerError, "Failed to fetch profile picture: %v", err)
			session.Set("pfp", "./pfp/default_pfp.jpg")
		}

		groups, err := fetchUserGroups(token.AccessToken)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to extract groups: %v", err)
			return
		}
		session.Set("name", claims["name"])
		session.Set("user_id", claims["sub"])
		session.Set("groups", groups)
		if err := session.Save(); err != nil {
			c.String(http.StatusInternalServerError, "Failed to save session: %v", err)
			return
		}
		fmt.Println(session.Get("user_id"), session.Get("groups"), session.Get("pfp"))
		c.Redirect(http.StatusFound, "/")
	})
	r.GET("/pfp", func(c *gin.Context) {
		userId := c.DefaultQuery("user", "")
		if userId != "" {
			http.ServeFile(c.Writer, c.Request, "./pfp/"+userId+".jpg")
		}
		session := sessions.Default(c)
		pfp := session.Get("pfp")
		if pfp == nil {
			pfp = "./pfp/default_pfp.jpg"
		}
		http.ServeFile(c.Writer, c.Request, pfp.(string))
	})
	r.GET("/logout", func(c *gin.Context) {
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
