package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func adminRoutes() {
	r.GET("/admin", adminAuthMiddleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.tmpl", gin.H{
			"colors": colorsEditor.GetStringMapString("colors"),
		})
	})
	r.POST("/admin", adminAuthMiddleware(), func(c *gin.Context) {
		colors := c.PostFormMap("colors")
		colorsEditor.Set("colors", colors)
		colorsEditor.WriteConfig()
		c.Redirect(http.StatusSeeOther, "/admin")
	})
}
