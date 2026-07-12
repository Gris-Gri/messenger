package ws

import (
	"testing"
)

func TestHub_BroadcastReadExcludesReader(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	reader := newClient(hub, nil, 1)
	other := newClient(hub, nil, 2)
	hub.Register(reader)
	hub.Register(other)

	payload := []byte(`{"type":"read","chat_id":10,"user_id":1,"last_read_message_id":5}`)
	hub.BroadcastRead(10, 1, payload, []int64{1, 2})

	select {
	case got := <-other.send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %s, want %s", got, payload)
		}
	default:
		t.Fatal("expected delivery to other participant")
	}

	select {
	case <-reader.send:
		t.Fatal("read event must not be delivered to the reader")
	default:
	}
}

func TestHub_BroadcastChatUpdatedExcludesActor(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	actor := newClient(hub, nil, 1)
	other := newClient(hub, nil, 2)
	hub.Register(actor)
	hub.Register(other)

	payload := []byte(`{"type":"chat_updated","chat_id":10,"title":"renamed"}`)
	hub.BroadcastChatUpdated(10, 1, payload, []int64{1, 2})

	select {
	case got := <-other.send:
		if string(got) != string(payload) {
			t.Fatalf("payload = %s, want %s", got, payload)
		}
	default:
		t.Fatal("expected delivery to other participant")
	}

	select {
	case <-actor.send:
		t.Fatal("chat_updated must not be delivered to the actor")
	default:
	}
}

func TestHub_RegisterUnregisterPresenceTransitions(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	first := newClient(hub, nil, 7)
	second := newClient(hub, nil, 7)

	if !hub.Register(first) {
		t.Fatal("first connection must become online")
	}
	if !hub.IsOnline(7) {
		t.Fatal("user must be online after first register")
	}
	if hub.Register(second) {
		t.Fatal("second connection must not re-signal online")
	}
	if hub.Unregister(first) {
		t.Fatal("unregister of non-last connection must not signal offline")
	}
	if !hub.IsOnline(7) {
		t.Fatal("user must stay online while second connection is alive")
	}
	if !hub.Unregister(second) {
		t.Fatal("last connection must signal offline")
	}
	if hub.IsOnline(7) {
		t.Fatal("user must be offline after last unregister")
	}
}
