package storage

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var Client *minio.Client
var bucketName string

func Init() {
	var err error

	endpoint := getEnv("MINIO_ENDPOINT", "minio:9000")
	accessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	secretKey := getEnv("MINIO_SECRET_KEY", "minioadmin")
	useSSL := getEnvBool("MINIO_USE_SSL", false)
	bucketName = getEnv("MINIO_BUCKET", "deployments")

	Client, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Fatalln("failed to connect to MinIO:", err)
	}
	if err := ensureBucket(context.Background(), bucketName); err != nil {
		log.Fatalln("failed to ensure MinIO bucket:", err)
	}

	log.Println("connected to minio")
}

func UploadFolder(deploymentID string) error {
	basePath := "/tmp/" + deploymentID

	return filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		objectName := deploymentID + "/" + path[len(basePath)+1:]

		_, err = Client.FPutObject(context.Background(), bucketName, objectName, path, minio.PutObjectOptions{})
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

func BucketName() string {
	return bucketName
}

func ensureBucket(ctx context.Context, name string) error {
	exists, err := Client.BucketExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return Client.MakeBucket(ctx, name, minio.MakeBucketOptions{})
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
