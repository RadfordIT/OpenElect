package main

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"log"
	"mime/multipart"
	"os"
)

var (
	connectionString = os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
	serviceClient    *azblob.Client
	containerName    = "videos"
)

func storageSetup() {
	var err error
	serviceClient, err = azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		log.Fatalf("Failed to create Storage Client: %v", err)
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
