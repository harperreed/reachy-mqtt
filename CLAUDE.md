# Reachy MQTT Bridge

## Names
- Claude: **BIGFOOT BYTE CRUSHER**
- Harp-Dog: **HARPZILLA CHAOS SUPREME**

## What is this?

A Go daemon that bridges a Reachy Mini robot's WebSocket + REST APIs to MQTT.
Full bidirectional: telemetry publishes to MQTT, MQTT commands forward to the robot.

## Architecture

- **WebSocket client** → `ws://{host}:8000/ws/sdk` for 50Hz telemetry + commands
- **REST client** → `http://{host}:8000/api/*` for state polling + on-demand queries
- **MQTT bridge** → publishes telemetry, subscribes to command topics
- **Router** → glues WS/REST/MQTT together, passes raw JSON through where possible

## MQTT Topics

### Telemetry (published by daemon)
- `reachy/{robot}/joint_positions` — head (7 DOF) + antenna (2 DOF) angles
- `reachy/{robot}/head_pose` — 4x4 homogeneous transform matrix
- `reachy/{robot}/imu_data` — accelerometer, gyroscope, quaternion, temperature
- `reachy/{robot}/daemon_status` — robot state, connectivity
- `reachy/{robot}/task_progress` — async task completion tracking
- `reachy/{robot}/state` — full state (REST-polled)
- `reachy/{robot}/motors` — motor status (REST-polled)
- `reachy/{robot}/doa` — Direction of Arrival (mic array angle + speech detection)
- `reachy/{robot}/app_status` — current app state

### WS Commands (routed through WebSocket)
- `reachy/{robot}/cmd/set_target` — head pose as flat 4x4 matrix
- `reachy/{robot}/cmd/set_full_target` — combined head/antennas/body_yaw
- `reachy/{robot}/cmd/set_head_joints` — 7 joint values (radians)
- `reachy/{robot}/cmd/set_body_yaw` — single float (radians)
- `reachy/{robot}/cmd/set_antennas` — [right, left] radians
- `reachy/{robot}/cmd/goto` — smooth move with duration + interpolation
- `reachy/{robot}/cmd/wake_up` / `cmd/sleep` — state transitions
- `reachy/{robot}/cmd/set_motor_mode` — enabled/disabled/gravity_compensation
- `reachy/{robot}/cmd/set_gravity_compensation` — compliant mode toggle
- `reachy/{robot}/cmd/set_automatic_body_yaw` — auto body following
- `reachy/{robot}/cmd/set_torque` — torque on/off
- `reachy/{robot}/cmd/set_volume` / `cmd/set_mic_volume` — audio levels (0-100)
- `reachy/{robot}/cmd/play_sound` — play WAV file on robot
- `reachy/{robot}/cmd/start_recording` / `cmd/stop_recording` — motion capture

### REST Commands (routed through HTTP API)
- `reachy/{robot}/cmd/play_emotion` — `{"name":"Happy"}` (81 emotions)
- `reachy/{robot}/cmd/play_dance` — `{"name":"Groovy Sway"}` (19 dances)
- `reachy/{robot}/cmd/list_emotions` / `cmd/list_dances` — list available
- `reachy/{robot}/cmd/goto_move` — smooth REST-based move with duration
- `reachy/{robot}/cmd/stop_move` / `cmd/list_moves` — move task management
- `reachy/{robot}/cmd/daemon_start` / `daemon_stop` / `daemon_restart`
- `reachy/{robot}/cmd/camera_specs` — camera intrinsics for CV
- `reachy/{robot}/cmd/app_list` / `app_start` / `app_stop` / `app_status`

### Generic REST Bridge
- `reachy/{robot}/api/request` + `api/response` — any REST endpoint

## Config

JSON config file at `--config path`, overridden by `.env`, overridden by real env vars.

## Building

```bash
go build -o reachy-mqtt .
```

## Running

```bash
./reachy-mqtt --config config.json
# or just use env vars / .env file
REACHY_HOST=192.168.1.100 ./reachy-mqtt
```

## Dependencies

- `nhooyr.io/websocket` — WebSocket client
- `github.com/eclipse/paho.mqtt.golang` — MQTT client
- `github.com/joho/godotenv` — .env file loading
