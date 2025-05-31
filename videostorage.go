package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
)

var (
	serviceClient *azblob.Client
	containerName = "videos"
)

func storageSetup() {
	if storageProvider == "azure" {
		var err error
		connectionString := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
		serviceClient, err = azblob.NewClientFromConnectionString(connectionString, nil)
		if err != nil {
			log.Fatalf("Failed to create Storage Client: %v", err)
		}
		pager := serviceClient.NewListBlobsFlatPager(containerName, &azblob.ListBlobsFlatOptions{
			Include: azblob.ListBlobsInclude{Snapshots: true, Versions: true},
		})

		fmt.Println("List blobs flat:")
		for pager.More() {
			resp, err := pager.NextPage(context.TODO())
			if err != nil {
				log.Fatalf("Failed to list blobs: %v", err)
			}

			for _, blob := range resp.Segment.BlobItems {
				fmt.Println(*blob.Name)
			}
		}
	} else if storageProvider == "local" {
		if _, err := os.Stat("/OpenElect/videos"); os.IsNotExist(err) {
			err := os.Mkdir("/OpenElect/videos", 0755)
			if err != nil {
				log.Fatalf("Failed to create videos directory: %v", err)
			}
		}
	} else {
		log.Fatalf("Unsupported storage provider: %s", storageProvider)
	}
}

func uploadVideo(filename string, video multipart.File) error {
	if storageProvider == "azure" {
		_, err := serviceClient.UploadStream(context.Background(), containerName, filename, video, nil)
		return err
	} else if storageProvider == "local" {
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
	} else {
		return fmt.Errorf("unsupported storage provider: %s", storageProvider)
	}
}

func deleteVideo(filename string) error {
	if storageProvider == "azure" {
		_, err := serviceClient.DeleteBlob(context.Background(), containerName, filename, nil)
		return err
	} else if storageProvider == "local" {
		dstPath := fmt.Sprintf("/OpenElect/videos/%s", filename)
		err := os.Remove(dstPath)
		return err
	} else {
		return fmt.Errorf("unsupported storage provider: %s", storageProvider)
	}
}

func getVideoRoutes() {
	r.GET("/video/:filename", authMiddleware(), func(c *gin.Context) {
		filename := c.Param("filename")
		if storageProvider == "azure" {
			response, err := serviceClient.DownloadStream(context.Background(), containerName, filename, nil)
			if err != nil {
				fmt.Println(err)
				c.String(500, "Failed to download video: %v", err)
				return
			}
			video := bytes.Buffer{}
			retryReader := response.NewRetryReader(context.Background(), &azblob.RetryReaderOptions{})
			_, err = video.ReadFrom(retryReader)
			if err != nil {
				fmt.Println(err)
				c.String(500, "Failed to read video: %v", err)
				return
			}
			err = retryReader.Close()
			if err != nil {
				fmt.Println(err)
				c.String(500, "Failed to close video: %v", err)
				return
			}
			c.Data(200, "video/mp4", video.Bytes())
		} else if storageProvider == "local" {
			filename := c.Param("filename")
			dstPath := fmt.Sprintf("/OpenElect/videos/%s", filename)
			video, err := os.ReadFile(dstPath)
			if err != nil {
				fmt.Println(err)
				c.String(500, "Failed to read video: %v", err)
				return
			}
			c.Data(200, "video/mp4", video)
		} else {
			c.String(500, "Unsupported storage provider: %s", storageProvider)
		}
	})
}
