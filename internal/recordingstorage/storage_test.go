package recordingstorage

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"reporter/internal/domain"
)

func TestLocalStoreSave(t *testing.T) {
	dir := t.TempDir()
	store := LocalStore{BasePath: dir}
	result, err := store.Save(context.Background(), Request{
		CallID:       "CALL001",
		OriginalName: "call.webm",
		MimeType:     "audio/webm",
		Size:         4,
		Reader:       bytes.NewBufferString("data"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Backend != "local" || result.SizeBytes != 4 || !strings.HasPrefix(result.URI, "file://") {
		t.Fatalf("unexpected result: %+v", result)
	}
	path := strings.TrimPrefix(result.URI, "file://")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "data" {
		t.Fatal("recording content mismatch")
	}
}

func TestNewFromEndpointS3Config(t *testing.T) {
	store := NewFromEndpoint(domain.SipEndpoint{Config: map[string]interface{}{
		"recordingStorage": map[string]interface{}{
			"type":      "s3",
			"endpoint":  "minio.local:9000",
			"bucket":    "recordings",
			"accessKey": "key",
			"secretKey": "secret",
			"useSSL":    false,
		},
	}})
	if _, ok := store.(S3Store); !ok {
		t.Fatalf("expected S3Store, got %T", store)
	}
}
