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
	"set_target":                "set_target",
	"set_full_target":           "set_full_target",
	"set_head_joints":           "set_head_joints",
	"set_body_yaw":              "set_body_yaw",
	"set_antennas":              "set_antennas",
	"goto":                      "goto_target",
	"wake_up":                   "wake_up",
	"sleep":                     "goto_sleep",
	"set_motor_mode":            "set_motor_mode",
	"set_torque":                "set_torque",
	"set_volume":                "set_volume",
	"get_volume":                "get_volume",
	"play_sound":                "play_sound",
	"get_state":                 "get_state",
	"get_version":               "get_version",
	"start_recording":           "start_recording",
	"stop_recording":            "stop_recording",
	"set_gravity_compensation":  "set_gravity_compensation",
	"set_automatic_body_yaw":    "set_automatic_body_yaw",
	"append_record":             "append_record",
	"set_mic_volume":            "set_microphone_volume",
	"get_mic_volume":            "get_microphone_volume",
	"get_motor_mode":            "get_motor_mode",
}

// restCommands are MQTT command topics that route through the REST API instead of WebSocket.
var restCommands = map[string]bool{
	// Emotions + dances (hardcoded Pollen datasets)
	"play_emotion":  true,
	"play_dance":    true,
	"list_emotions": true,
	"list_dances":   true,
	// Arbitrary dataset playback
	"play_dataset": true,
	"list_dataset": true,
	// Movement
	"goto_move":  true,
	"stop_move":  true,
	"list_moves": true,
	// Daemon lifecycle
	"daemon_start":       true,
	"daemon_stop":        true,
	"daemon_restart":     true,
	"daemon_lock_status": true,
	// Sound / media
	"list_sounds":    true,
	"delete_sound":   true,
	"stop_sound":     true,
	"test_sound":     true,
	"media_release":  true,
	"media_acquire":  true,
	"media_status":   true,
	// Camera
	"camera_specs": true,
	// Volume (REST variants)
	"get_volume_rest":     true,
	"get_mic_volume_rest": true,
	// Apps
	"app_list":          true,
	"app_start":         true,
	"app_stop":          true,
	"app_restart":       true,
	"app_status":        true,
	"app_install":       true,
	"app_remove":        true,
	"app_check_updates": true,
	"app_update":        true,
	"app_job_status":    true,
	// Kinematics
	"kinematics_info": true,
	"kinematics_urdf": true,
}

// IsRESTCommand returns true if the MQTT command should be routed through REST.
func IsRESTCommand(cmdTopic string) bool {
	return restCommands[cmdTopic]
}

// WSTypeForCommand returns the WebSocket message type for a given MQTT command topic suffix.
// If no explicit mapping exists, the command name is used as-is.
func WSTypeForCommand(cmdTopic string) string {
	if wsType, ok := mqttCmdToWSType[cmdTopic]; ok {
		return wsType
	}
	return cmdTopic
}

// EmotionRequest is the payload for play_emotion commands.
type EmotionRequest struct {
	Name string `json:"name"`
}

// DanceRequest is the payload for play_dance commands.
type DanceRequest struct {
	Name string `json:"name"`
}

// GotoMoveRequest is the payload for goto_move commands via REST.
type GotoMoveRequest struct {
	HeadPose      json.RawMessage `json:"head_pose,omitempty"`
	Antennas      []float64       `json:"antennas,omitempty"`
	BodyYaw       *float64        `json:"body_yaw,omitempty"`
	Duration      float64         `json:"duration"`
	Interpolation string          `json:"interpolation,omitempty"`
}

// AppRequest is the payload for app_start commands.
type AppRequest struct {
	Name string `json:"name"`
}

// MoveStopRequest is the payload for stop_move commands.
type MoveStopRequest struct {
	UUID string `json:"uuid"`
}

// SoundDeleteRequest is the payload for delete_sound commands.
type SoundDeleteRequest struct {
	Name string `json:"name"`
}

// DatasetPlayRequest is the payload for arbitrary dataset playback.
type DatasetPlayRequest struct {
	Dataset string `json:"dataset"`
	Name    string `json:"name"`
}

// AppInstallRequest is the payload for app_install commands.
type AppInstallRequest struct {
	SpaceID string `json:"space_id"`
}

// AppJobStatusRequest is the payload for app_job_status commands.
type AppJobStatusRequest struct {
	JobID string `json:"job_id"`
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
