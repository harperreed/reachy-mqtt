// ABOUTME: Message type definitions for the Reachy Mini WebSocket protocol.
// ABOUTME: Envelope for routing plus command-type mapping between MQTT topics and WS types.
package main

import "encoding/json"

// WSEnvelope extracts just the type field from a raw WebSocket JSON message.
type WSEnvelope struct {
	Type string `json:"type"`
}

// ParseWSType extracts the message type from raw JSON without full deserialization.
func ParseWSType(data []byte) (string, error) {
	var env WSEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return "", err
	}
	return env.Type, nil
}

// InjectType sets or overrides the "type" field in a raw JSON payload.
// Returns the modified JSON. If payload is empty or invalid, wraps with just the type.
func InjectType(payload []byte, msgType string) ([]byte, error) {
	var m map[string]json.RawMessage
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &m); err != nil {
			// Payload isn't valid JSON object — create one with just the type
			m = make(map[string]json.RawMessage)
		}
	} else {
		m = make(map[string]json.RawMessage)
	}

	typeBytes, err := json.Marshal(msgType)
	if err != nil {
		return nil, err
	}
	m["type"] = typeBytes

	return json.Marshal(m)
}

// mqttCmdToWSType maps MQTT command topic suffixes to WebSocket message type values.
// Most map 1:1 but a few have different names.
var mqttCmdToWSType = map[string]string{
	"set_target":      "set_target",
	"set_full_target": "set_full_target",
	"set_head_joints": "set_head_joints",
	"set_body_yaw":    "set_body_yaw",
	"set_antennas":    "set_antennas",
	"goto":            "goto_target",
	"wake_up":         "wake_up",
	"sleep":           "goto_sleep",
	"set_motor_mode":  "set_motor_mode",
	"set_torque":      "set_torque",
	"set_volume":      "set_volume",
	"get_volume":      "get_volume",
	"play_sound":      "play_sound",
	"get_state":       "get_state",
	"get_version":     "get_version",
	"start_recording": "start_recording",
	"stop_recording":  "stop_recording",
}

// WSTypeForCommand returns the WebSocket message type for a given MQTT command topic suffix.
// If no explicit mapping exists, the command name is used as-is.
func WSTypeForCommand(cmdTopic string) string {
	if wsType, ok := mqttCmdToWSType[cmdTopic]; ok {
		return wsType
	}
	return cmdTopic
}

// APIRequest is the JSON structure for MQTT API bridge requests.
type APIRequest struct {
	Endpoint string          `json:"endpoint"`
	Method   string          `json:"method"`
	Body     json.RawMessage `json:"body,omitempty"`
}

// APIResponse is the JSON structure for MQTT API bridge responses.
type APIResponse struct {
	Endpoint   string          `json:"endpoint"`
	StatusCode int             `json:"status_code"`
	Data       json.RawMessage `json:"data"`
	Error      string          `json:"error,omitempty"`
}
