package application

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/wyw14/cry-075/internal/domain"
)

type FileStorage interface {
	Save(context.Context, string, io.Reader, int64) (string, error)
	Delete(context.Context, string) error
}
type AttachmentService struct {
	Storage      FileStorage
	MaxBytes     int64
	AllowedTypes map[string]bool
	Audit        AuditWriter
}

func (s AttachmentService) Upload(ctx context.Context, filename, mediaType string, size int64, body io.Reader, actor domain.Actor, requestID string) (string, error) {
	if err := require(actor, "campaign.edit"); err != nil {
		return "", err
	}
	if size < 1 || size > s.MaxBytes {
		return "", fmt.Errorf("attachment size out of range")
	}
	if !s.AllowedTypes[mediaType] {
		return "", fmt.Errorf("attachment type is not allowed")
	}
	name := filepath.Base(strings.TrimSpace(filename))
	if name == "." || name == "" || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid filename")
	}
	key, err := s.Storage.Save(ctx, name, io.LimitReader(body, s.MaxBytes+1), size)
	if err != nil {
		return "", err
	}
	if err := s.Audit.Record(ctx, AuditChange{Actor: actor, Action: "attachment.uploaded", Subject: "attachment", SubjectID: domain.ID(key), RequestID: requestID, Metadata: map[string]any{"filename": name, "media_type": mediaType, "size": size}}); err != nil {
		_ = s.Storage.Delete(ctx, key)
		return "", err
	}
	return key, nil
}
