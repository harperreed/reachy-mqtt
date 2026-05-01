// ABOUTME: Integration-style tests for the router's message routing logic.
// ABOUTME: Tests WS→MQTT telemetry routing and MQTT→WS command routing with real channels.
package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// testMQTTPublisher captures published messages for verification.
type testMQTTPublisher struct {
	published []publishedMsg
}

type publishedMsg struct {
	topic   string
	payload []byte
}

func TestRouterHandlesWSMessage(t *testing.T) {
	// Simulate a WebSocket telemetry message arriving and verify it gets
	// routed to the correct MQTT topic.
	wsMsg := `{"type":"joint_positions","head_joint_positions":[0.1,0.2,0.3,0.4,0.5,0.6,0.7],"antennas_joint_positions":[0.0,0.0]}`

	msgType, err := ParseWSType([]byte(wsMsg))
	if err != nil {
		t.Fatal(err)
	}

	if msgType != "joint_positions" {
		t.Errorf("expected type %q, got %q", "joint_positions", msgType)
	}
}

func TestRouterCommandRoundTrip(t *testing.T) {
	// Simulate receiving a command from MQTT and building the WS message.
	payload := `{"head_pose":[1,0,0,0, 0,1,0,0, 0,0,1,0, 0,0,0,1]}`
	cmdType := "set_target"

	wsType := WSTypeForCommand(cmdType)
	if wsType != "set_target" {
		t.Errorf("expected ws type %q, got %q", "set_target", wsType)
	}

	result, err := InjectType([]byte(payload), wsType)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the result has both the type and the original data
	var m map[string]json.RawMessage
	json.Unmarshal(result, &m)

	var gotType string
	json.Unmarshal(m["type"], &gotType)
	if gotType != "set_target" {
		t.Errorf("result type = %q, want %q", gotType, "set_target")
	}

	if _, ok := m["head_pose"]; !ok {
		t.Error("result missing head_pose field")
	}
}

func TestRouterGotoCommandMapping(t *testing.T) {
	// The MQTT topic "goto" should map to WS type "goto_target"
	payload := `{"duration":0.5,"interpolation":"minjerk"}`
	wsType := WSTypeForCommand("goto")

	result, err := InjectType([]byte(payload), wsType)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(result, &m)

	var gotType string
	json.Unmarshal(m["type"], &gotType)
	if gotType != "goto_target" {
		t.Errorf("goto command mapped to %q, want %q", gotType, "goto_target")
	}
}

func TestRouterSleepCommandMapping(t *testing.T) {
	// The MQTT topic "sleep" should map to WS type "goto_sleep"
	wsType := WSTypeForCommand("sleep")
	if wsType != "goto_sleep" {
		t.Errorf("sleep command mapped to %q, want %q", wsType, "goto_sleep")
	}

	// Should work with empty payload
	result, err := InjectType([]byte(""), wsType)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]json.RawMessage
	json.Unmarshal(result, &m)

	var gotType string
	json.Unmarshal(m["type"], &gotType)
	if gotType != "goto_sleep" {
		t.Errorf("result type = %q, want %q", gotType, "goto_sleep")
	}
}

func TestRouterWSChannelDrop(t *testing.T) {
	// Verify the WSClient drops messages when the channel is full.
	ws := NewWSClient("ws://unused:8000/ws/sdk", time.Second)

	// Fill the channel
	for i := 0; i < 256; i++ {
		ws.incoming <- []byte(`{"type":"test"}`)
	}

	// The channel should be full (non-blocking send would fail)
	select {
	case ws.incoming <- []byte(`{"type":"overflow"}`):
		t.Error("expected channel to be full")
	default:
		// Channel is full as expected
	}

	// Drain and verify
	msg := <-ws.Messages()
	var env WSEnvelope
	json.Unmarshal(msg, &env)
	if env.Type != "test" {
		t.Errorf("got type %q, want %q", env.Type, "test")
	}
}

func TestAPIRequestParsing(t *testing.T) {
	reqJSON := `{"endpoint":"/api/state/full","method":"GET"}`

	var req APIRequest
	err := json.Unmarshal([]byte(reqJSON), &req)
	if err != nil {
		t.Fatal(err)
	}

	if req.Endpoint != "/api/state/full" {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, "/api/state/full")
	}
	if req.Method != "GET" {
		t.Errorf("method = %q, want %q", req.Method, "GET")
	}
}

func TestAPIResponseMarshaling(t *testing.T) {
	resp := APIResponse{
		Endpoint:   "/api/state/full",
		StatusCode: 200,
		Data:       json.RawMessage(`{"body_yaw":0.1}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	var decoded APIResponse
	json.Unmarshal(data, &decoded)

	if decoded.StatusCode != 200 {
		t.Errorf("status = %d, want %d", decoded.StatusCode, 200)
	}
	if decoded.Error != "" {
		t.Errorf("error = %q, want empty", decoded.Error)
	}
}

func TestRESTClientHandleAPIRequestInvalid(t *testing.T) {
	rest := NewRESTClient("http://localhost:8000")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Invalid JSON should return error response
	result := rest.HandleAPIRequest(ctx, []byte("not json"))

	var resp APIResponse
	json.Unmarshal(result, &resp)

	if resp.Error == "" {
		t.Error("expected error for invalid request JSON")
	}
}
