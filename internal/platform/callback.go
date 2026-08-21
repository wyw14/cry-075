package platform

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type Callback struct {
	Topic      string          `json:"topic"`
	Payload    json.RawMessage `json:"payload"`
	OccurredAt time.Time       `json:"occurred_at"`
}
type LocalCallbackSink struct {
	mu        sync.Mutex
	callbacks []Callback
}

func NewLocalCallbackSink() *LocalCallbackSink { return &LocalCallbackSink{callbacks: []Callback{}} }
func (s *LocalCallbackSink) Publish(ctx context.Context, topic string, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callbacks = append(s.callbacks, Callback{Topic: topic, Payload: encoded, OccurredAt: time.Now().UTC()})
	return nil
}
func (s *LocalCallbackSink) Drain() []Callback {
	s.mu.Lock()
	defer s.mu.Unlock()
	values := append([]Callback(nil), s.callbacks...)
	s.callbacks = s.callbacks[:0]
	return values
}
