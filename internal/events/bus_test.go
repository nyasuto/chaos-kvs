package events

import (
	"testing"
	"time"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	if bus == nil {
		t.Fatal("expected non-nil bus")
	}
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}
}

func TestBusSubscribe(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe()
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	ch2 := bus.Subscribe()
	if bus.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount())
	}

	if ch1 == nil || ch2 == nil {
		t.Error("expected non-nil channels")
	}
}

func TestBusUnsubscribe(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe()
	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	bus.Unsubscribe(ch)
	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}
}

func TestBusPublish(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe()

	event := NewChaosAttackEvent("node-1", AttackTypeKill)
	bus.Publish(event)

	select {
	case received := <-ch:
		if received.Type != EventChaosAttack {
			t.Errorf("expected type %s, got %s", EventChaosAttack, received.Type)
		}
		if received.NodeID != "node-1" {
			t.Errorf("expected node-1, got %s", received.NodeID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for event")
	}
}

func TestBusPublishMultipleSubscribers(t *testing.T) {
	bus := NewBus()

	ch1 := bus.Subscribe()
	ch2 := bus.Subscribe()

	event := NewChaosAttackEvent("node-1", AttackTypeSuspend)
	bus.Publish(event)

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case received := <-ch:
			if received.Type != EventChaosAttack {
				t.Errorf("subscriber %d: expected type %s, got %s", i, EventChaosAttack, received.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: timeout waiting for event", i)
		}
	}
}

func TestBusPublishNonBlocking(t *testing.T) {
	bus := NewBus()
	bus.bufferSize = 1 // Small buffer for testing

	ch := bus.Subscribe()

	// Fill the buffer
	bus.Publish(NewChaosAttackEvent("node-1", AttackTypeKill))
	bus.Publish(NewChaosAttackEvent("node-2", AttackTypeKill))
	bus.Publish(NewChaosAttackEvent("node-3", AttackTypeKill))

	// Should not block - test passes if it completes
	// First event should be received
	select {
	case <-ch:
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for first event")
	}
}

func TestBusClose(t *testing.T) {
	bus := NewBus()

	ch := bus.Subscribe()
	bus.Close()

	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after close, got %d", bus.SubscriberCount())
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed")
	}
}

func TestEventCreation(t *testing.T) {
	t.Run("ChaosAttackEvent", func(t *testing.T) {
		event := NewChaosAttackEvent("node-1", AttackTypeKill)
		if event.Type != EventChaosAttack {
			t.Errorf("expected %s, got %s", EventChaosAttack, event.Type)
		}
		if event.NodeID != "node-1" {
			t.Errorf("expected node-1, got %s", event.NodeID)
		}
		if event.Data.AttackType != AttackTypeKill {
			t.Errorf("expected kill, got %s", event.Data.AttackType)
		}
	})

	t.Run("ChaosAttackEventWithDelay", func(t *testing.T) {
		event := NewChaosAttackEventWithDelay("node-2", 100*time.Millisecond)
		if event.Data.AttackType != AttackTypeDelay {
			t.Errorf("expected delay, got %s", event.Data.AttackType)
		}
		if event.Data.DelayDuration != "100ms" {
			t.Errorf("expected 100ms, got %s", event.Data.DelayDuration)
		}
	})

	t.Run("RecoveryEvents", func(t *testing.T) {
		start := NewRecoveryStartEvent("node-1", 1)
		if start.Type != EventRecoveryStart {
			t.Errorf("expected %s, got %s", EventRecoveryStart, start.Type)
		}
		if start.Data.Attempt != 1 {
			t.Errorf("expected attempt 1, got %d", start.Data.Attempt)
		}

		success := NewRecoverySuccessEvent("node-1")
		if success.Type != EventRecoverySuccess {
			t.Errorf("expected %s, got %s", EventRecoverySuccess, success.Type)
		}
	})
}
