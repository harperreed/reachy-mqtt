// ABOUTME: Per-topic publish throttle with change detection and heartbeat.
// ABOUTME: Suppresses duplicate messages and rate-limits MQTT publishes per signal type.
package main

import (
	"bytes"
	"sync"
	"time"
)

// TopicThrottle tracks per-topic state for change detection and rate limiting.
type TopicThrottle struct {
	mu               sync.Mutex
	lastPayload      map[string][]byte
	lastPublishTime  map[string]time.Time
	minInterval      time.Duration
	heartbeatInterval time.Duration
}

func NewTopicThrottle(minInterval, heartbeatInterval time.Duration) *TopicThrottle {
	return &TopicThrottle{
		lastPayload:       make(map[string][]byte),
		lastPublishTime:   make(map[string]time.Time),
		minInterval:       minInterval,
		heartbeatInterval: heartbeatInterval,
	}
}

// ShouldPublish returns true if the message should be published to MQTT.
// A message is published when:
//   - The payload has changed since last publish, AND enough time has passed (rate limit)
//   - OR the heartbeat interval has elapsed (even if payload is identical)
func (t *TopicThrottle) ShouldPublish(topic string, payload []byte) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	lastTime := t.lastPublishTime[topic]
	lastPayload := t.lastPayload[topic]

	// Heartbeat: always publish if we haven't sent anything for this topic in a while
	if now.Sub(lastTime) >= t.heartbeatInterval {
		t.lastPayload[topic] = copyBytes(payload)
		t.lastPublishTime[topic] = now
		return true
	}

	// Rate limit: don't publish faster than minInterval
	if now.Sub(lastTime) < t.minInterval {
		return false
	}

	// Change detection: only publish if payload differs
	if bytes.Equal(payload, lastPayload) {
		return false
	}

	t.lastPayload[topic] = copyBytes(payload)
	t.lastPublishTime[topic] = now
	return true
}

func copyBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
