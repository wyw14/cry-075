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
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(s.Root, 0o750); err != nil {
		return "", fmt.Errorf("create attachment root: %w", err)
	}
	temporary, err := os.CreateTemp(s.Root, "upload-*")
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
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(temporary, hash), source)
	if err != nil {
		return "", err
	}
	if written != expected {
		return "", fmt.Errorf("attachment length mismatch")
	}
	if err := temporary.Sync(); err != nil {
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	key := hex.EncodeToString(hash.Sum(nil)) + filepath.Ext(name)
	target := filepath.Join(s.Root, key)
	if err := os.Rename(tempName, target); err != nil {
		if !os.IsExist(err) {
			return "", err
		}
	}
	committed = true
	return key, nil
}
func (s LocalFileStorage) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return os.Remove(filepath.Join(s.Root, filepath.Base(key)))
}
