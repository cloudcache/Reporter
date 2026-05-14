package recordingstorage

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"
)

type S3Store struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Prefix    string
}

func (s S3Store) Save(ctx context.Context, req Request) (Result, error) {
	if s.Bucket == "" {
		return Result{}, fmt.Errorf("s3 bucket is required")
	}
	client, err := newMinioClient(s.Endpoint, s.AccessKey, s.SecretKey, s.UseSSL)
	if err != nil {
		return Result{}, err
	}
	objectName := BuildObjectName(req.CallID, req.OriginalName)
	if s.Prefix != "" {
		objectName = strings.Trim(strings.TrimSpace(s.Prefix), "/") + "/" + objectName
	}
	info, err := client.PutObject(ctx, s.Bucket, objectName, req.Reader, req.Size, minio.PutObjectOptions{
		ContentType: req.MimeType,
	})
	if err != nil {
		return Result{}, err
	}
	return Result{
		URI:        "s3://" + s.Bucket + "/" + objectName,
		Filename:   req.OriginalName,
		MimeType:   req.MimeType,
		SizeBytes:  info.Size,
		Backend:    "s3",
		ObjectName: objectName,
	}, nil
}
