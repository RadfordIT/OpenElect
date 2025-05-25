package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/gin-gonic/gin"
	"log"
	"mime/multipart"
	"os"
)

var (
	serviceClient    *azblob.Client
	containerName    = "videos"
)

func storageSetup() {
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
}

func uploadVideo(filename string, video multipart.File) error {
	_, err := serviceClient.UploadStream(context.Background(), containerName, filename, video, nil)
	return err
}

func deleteVideo(filename string) error {
	_, err := serviceClient.DeleteBlob(context.Background(), containerName, filename, nil)
	return err
}

func getVideoRoutes() {
	r.GET("/video/:filename", authMiddleware(), func(c *gin.Context) {
		filename := c.Param("filename")
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
	})
}
