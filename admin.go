package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func adminRoutes() {
	r.GET("/admin", adminAuthMiddleware(), func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.tmpl", gin.H{
			"colors": colorsEditor.GetStringMapString("colors"),
			"colorNames": [...]string{
				"accent",
				"accentContent",
				"base100",
				"base200",
				"base300",
				"baseContent",
				"error",
				"errorContent",
				"info",
				"infoContent",
				"neutral",
				"neutralContent",
				"primary",
				"primaryContent",
				"secondary",
				"secondaryContent",
				"success",
				"successContent",
				"warning",
				"warningContent",
			},
			"positions": configEditor.GetStringSlice("positions"),
		})
	})
	r.POST("/admin", adminAuthMiddleware(), func(c *gin.Context) {
		colors := c.PostFormMap("colors")
		colorsEditor.Set("colors", colors)
		colorsEditor.WriteConfig()
		positions := c.PostFormArray("position[]")
		configEditor.Set("positions", positions)
		configEditor.WriteConfig()
		c.Redirect(http.StatusSeeOther, "/admin")
	})
}
