// Package events provides an event system for chaos and recovery notifications.
package events

import "time"

// EventType represents the type of event
type EventType string

const (
	// EventChaosAttack is emitted when a chaos attack is executed
	EventChaosAttack EventType = "chaos_attack"
	// EventChaosResume is emitted when a suspended node is auto-resumed by chaos
	EventChaosResume EventType = "chaos_resume"
	// EventRecoveryStart is emitted when recovery attempts to restore a node
	EventRecoveryStart EventType = "recovery_start"
	// EventRecoverySuccess is emitted when recovery successfully restores a node
	EventRecoverySuccess EventType = "recovery_success"
	// EventRecoveryFailed is emitted when recovery fails to restore a node
	EventRecoveryFailed EventType = "recovery_failed"
)

// AttackType represents the type of chaos attack
type AttackType string

const (
	AttackTypeKill    AttackType = "kill"
	AttackTypeSuspend AttackType = "suspend"
	AttackTypeDelay   AttackType = "delay"
)

// Event represents a chaos or recovery event
type Event struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	NodeID    string    `json:"node_id"`
	Data      EventData `json:"data,omitempty"`
}

// EventData contains event-specific data
type EventData struct {
	AttackType    AttackType `json:"attack_type,omitempty"`
	DelayDuration string     `json:"delay_duration,omitempty"`
	Attempt       int        `json:"attempt,omitempty"`
	Error         string     `json:"error,omitempty"`
}

// NewChaosAttackEvent creates a new chaos attack event
func NewChaosAttackEvent(nodeID string, attackType AttackType) Event {
	return Event{
		Type:      EventChaosAttack,
		Timestamp: time.Now(),
		NodeID:    nodeID,
		Data: EventData{
			AttackType: attackType,
		},
	}
}

// NewChaosAttackEventWithDelay creates a chaos attack event for delay injection
func NewChaosAttackEventWithDelay(nodeID string, delay time.Duration) Event {
	return Event{
		Type:      EventChaosAttack,
		Timestamp: time.Now(),
		NodeID:    nodeID,
		Data: EventData{
			AttackType:    AttackTypeDelay,
			DelayDuration: delay.String(),
		},
	}
}

// NewChaosResumeEvent creates a chaos resume event
func NewChaosResumeEvent(nodeID string) Event {
	return Event{
		Type:      EventChaosResume,
		Timestamp: time.Now(),
		NodeID:    nodeID,
	}
}

// NewRecoveryStartEvent creates a recovery start event
func NewRecoveryStartEvent(nodeID string, attempt int) Event {
	return Event{
		Type:      EventRecoveryStart,
		Timestamp: time.Now(),
		NodeID:    nodeID,
		Data: EventData{
			Attempt: attempt,
		},
	}
}

// NewRecoverySuccessEvent creates a recovery success event
func NewRecoverySuccessEvent(nodeID string) Event {
	return Event{
		Type:      EventRecoverySuccess,
		Timestamp: time.Now(),
		NodeID:    nodeID,
	}
}

// NewRecoveryFailedEvent creates a recovery failed event
func NewRecoveryFailedEvent(nodeID string, err error) Event {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return Event{
		Type:      EventRecoveryFailed,
		Timestamp: time.Now(),
		NodeID:    nodeID,
		Data: EventData{
			Error: errMsg,
		},
	}
}
