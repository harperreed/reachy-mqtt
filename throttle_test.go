// ABOUTME: Tests for the per-topic publish throttle with change detection.
// ABOUTME: Verifies rate limiting, deduplication, and heartbeat behavior.
package main

import (
	"testing"
	"time"
)

func TestThrottleFirstMessageAlwaysPublishes(t *testing.T) {
	throttle := NewTopicThrottle(100*time.Millisecond, 5*time.Second)

	if !throttle.ShouldPublish("test", []byte(`{"value":1}`)) {
		t.Error("first message should always publish")
	}
}

func TestThrottleDuplicateSuppressed(t *testing.T) {
	throttle := NewTopicThrottle(10*time.Millisecond, 5*time.Second)

	payload := []byte(`{"type":"joint_positions","head":[0,0,0,0,0,0,0]}`)

	if !throttle.ShouldPublish("joints", payload) {
		t.Error("first message should publish")
	}

	// Wait past rate limit
	time.Sleep(15 * time.Millisecond)

	// Same payload — should be suppressed
	if throttle.ShouldPublish("joints", payload) {
		t.Error("duplicate payload should be suppressed")
	}
}

func TestThrottleChangedPayloadPublishes(t *testing.T) {
	throttle := NewTopicThrottle(10*time.Millisecond, 5*time.Second)

	if !throttle.ShouldPublish("joints", []byte(`{"v":1}`)) {
		t.Error("first message should publish")
	}

	time.Sleep(15 * time.Millisecond)

	// Different payload — should publish
	if !throttle.ShouldPublish("joints", []byte(`{"v":2}`)) {
		t.Error("changed payload should publish")
	}
}

func TestThrottleRateLimits(t *testing.T) {
	throttle := NewTopicThrottle(50*time.Millisecond, 5*time.Second)

	throttle.ShouldPublish("fast", []byte(`{"v":1}`))

	// Immediately send different payload — should be rate limited
	if throttle.ShouldPublish("fast", []byte(`{"v":2}`)) {
		t.Error("should be rate limited (too fast)")
	}

	// Wait past rate limit
	time.Sleep(55 * time.Millisecond)

	if !throttle.ShouldPublish("fast", []byte(`{"v":2}`)) {
		t.Error("should publish after rate limit expires")
	}
}

func TestThrottleHeartbeat(t *testing.T) {
	throttle := NewTopicThrottle(10*time.Millisecond, 50*time.Millisecond)

	payload := []byte(`{"v":"static"}`)

	if !throttle.ShouldPublish("hb", payload) {
		t.Error("first message should publish")
	}

	time.Sleep(15 * time.Millisecond)

	// Same payload, within heartbeat — suppressed
	if throttle.ShouldPublish("hb", payload) {
		t.Error("duplicate within heartbeat should be suppressed")
	}

	// Wait past heartbeat interval
	time.Sleep(50 * time.Millisecond)

	// Same payload, but heartbeat expired — should publish
	if !throttle.ShouldPublish("hb", payload) {
		t.Error("should publish as heartbeat even with same payload")
	}
}

func TestThrottleIndependentTopics(t *testing.T) {
	throttle := NewTopicThrottle(10*time.Millisecond, 5*time.Second)

	if !throttle.ShouldPublish("a", []byte(`{"topic":"a"}`)) {
		t.Error("topic a first message should publish")
	}
	if !throttle.ShouldPublish("b", []byte(`{"topic":"b"}`)) {
		t.Error("topic b first message should publish (independent of a)")
	}
}
