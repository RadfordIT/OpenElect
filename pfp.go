package main

import (
	"bytes"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
)

func cropToSquare(imageData []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	size := width
	if height < width {
		size = height
	}

	startX := (width - size) / 2
	startY := (height - size) / 2
	cropRect := image.Rect(startX, startY, startX+size, startY+size)

	cropped := image.NewRGBA(cropRect)
	draw.Draw(cropped, cropped.Bounds(), img, cropRect.Min, draw.Src)

	var buffer bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buffer, cropped, nil)
	case "png":
		err = png.Encode(&buffer, cropped)
	default:
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func pfpRoutes() {
	r.GET("/pfp", authMiddleware(), func(c *gin.Context) {
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
}
