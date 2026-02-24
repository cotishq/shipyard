package storage

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)


var Client *minio.Client

func Init() {
	var err error

	Client, err = minio.New("localhost:9000", &minio.Options{
		Creds: credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})

	if err != nil {
		log.Fatalln("failed to connect to MinIO:", err)
	}

	log.Println("connected to minio")
}

func UploadFolder(deploymentID string) error {
	bucket := "deployments"
	basePath := "/tmp/" + deploymentID

	return filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		objectName := deploymentID + "/" + path[len(basePath)+1:]

		_, err = Client.FPutObject(context.Background(), bucket, objectName, path, minio.PutObjectOptions{})
		if err != nil {
			return err
		}

		log.Println("Uploaded:", objectName)
		return nil
	})
}

func GetOptions() minio.GetObjectOptions {
	return minio.GetObjectOptions{}
}