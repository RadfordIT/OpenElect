package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
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

func saveProfilePicture(userId string, file []byte) error {
	fmt.Println("Uploading video:", userId+".jpg", "Size:", len(file))
	reader := bytes.NewReader(file)
	_, err := minioClient.PutObject(
		context.Background(),
		bucketName,
		"pfp/"+userId+".jpg",
		reader,
		int64(len(file)),
		minio.PutObjectOptions{ContentType: "image/jpeg"},
	)
	if err != nil {
		return err
	}
	return nil
}

func pfpRoutes() {
	r.GET("/pfp", authMiddleware(), func(c *gin.Context) {
		userId := c.DefaultQuery("user", "")
		if userId != "" {
			reader, err := minioClient.GetObject(context.Background(), bucketName, "pfp/"+userId+".jpg", minio.GetObjectOptions{})
			if err != nil {
				c.String(500, "Failed to read profile picture: %v", err)
				return
			}
			defer reader.Close()
			info, err := reader.Stat()
			if err != nil {
				c.String(500, "Failed to get profile picture info: %v", err)
				return
			}
			http.ServeContent(c.Writer, c.Request, info.Key, info.LastModified, reader)
			return
		}
		session := sessions.Default(c)
		pfp := session.Get("pfp")
		fmt.Println(pfp)
		if pfp == nil {
			pfp = "pfp/default_pfp.jpg"
		}
		reader, err := minioClient.GetObject(context.Background(), bucketName, pfp.(string), minio.GetObjectOptions{})
		if err != nil {
			c.String(500, "Failed to read profile picture: %v", err)
			return
		}
		defer reader.Close()
		info, err := reader.Stat()
		if err != nil {
			c.String(500, "Failed to get profile picture info: %v", err)
			return
		}
		http.ServeContent(c.Writer, c.Request, info.Key, info.LastModified, reader)
	})
}
