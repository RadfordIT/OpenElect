package main

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"

	"github.com/gin-gonic/gin"
)

func storageSetup() {
	if _, err := os.Stat("/OpenElect/videos"); os.IsNotExist(err) {
		err := os.Mkdir("/OpenElect/videos", 0755)
		if err != nil {
			log.Fatalf("Failed to create videos directory: %v", err)
		}
	}
}

func uploadVideo(filename string, video multipart.File) error {
	dstPath := fmt.Sprintf("/OpenElect/videos/%s", filename)
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, video); err != nil {
		return err
	}
	return nil
}

func deleteVideo(filename string) error {
	dstPath := fmt.Sprintf("/OpenElect/videos/%s", filename)
	err := os.Remove(dstPath)
	return err
}

func getVideoRoutes() {
	r.GET("/video/:filename", authMiddleware(), func(c *gin.Context) {
		filename := c.Param("filename")
		dstPath := fmt.Sprintf("/OpenElect/videos/%s", filename)
		video, err := os.ReadFile(dstPath)
		if err != nil {
			fmt.Println(err)
			c.String(500, "Failed to read video: %v", err)
			return
		}
		c.Data(200, "video/mp4", video)
	})
}
