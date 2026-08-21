package platform

import (
	"context"
	"testing"
)

func TestLocalNotifierPersistsOrderedDeliveries(t *testing.T) {
	inbox := NewLocalNotifier(t.TempDir())
	for _, recipient := range []string{"reviewer-2", "reviewer-1"} {
		if err := inbox.Send(context.Background(), Notification{Recipient: recipient, Template: "campaign-ready", Values: map[string]string{"campaign": "summer"}}); err != nil {
			t.Fatal(err)
		}
	}
	values, err := inbox.Sent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].Sequence != 1 || values[1].Sequence != 2 {
		t.Fatalf("unexpected persisted order: %#v", values)
	}
	values[0].Values["campaign"] = "changed"
	reloaded, err := inbox.Sent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded[0].Values["campaign"] != "summer" {
		t.Fatal("returned values must not mutate the persisted notification")
	}
}
