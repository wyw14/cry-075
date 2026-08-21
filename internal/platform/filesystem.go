package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LocalFileStorage struct{ Root string }

func (s LocalFileStorage) Save(ctx context.Context, name string, source io.Reader, expected int64) (string, error) {
	plan, err := s.prepareFileWrite(name, expected)
	if err != nil {
		return "", err
	}
	return s.commitFileWrite(context.Background(), plan, source)
}

type localFileWritePlan struct {
	root      string
	extension string
	expected  int64
}

func (s LocalFileStorage) prepareFileWrite(name string, expected int64) (localFileWritePlan, error) {
	root := filepath.Clean(s.Root)
	if root == "." || root == "" {
		return localFileWritePlan{}, fmt.Errorf("attachment root is required")
	}
	if expected < 0 {
		return localFileWritePlan{}, fmt.Errorf("attachment length is invalid")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return localFileWritePlan{}, fmt.Errorf("create attachment root: %w", err)
	}
	return localFileWritePlan{root: root, extension: filepath.Ext(name), expected: expected}, nil
}

func (s LocalFileStorage) commitFileWrite(ctx context.Context, plan localFileWritePlan, source io.Reader) (string, error) {
	temporary, err := os.CreateTemp(plan.root, "upload-*")
	if err != nil {
		return "", err
	}
	tempName := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(tempName)
		}
	}()
	digest, err := writeAttachmentPayload(ctx, temporary, source, plan.expected)
	if err != nil {
		return "", err
	}
	if err := temporary.Sync(); err != nil {
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	key := digest + plan.extension
	target := filepath.Join(plan.root, key)
	if err := os.Rename(tempName, target); err != nil {
		if !os.IsExist(err) {
			return "", err
		}
	}
	committed = true
	return key, nil
}

func writeAttachmentPayload(ctx context.Context, target io.Writer, source io.Reader, expected int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(target, hash), source)
	if err != nil {
		return "", err
	}
	if written != expected {
		return "", fmt.Errorf("attachment length mismatch")
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func (s LocalFileStorage) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Remove(filepath.Join(s.Root, filepath.Base(key)))
}
