package main

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *minio.Client
var bucketName = "openelect"

func storageSetup() {
	var err error
	ctx := context.Background()
	endpoint := os.Getenv("STORAGE_URL")
	accessKeyID := os.Getenv("STORAGE_ACCESS_KEY")
	secretAccessKey := os.Getenv("STORAGE_SECRET_ACCESS_KEY")
	fmt.Println("Initializing MinIO client with endpoint:", endpoint, "and credentials:", accessKeyID, secretAccessKey)
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatal("Error initializing MinIO client:", err)
		return
	}
	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		exists, err := minioClient.BucketExists(ctx, bucketName)
		if err != nil {
			log.Fatal("Error checking if bucket exists:", err)
			return
		}
		if !exists {
			log.Fatal("Bucket does not exist and could not be created:", err)
			return
		}
	}
	fmt.Println("MinIO client initialized successfully")
}

type Sizer interface {
	Size() int64
}

func uploadVideo(filename string, video multipart.File) error {
	fmt.Println("Uploading video:", filename, "Size:", video.(Sizer).Size())
	_, err := minioClient.PutObject(
		context.Background(),
		bucketName,
		"video/"+filename,
		video,
		video.(Sizer).Size(),
		minio.PutObjectOptions{ContentType: "video/mp4"},
	)
	if err != nil {
		return err
	}
	return nil
}

func deleteVideo(filename string) error {
	err := minioClient.RemoveObject(context.Background(), bucketName, "video/"+filename, minio.RemoveObjectOptions{})
	return err
}

func acceptVideo(filename string) error {
	_, err := minioClient.CopyObject(context.Background(), minio.CopyDestOptions{
		Bucket: bucketName,
		Object: "video/" + filename,
	}, minio.CopySrcOptions{
		Bucket: bucketName,
		Object: "video/pending/" + filename,
	})
	if err != nil {
		return err
	}
	err = minioClient.RemoveObject(context.Background(), bucketName, "video/pending/"+filename, minio.RemoveObjectOptions{})
	return err
}

func getVideoRoutes() {
	r.GET("/video/:filename", authMiddleware(), func(c *gin.Context) {
		if r == nil {
			log.Panic("router is nil")
		}
		if minioClient == nil {
			log.Panic("minio client is nil")
		}
		filename := c.Param("filename")
		reader, err := minioClient.GetObject(context.Background(), bucketName, "video/"+filename, minio.GetObjectOptions{})
		if err != nil {
			c.String(500, "Failed to read video: %v", err)
			return
		}
		defer reader.Close()
		info, err := reader.Stat()
		if err != nil {
			c.String(500, "Failed to get video info: %v", err)
			return
		}
		c.DataFromReader(
			200,
			info.Size,
			info.ContentType,
			reader,
			map[string]string{
				"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, info.Key),
			},
		)
	})
	r.GET("/video/pending/:filename", authMiddleware(), func(c *gin.Context) {
			if r == nil {
				log.Panic("router is nil")
			}
			if minioClient == nil {
				log.Panic("minio client is nil")
			}
			filename := c.Param("filename")
			reader, err := minioClient.GetObject(context.Background(), bucketName, "video/pending/"+filename, minio.GetObjectOptions{})
			if err != nil {
				c.String(500, "Failed to read video: %v", err)
				return
			}
			defer reader.Close()
			info, err := reader.Stat()
			if err != nil {
				c.String(500, "Failed to get video info: %v", err)
				return
			}
			c.DataFromReader(
				200,
				info.Size,
				info.ContentType,
				reader,
				map[string]string{
					"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, info.Key),
				},
			)
		})
}
