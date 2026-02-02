package util

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	// Create a temp file to serve
	content := "Hello, World! This is a test file for download."
	tmpServerFile, err := os.CreateTemp("", "server_file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpServerFile.Name())
	tmpServerFile.WriteString(content)
	tmpServerFile.Close()

	// Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, tmpServerFile.Name())
	}))
	defer ts.Close()

	// Destination path
	destDir, err := os.MkdirTemp("", "download_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(destDir)
	destPath := filepath.Join(destDir, "downloaded.txt")

	// Test 1: Normal Download
	err = DownloadFile(context.Background(), ts.URL, destPath)
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}

	if !FileExists(destPath) {
		t.Fatal("File not found after download")
	}

	// Verify content
	downloadedData, _ := os.ReadFile(destPath)
	if string(downloadedData) != content {
		t.Errorf("Content mismatch. Got %s, want %s", string(downloadedData), content)
	}

	// Test 2: Resume Download (Simulate partial)
	os.Remove(destPath)
	partialContent := "Hello, World!"
	tmpPath := destPath + ".tmp"
	os.WriteFile(tmpPath, []byte(partialContent), 0644)

	err = DownloadFile(context.Background(), ts.URL, destPath)
	if err != nil {
		t.Fatalf("Resume Download failed: %v", err)
	}

	downloadedData, _ = os.ReadFile(destPath)
	if string(downloadedData) != content {
		t.Errorf("Content mismatch after resume. Got %s, want %s", string(downloadedData), content)
	}
}

func TestCalculateFileMD5(t *testing.T) {
	content := "test content"
	tmpFile, _ := os.CreateTemp("", "md5_test")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(content)
	tmpFile.Close()

	hash := md5.New()
	hash.Write([]byte(content))
	expected := hex.EncodeToString(hash.Sum(nil))

	fileHash, err := CalculateFileMD5(tmpFile.Name())
	if err != nil {
		t.Fatalf("CalculateFileMD5 failed: %v", err)
	}

	if fileHash != expected {
		t.Errorf("MD5 mismatch. Got %s, want %s", fileHash, expected)
	}
}
