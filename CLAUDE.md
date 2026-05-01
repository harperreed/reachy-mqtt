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

- `reachy/{robot}/{signal_type}` — telemetry (joint_positions, head_pose, imu, etc.)
- `reachy/{robot}/cmd/{command}` — commands (set_target, wake_up, sleep, etc.)
- `reachy/{robot}/api/request` + `api/response` — REST bridge

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
