package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/repository"
)

type Clock func() time.Time

func SystemClock() time.Time { return time.Now().UTC() }

type AuditWriter struct {
	Repository repository.AuditRepository
	Clock      Clock
}

type AuditChange struct {
	Actor     domain.Actor
	Action    string
	Subject   string
	SubjectID domain.ID
	RequestID string
	Metadata  any
}

func (w AuditWriter) Record(ctx context.Context, change AuditChange) error {
	if w.Repository == nil || w.Clock == nil {
		return fmt.Errorf("audit writer is not configured")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if change.Action == "" || change.Subject == "" || change.SubjectID.Empty() {
		return fmt.Errorf("audit change requires action, subject and subject id")
	}
	event := domain.NewAudit(change.Actor, change.Action, change.Subject, change.SubjectID, change.RequestID, change.Metadata, w.Clock())
	return w.Repository.AppendAudit(ctx, event)
}

func requestDigest(value any) (string, error) {
	digest := sha256.New()
	writeDigestNamespace(digest)
	encoder := json.NewEncoder(digest)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", fmt.Errorf("encode idempotency request: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func writeDigestNamespace(target hash.Hash) {
	_, _ = io.WriteString(target, "seasonal-publication/idempotency/v1\n")
}
func require(actor domain.Actor, permission string) error {
	if !actor.Can(permission) {
		return domain.ErrForbidden
	}
	return nil
}
