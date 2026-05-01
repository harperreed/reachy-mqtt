// ABOUTME: Central router that connects WebSocket, MQTT, and REST components.
// ABOUTME: Routes telemetry from WS to MQTT, commands from MQTT to WS, and API requests through REST.
package main

import (
	"context"
	"log"
)

type Router struct {
	ws   *WSClient
	mqtt *MQTTBridge
	rest *RESTClient
}

func NewRouter(ws *WSClient, mqtt *MQTTBridge, rest *RESTClient) *Router {
	return &Router{
		ws:   ws,
		mqtt: mqtt,
		rest: rest,
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
// Parses the type field and publishes the raw JSON to the corresponding topic.
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

	// Publish raw JSON to reachy/{robot}/{type}
	r.mqtt.Publish(msgType, data)
}

// handleMQTTCommand routes an MQTT command to the WebSocket connection.
// Maps the MQTT topic suffix to the WS message type and injects it into the payload.
func (r *Router) handleMQTTCommand(ctx context.Context, cmd MQTTCommand) {
	wsType := WSTypeForCommand(cmd.CommandType)

	// Inject the "type" field into the JSON payload
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

// handleAPIRequest forwards an MQTT API request through the REST client.
func (r *Router) handleAPIRequest(ctx context.Context, reqData []byte) {
	resp := r.rest.HandleAPIRequest(ctx, reqData)
	r.mqtt.PublishAPIResponse(resp)
}
