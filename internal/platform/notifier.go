package platform

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Notification struct {
	Sequence  uint64
	Recipient string
	Template  string
	Values    map[string]string
	CreatedAt time.Time
}

type LocalNotifier struct {
	root     string
	writes   sync.Mutex
	sequence atomic.Uint64
}

func NewLocalNotifier(root string) *LocalNotifier {
	return &LocalNotifier{root: filepath.Clean(root)}
}

func (n *LocalNotifier) Send(ctx context.Context, value Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	value.Recipient = strings.TrimSpace(value.Recipient)
	value.Template = strings.TrimSpace(value.Template)
	if value.Recipient == "" || value.Template == "" {
		return fmt.Errorf("local notification requires recipient and template")
	}
	if n.root == "." || n.root == "" {
		return fmt.Errorf("local notification root is required")
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	} else {
		value.CreatedAt = value.CreatedAt.UTC()
	}
	n.writes.Lock()
	defer n.writes.Unlock()
	if err := os.MkdirAll(n.root, 0o750); err != nil {
		return fmt.Errorf("create notification inbox: %w", err)
	}
	var target string
	for {
		value.Sequence = n.sequence.Add(1)
		target = filepath.Join(n.root, fmt.Sprintf("%020d.notice", value.Sequence))
		if _, err := os.Stat(target); os.IsNotExist(err) {
			break
		} else if err != nil {
			return fmt.Errorf("inspect notification sequence: %w", err)
		}
	}
	temporary, err := os.CreateTemp(n.root, "pending-*.notice")
	if err != nil {
		return fmt.Errorf("create notification draft: %w", err)
	}
	temporaryName := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryName)
		}
	}()
	if err := gob.NewEncoder(temporary).Encode(value); err != nil {
		return fmt.Errorf("encode notification draft: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync notification draft: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close notification draft: %w", err)
	}
	if err := os.Rename(temporaryName, target); err != nil {
		return fmt.Errorf("commit notification: %w", err)
	}
	committed = true
	return nil
}

func (n *LocalNotifier) Sent(ctx context.Context) ([]Notification, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(n.root)
	if os.IsNotExist(err) {
		return []Notification{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read notification inbox: %w", err)
	}
	sort.Slice(entries, func(left, right int) bool { return entries[left].Name() < entries[right].Name() })
	values := make([]Notification, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".notice" || strings.HasPrefix(entry.Name(), "pending-") {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		file, err := os.Open(filepath.Join(n.root, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read notification %s: %w", entry.Name(), err)
		}
		var value Notification
		if err := gob.NewDecoder(file).Decode(&value); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("decode notification %s: %w", entry.Name(), err)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("close notification %s: %w", entry.Name(), err)
		}
		values = append(values, value)
	}
	return values, nil
}
