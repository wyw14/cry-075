package domain

import (
	"encoding/json"
	"strings"
	"time"
)

type AuditEvent struct {
	ID        ID              `json:"id"`
	ActorID   ID              `json:"actor_id"`
	ActorRole Role            `json:"actor_role"`
	Action    string          `json:"action"`
	Subject   string          `json:"subject"`
	SubjectID ID              `json:"subject_id"`
	RequestID string          `json:"request_id"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time       `json:"created_at"`
}

func NewAudit(actor Actor, action, subject string, subjectID ID, requestID string, metadata any, now time.Time) AuditEvent {
	payload, _ := json.Marshal(metadata)
	return AuditEvent{ID: NewID("aud"), ActorID: actor.ID, ActorRole: actor.Role, Action: action, Subject: subject, SubjectID: subjectID, RequestID: requestID, Metadata: RedactJSON(payload), CreatedAt: now.UTC()}
}

func RedactJSON(payload []byte) []byte {
	var value any
	if json.Unmarshal(payload, &value) != nil {
		return []byte(`{}`)
	}
	redactValue(value)
	clean, _ := json.Marshal(value)
	return clean
}

func redactValue(value any) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}
	for key, current := range object {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
			object[key] = "[REDACTED]"
			continue
		}
		redactValue(current)
	}
}

type ExportRequest struct {
	ID          ID        `json:"id"`
	RequestedBy ID        `json:"requested_by"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	Reason      string    `json:"reason"`
	Fields      []string  `json:"fields"`
}
