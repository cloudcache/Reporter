package recordingstorage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"reporter/internal/domain"
)

type Request struct {
	CallID       string
	OriginalName string
	MimeType     string
	Size         int64
	Reader       io.Reader
}

type Result struct {
	URI        string
	Filename   string
	MimeType   string
	SizeBytes  int64
	Backend    string
	ObjectName string
}

type Store interface {
	Save(ctx context.Context, req Request) (Result, error)
}

func NewFromEndpoint(endpoint domain.SipEndpoint) Store {
	config := storageConfig(endpoint.Config)
	return newFromConfig(config)
}

func NewFromStorageConfig(config domain.StorageConfig) Store {
	settings := map[string]interface{}{}
	for key, value := range config.Config {
		settings[key] = value
	}
	putString(settings, "type", config.Kind)
	putString(settings, "endpoint", config.Endpoint)
	putString(settings, "bucket", config.Bucket)
	putString(settings, "basePath", config.BasePath)
	putString(settings, "baseUri", config.BaseURI)
	return newFromConfig(settings)
}

func newFromConfig(config map[string]interface{}) Store {
	switch strings.ToLower(configString(config, "type", "local")) {
	case "s3", "minio", "object":
		return S3Store{
			Endpoint:  configString(config, "endpoint", ""),
			Bucket:    configString(config, "bucket", ""),
			AccessKey: configString(config, "accessKey", ""),
			SecretKey: configString(config, "secretKey", ""),
			UseSSL:    configBool(config, "useSSL", true),
			Prefix:    configString(config, "prefix", "recordings"),
		}
	default:
		return LocalStore{
			BasePath: configString(config, "basePath", filepath.Join("data", "recordings")),
			BaseURI:  configString(config, "baseUri", ""),
		}
	}
}

func BuildObjectName(callID, originalName string) string {
	ext := filepath.Ext(originalName)
	if ext == "" {
		if detected, _ := mime.ExtensionsByType("audio/webm"); len(detected) > 0 {
			ext = detected[0]
		} else {
			ext = ".webm"
		}
	}
	now := time.Now().UTC()
	cleanCall := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(strings.TrimSpace(callID))
	if cleanCall == "" {
		cleanCall = "call"
	}
	return fmt.Sprintf("%04d/%02d/%02d/%s-%s%s", now.Year(), now.Month(), now.Day(), cleanCall, uuid.NewString(), ext)
}

func storageConfig(config map[string]interface{}) map[string]interface{} {
	if config == nil {
		return map[string]interface{}{"type": "local"}
	}
	if nested, ok := config["recordingStorage"].(map[string]interface{}); ok {
		return nested
	}
	return config
}

func configString(config map[string]interface{}, key, fallback string) string {
	value, ok := config[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback
		}
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func putString(config map[string]interface{}, key, value string) {
	if strings.TrimSpace(value) != "" {
		config[key] = value
	}
}

func configBool(config map[string]interface{}, key string, fallback bool) bool {
	value, ok := config[key]
	if !ok || value == nil {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return fallback
}

func newMinioClient(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("s3 endpoint, accessKey and secretKey are required")
	}
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
}
