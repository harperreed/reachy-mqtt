// ABOUTME: Central router that connects WebSocket, MQTT, and REST components.
// ABOUTME: Routes telemetry from WS to MQTT, commands from MQTT to WS or REST, and API requests through REST.
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

type Router struct {
	ws       *WSClient
	mqtt     *MQTTBridge
	rest     *RESTClient
	throttle *TopicThrottle
}

func NewRouter(ws *WSClient, mqtt *MQTTBridge, rest *RESTClient, minInterval, heartbeat time.Duration) *Router {
	return &Router{
		ws:       ws,
		mqtt:     mqtt,
		rest:     rest,
		throttle: NewTopicThrottle(minInterval, heartbeat),
	}
}

// Run is the main routing loop. It reads from all channels and routes messages
// between WebSocket, MQTT, and REST. Blocks until ctx is cancelled.
func (r *Router) Run(ctx context.Context) {
	log.Printf("[router] started")

	for {
		select {
		case <-ctx.Done():
			log.Printf("[router] stopped")
			return

		case data := <-r.ws.Messages():
			r.handleWSMessage(ctx, data)

		case cmd := <-r.mqtt.Commands():
			r.handleMQTTCommand(ctx, cmd)

		case reqData := <-r.mqtt.APIRequests():
			r.handleAPIRequest(ctx, reqData)
		}
	}
}

// handleWSMessage routes an incoming WebSocket telemetry message to MQTT.
// Applies change detection and rate limiting before publishing.
func (r *Router) handleWSMessage(_ context.Context, data []byte) {
	msgType, err := ParseWSType(data)
	if err != nil {
		log.Printf("[router] parse ws message type: %v", err)
		return
	}
	if msgType == "" {
		log.Printf("[router] ws message missing type field")
		return
	}

	if !r.throttle.ShouldPublish(msgType, data) {
		return
	}

	r.mqtt.Publish(msgType, data)
}

// handleMQTTCommand routes an MQTT command to either WebSocket or REST.
func (r *Router) handleMQTTCommand(ctx context.Context, cmd MQTTCommand) {
	if IsRESTCommand(cmd.CommandType) {
		r.handleRESTCommand(ctx, cmd)
		return
	}
	r.handleWSCommand(ctx, cmd)
}

// handleWSCommand routes an MQTT command to the WebSocket connection.
func (r *Router) handleWSCommand(ctx context.Context, cmd MQTTCommand) {
	wsType := WSTypeForCommand(cmd.CommandType)

	data, err := InjectType(cmd.Payload, wsType)
	if err != nil {
		log.Printf("[router] build ws command %s: %v", cmd.CommandType, err)
		return
	}

	if err := r.ws.Send(ctx, data); err != nil {
		log.Printf("[router] send ws command %s: %v", cmd.CommandType, err)
		return
	}

	log.Printf("[router] forwarded cmd %s → ws type %s", cmd.CommandType, wsType)
}

// handleRESTCommand routes an MQTT command through the REST API and publishes the response.
func (r *Router) handleRESTCommand(ctx context.Context, cmd MQTTCommand) {
	var data []byte
	var err error

	switch cmd.CommandType {
	case "play_emotion":
		var req EmotionRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.Name == "" {
			log.Printf("[router] play_emotion: invalid payload: %v", err)
			return
		}
		data, err = r.rest.PlayEmotion(ctx, req.Name)

	case "play_dance":
		var req DanceRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.Name == "" {
			log.Printf("[router] play_dance: invalid payload: %v", err)
			return
		}
		data, err = r.rest.PlayDance(ctx, req.Name)

	case "list_emotions":
		data, err = r.rest.ListEmotions(ctx)

	case "list_dances":
		data, err = r.rest.ListDances(ctx)

	case "goto_move":
		data, err = r.rest.GotoMove(ctx, cmd.Payload)

	case "stop_move":
		var req MoveStopRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil {
			log.Printf("[router] stop_move: invalid payload: %v", err)
			return
		}
		data, err = r.rest.StopMove(ctx, req.UUID)

	case "list_moves":
		data, err = r.rest.ListRunningMoves(ctx)

	case "daemon_start":
		data, err = r.rest.DaemonStart(ctx, true)

	case "daemon_stop":
		data, err = r.rest.DaemonStop(ctx, true)

	case "daemon_restart":
		data, err = r.rest.DaemonRestart(ctx)

	case "camera_specs":
		data, err = r.rest.GetCameraSpecs(ctx)

	case "app_list":
		data, err = r.rest.ListApps(ctx)

	case "app_start":
		var req AppRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.Name == "" {
			log.Printf("[router] app_start: invalid payload: %v", err)
			return
		}
		data, err = r.rest.StartApp(ctx, req.Name)

	case "app_stop":
		data, err = r.rest.StopApp(ctx)

	case "app_restart":
		data, err = r.rest.RestartApp(ctx)

	case "app_status":
		data, err = r.rest.GetCurrentAppStatus(ctx)

	case "app_install":
		var req AppInstallRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.SpaceID == "" {
			log.Printf("[router] app_install: invalid payload: %v", err)
			return
		}
		data, err = r.rest.InstallApp(ctx, req.SpaceID)

	case "app_remove":
		var req AppRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.Name == "" {
			log.Printf("[router] app_remove: invalid payload: %v", err)
			return
		}
		data, err = r.rest.RemoveApp(ctx, req.Name)

	case "app_check_updates":
		data, err = r.rest.CheckAppUpdates(ctx)

	case "app_update":
		var req AppRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.Name == "" {
			log.Printf("[router] app_update: invalid payload: %v", err)
			return
		}
		data, err = r.rest.UpdateApp(ctx, req.Name)

	case "app_job_status":
		var req AppJobStatusRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.JobID == "" {
			log.Printf("[router] app_job_status: invalid payload: %v", err)
			return
		}
		data, err = r.rest.GetAppJobStatus(ctx, req.JobID)

	case "daemon_lock_status":
		data, err = r.rest.GetDaemonLockStatus(ctx)

	case "list_sounds":
		data, err = r.rest.ListSounds(ctx)

	case "delete_sound":
		var req SoundDeleteRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.Name == "" {
			log.Printf("[router] delete_sound: invalid payload: %v", err)
			return
		}
		data, err = r.rest.DeleteSound(ctx, req.Name)

	case "stop_sound":
		data, err = r.rest.StopSound(ctx)

	case "test_sound":
		data, err = r.rest.TestSound(ctx)

	case "media_release":
		data, err = r.rest.MediaRelease(ctx)

	case "media_acquire":
		data, err = r.rest.MediaAcquire(ctx)

	case "media_status":
		data, err = r.rest.MediaStatus(ctx)

	case "get_volume_rest":
		data, err = r.rest.GetVolumeREST(ctx)

	case "get_mic_volume_rest":
		data, err = r.rest.GetMicVolumeREST(ctx)

	case "kinematics_info":
		data, err = r.rest.GetKinematicsInfo(ctx)

	case "kinematics_urdf":
		data, err = r.rest.GetKinematicsURDF(ctx)

	case "play_dataset":
		var req DatasetPlayRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.Dataset == "" || req.Name == "" {
			log.Printf("[router] play_dataset: invalid payload: %v", err)
			return
		}
		data, err = r.rest.PlayDataset(ctx, req.Dataset, req.Name)

	case "list_dataset":
		var req DatasetPlayRequest
		if err := json.Unmarshal(cmd.Payload, &req); err != nil || req.Dataset == "" {
			log.Printf("[router] list_dataset: invalid payload: %v", err)
			return
		}
		data, err = r.rest.ListDataset(ctx, req.Dataset)

	default:
		log.Printf("[router] unknown REST command: %s", cmd.CommandType)
		return
	}

	if err != nil {
		log.Printf("[router] REST cmd %s error: %v", cmd.CommandType, err)
		resp := APIResponse{
			Endpoint: cmd.CommandType,
			Error:    err.Error(),
		}
		respData, _ := json.Marshal(resp)
		r.mqtt.PublishAPIResponse(respData)
		return
	}

	// Publish response to the command-specific response topic
	r.mqtt.Publish("cmd/"+cmd.CommandType+"/response", data)
	log.Printf("[router] REST cmd %s completed", cmd.CommandType)
}

// handleAPIRequest forwards an MQTT API request through the REST client.
func (r *Router) handleAPIRequest(ctx context.Context, reqData []byte) {
	resp := r.rest.HandleAPIRequest(ctx, reqData)
	r.mqtt.PublishAPIResponse(resp)
}
