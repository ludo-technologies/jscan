package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ludo-technologies/jscan/domain"
)

func TestCloneServiceDetectClones_AllFilesFailReturnsError(t *testing.T) {
	svc := NewCloneServiceWithDefaults()
	req := domain.DefaultCloneRequest()
	req.Paths = []string{"missing.js"}

	resp, err := svc.DetectClones(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when all clone inputs fail")
	}
	if resp == nil {
		t.Fatal("expected response even when analysis fails")
	}
	if resp.Success {
		t.Fatal("expected Success=false when all files fail")
	}
	if !strings.Contains(resp.Error, "missing.js") {
		t.Fatalf("expected response error to mention failing file, got: %q", resp.Error)
	}
}

func TestCloneServiceDetectClones_PartialFailureReturnsResponseAndError(t *testing.T) {
	svc := NewCloneServiceWithDefaults()
	req := domain.DefaultCloneRequest()

	tempDir := t.TempDir()
	validFile := filepath.Join(tempDir, "valid.js")
	content := []byte(`function alpha(value) {
  if (value > 10) {
    return value + 1;
  }
  return value - 1;
}
`)
	if err := os.WriteFile(validFile, content, 0o644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	req.Paths = []string{validFile, filepath.Join(tempDir, "missing.js")}

	resp, err := svc.DetectClones(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when one input file fails")
	}
	if resp == nil {
		t.Fatal("expected response for partial failure")
	}
	if resp.Success {
		t.Fatal("expected Success=false when any input file fails")
	}
	if resp.Statistics == nil || resp.Statistics.FilesAnalyzed != 1 {
		t.Fatalf("expected one successfully analyzed file, got %+v", resp.Statistics)
	}
	if !strings.Contains(resp.Error, "missing.js") {
		t.Fatalf("expected response error to mention failing file, got: %q", resp.Error)
	}
}
