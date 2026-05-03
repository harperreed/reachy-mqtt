// ABOUTME: Unit tests for message parsing, type injection, and command mapping.
// ABOUTME: Covers the pure logic in messages.go without needing network connections.
package main

import (
	"encoding/json"
	"testing"
)

func TestParseWSType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "joint_positions",
			input: `{"type":"joint_positions","head_joint_positions":[0,0,0,0,0,0,0]}`,
			want:  "joint_positions",
		},
		{
			name:  "head_pose",
			input: `{"type":"head_pose","head_pose":[[1,0,0,0],[0,1,0,0],[0,0,1,0],[0,0,0,1]]}`,
			want:  "head_pose",
		},
		{
			name:  "imu_data",
			input: `{"type":"imu_data","accelerometer":[0,0,9.8],"gyroscope":[0,0,0],"quaternion":[1,0,0,0],"temperature":25.0}`,
			want:  "imu_data",
		},
		{
			name:  "daemon_status",
			input: `{"type":"daemon_status","robot_name":"reachy","state":"running"}`,
			want:  "daemon_status",
		},
		{
			name:    "invalid json",
			input:   `not json`,
			wantErr: true,
		},
		{
			name:  "missing type returns empty",
			input: `{"data":"hello"}`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseWSType([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseWSType() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("ParseWSType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInjectType(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		msgType  string
		wantType string
		wantErr  bool
	}{
		{
			name:     "inject into existing object",
			payload:  `{"head_pose":[1,0,0,0,0,1,0,0,0,0,1,0,0,0,0,1]}`,
			msgType:  "set_target",
			wantType: "set_target",
		},
		{
			name:     "override existing type",
			payload:  `{"type":"wrong","data":123}`,
			msgType:  "correct_type",
			wantType: "correct_type",
		},
		{
			name:     "empty payload",
			payload:  "",
			msgType:  "wake_up",
			wantType: "wake_up",
		},
		{
			name:     "empty json object",
			payload:  `{}`,
			msgType:  "goto_sleep",
			wantType: "goto_sleep",
		},
		{
			name:     "invalid json creates new object",
			payload:  `not json`,
			msgType:  "wake_up",
			wantType: "wake_up",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InjectType([]byte(tt.payload), tt.msgType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("InjectType() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			var m map[string]json.RawMessage
			if err := json.Unmarshal(result, &m); err != nil {
				t.Fatalf("InjectType() returned invalid JSON: %v", err)
			}

			var gotType string
			if err := json.Unmarshal(m["type"], &gotType); err != nil {
				t.Fatalf("InjectType() type field not a string: %v", err)
			}

			if gotType != tt.wantType {
				t.Errorf("InjectType() type = %q, want %q", gotType, tt.wantType)
			}
		})
	}

	// Verify InjectType preserves other fields
	t.Run("preserves fields", func(t *testing.T) {
		payload := `{"head_pose":[1,2,3],"duration":0.5}`
		result, err := InjectType([]byte(payload), "goto_target")
		if err != nil {
			t.Fatal(err)
		}

		var m map[string]json.RawMessage
		json.Unmarshal(result, &m)

		if _, ok := m["head_pose"]; !ok {
			t.Error("InjectType() lost head_pose field")
		}
		if _, ok := m["duration"]; !ok {
			t.Error("InjectType() lost duration field")
		}
		if _, ok := m["type"]; !ok {
			t.Error("InjectType() missing type field")
		}
	})
}

func TestWSTypeForCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"set_target", "set_target"},
		{"set_head_joints", "set_head_joints"},
		{"set_body_yaw", "set_body_yaw"},
		{"set_antennas", "set_antennas"},
		{"goto", "goto_target"},
		{"wake_up", "wake_up"},
		{"sleep", "goto_sleep"},
		{"set_motor_mode", "set_motor_mode"},
		{"set_volume", "set_volume"},
		{"play_sound", "play_sound"},
		{"set_gravity_compensation", "set_gravity_compensation"},
		{"set_automatic_body_yaw", "set_automatic_body_yaw"},
		{"set_mic_volume", "set_microphone_volume"},
		{"get_mic_volume", "get_microphone_volume"},
		{"get_motor_mode", "get_motor_mode"},
		{"unknown_cmd", "unknown_cmd"}, // passthrough for unknown commands
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := WSTypeForCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("WSTypeForCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}
