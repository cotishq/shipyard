package storage

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cotishq/shipyard/internal/config"
	"github.com/cotishq/shipyard/internal/observability"
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
	maxAttempts := getEnvInt("MINIO_INIT_MAX_ATTEMPTS", 20)
	retryDelay := time.Duration(getEnvInt("MINIO_INIT_RETRY_DELAY_SECONDS", 2)) * time.Second

	if err := config.ValidateMinIOCredentials(accessKey, secretKey); err != nil && !config.AllowInsecureDefaults() {
		log.Fatalln(err)
	}

	Client, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})

	if err != nil {
		log.Fatalln("failed to connect to MinIO:", err)
	}

	var lastErr error
	for i := 1; i <= maxAttempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		lastErr = ensureBucket(ctx, bucketName)
		cancel()
		if lastErr == nil {
			observability.Info("connected to minio", map[string]any{
				"bucket": bucketName,
			})
			return
		}

		observability.Info("waiting for minio", map[string]any{
			"attempt":      i,
			"max_attempts": maxAttempts,
			"error":        lastErr.Error(),
		})
		if i < maxAttempts {
			time.Sleep(retryDelay)
		}
	}

	log.Fatalln("failed to ensure MinIO bucket:", lastErr)
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

		observability.Info("uploaded artifact", map[string]any{
			"object_name": objectName,
		})
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

func HealthCheck(ctx context.Context) error {
	if Client == nil {
		return errors.New("minio client is not initialized")
	}
	_, err := Client.BucketExists(ctx, bucketName)
	return err
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

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
