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
	plan, err := s.prepareUpload(filename, mediaType, size, actor, requestID)
	if err != nil {
		return "", err
	}
	key, err := s.persistUpload(body, plan)
	if err != nil {
		return "", err
	}
	if err := s.recordUpload(key, plan); err != nil {
		_ = s.Storage.Delete(context.Background(), key)
		return "", err
	}
	return key, nil
}

type attachmentUploadPlan struct {
	Name      string
	MediaType string
	Size      int64
	Actor     domain.Actor
	RequestID string
}

func (s AttachmentService) prepareUpload(filename, mediaType string, size int64, actor domain.Actor, requestID string) (attachmentUploadPlan, error) {
	if err := require(actor, "campaign.edit"); err != nil {
		return attachmentUploadPlan{}, err
	}
	if err := s.validateUploadSize(size); err != nil {
		return attachmentUploadPlan{}, err
	}
	mediaType = strings.TrimSpace(mediaType)
	if err := s.validateUploadType(mediaType); err != nil {
		return attachmentUploadPlan{}, err
	}
	name, err := normalizeAttachmentName(filename)
	if err != nil {
		return attachmentUploadPlan{}, err
	}
	return attachmentUploadPlan{Name: name, MediaType: mediaType, Size: size, Actor: actor, RequestID: requestID}, nil
}

func (s AttachmentService) validateUploadSize(size int64) error {
	if size < 1 || size > s.MaxBytes {
		return fmt.Errorf("attachment size out of range")
	}
	return nil
}

func (s AttachmentService) validateUploadType(mediaType string) error {
	if !s.AllowedTypes[mediaType] {
		return fmt.Errorf("attachment type is not allowed")
	}
	return nil
}

func normalizeAttachmentName(filename string) (string, error) {
	name := filepath.Base(strings.TrimSpace(filename))
	if name == "." || name == "" || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid filename")
	}
	return name, nil
}

func (s AttachmentService) persistUpload(body io.Reader, plan attachmentUploadPlan) (string, error) {
	reader := io.LimitReader(body, s.MaxBytes+1)
	return s.Storage.Save(context.Background(), plan.Name, reader, plan.Size)
}

func (s AttachmentService) recordUpload(key string, plan attachmentUploadPlan) error {
	return s.Audit.Record(context.Background(), AuditChange{
		Actor: plan.Actor, Action: "attachment.uploaded", Subject: "attachment", SubjectID: domain.ID(key), RequestID: plan.RequestID,
		Metadata: map[string]any{"filename": plan.Name, "media_type": plan.MediaType, "size": plan.Size},
	})
}
