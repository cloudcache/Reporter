package recordingstorage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStore struct {
	BasePath string
	BaseURI  string
}

func (s LocalStore) Save(_ context.Context, req Request) (Result, error) {
	basePath := s.BasePath
	if basePath == "" {
		basePath = filepath.Join("data", "recordings")
	}
	objectName := BuildObjectName(req.CallID, req.OriginalName)
	path := filepath.Join(basePath, filepath.FromSlash(objectName))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Result{}, err
	}
	dst, err := os.Create(path)
	if err != nil {
		return Result{}, err
	}
	size, copyErr := io.Copy(dst, req.Reader)
	closeErr := dst.Close()
	if copyErr != nil {
		return Result{}, copyErr
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	uri := "file://" + path
	if s.BaseURI != "" {
		uri = strings.TrimRight(s.BaseURI, "/") + "/" + objectName
	}
	return Result{
		URI:        uri,
		Filename:   req.OriginalName,
		MimeType:   req.MimeType,
		SizeBytes:  size,
		Backend:    "local",
		ObjectName: objectName,
	}, nil
}
