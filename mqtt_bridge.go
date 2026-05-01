// ABOUTME: MQTT client wrapper for the reachy-mqtt bridge.
// ABOUTME: Publishes robot telemetry and subscribes to command + API request topics.
package main

import (
	"fmt"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTCommand represents a command received from MQTT destined for the robot.
type MQTTCommand struct {
	CommandType string
	Payload     []byte
}

type MQTTBridge struct {
	client    mqtt.Client
	robotName string
	commands  chan MQTTCommand
	apiReqs   chan []byte
}

func NewMQTTBridge(cfg Config) (*MQTTBridge, error) {
	bridge := &MQTTBridge{
		robotName: cfg.ReachyName,
		commands:  make(chan MQTTCommand, 64),
		apiReqs:   make(chan []byte, 16),
	}

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTBroker).
		SetClientID(cfg.MQTTClientID).
		SetAutoReconnect(true).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Printf("[mqtt] connection lost: %v", err)
		}).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Printf("[mqtt] connected to %s", cfg.MQTTBroker)
			bridge.subscribe(c)
		})

	if cfg.MQTTUsername != "" {
		opts.SetUsername(cfg.MQTTUsername)
		opts.SetPassword(cfg.MQTTPassword)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		return nil, fmt.Errorf("mqtt connect: %w", token.Error())
	}

	bridge.client = client
	return bridge, nil
}

func (b *MQTTBridge) subscribe(client mqtt.Client) {
	// Subscribe to command topics: reachy/{robot}/cmd/#
	cmdTopic := fmt.Sprintf("reachy/%s/cmd/#", b.robotName)
	token := client.Subscribe(cmdTopic, 1, b.handleCommand)
	token.Wait()
	if token.Error() != nil {
		log.Printf("[mqtt] subscribe error for %s: %v", cmdTopic, token.Error())
	} else {
		log.Printf("[mqtt] subscribed to %s", cmdTopic)
	}

	// Subscribe to API request topic: reachy/{robot}/api/request
	apiTopic := fmt.Sprintf("reachy/%s/api/request", b.robotName)
	token = client.Subscribe(apiTopic, 1, b.handleAPIRequest)
	token.Wait()
	if token.Error() != nil {
		log.Printf("[mqtt] subscribe error for %s: %v", apiTopic, token.Error())
	} else {
		log.Printf("[mqtt] subscribed to %s", apiTopic)
	}
}

func (b *MQTTBridge) handleCommand(_ mqtt.Client, msg mqtt.Message) {
	// Extract command type from topic: reachy/{robot}/cmd/{command_type}
	prefix := fmt.Sprintf("reachy/%s/cmd/", b.robotName)
	topic := msg.Topic()
	if !strings.HasPrefix(topic, prefix) {
		return
	}
	cmdType := topic[len(prefix):]
	if cmdType == "" {
		return
	}

	select {
	case b.commands <- MQTTCommand{CommandType: cmdType, Payload: msg.Payload()}:
	default:
		log.Printf("[mqtt] dropping command %s, channel full", cmdType)
	}
}

func (b *MQTTBridge) handleAPIRequest(_ mqtt.Client, msg mqtt.Message) {
	select {
	case b.apiReqs <- msg.Payload():
	default:
		log.Printf("[mqtt] dropping API request, channel full")
	}
}

// Publish sends telemetry data to the appropriate MQTT topic.
func (b *MQTTBridge) Publish(signalType string, payload []byte) {
	topic := fmt.Sprintf("reachy/%s/%s", b.robotName, signalType)
	token := b.client.Publish(topic, 0, false, payload)
	token.Wait()
}

// PublishAPIResponse sends an API response back via MQTT.
func (b *MQTTBridge) PublishAPIResponse(payload []byte) {
	topic := fmt.Sprintf("reachy/%s/api/response", b.robotName)
	token := b.client.Publish(topic, 0, false, payload)
	token.Wait()
}

// Commands returns a read-only channel of incoming robot commands.
func (b *MQTTBridge) Commands() <-chan MQTTCommand {
	return b.commands
}

// APIRequests returns a read-only channel of incoming API bridge requests.
func (b *MQTTBridge) APIRequests() <-chan []byte {
	return b.apiReqs
}

// Close disconnects the MQTT client.
func (b *MQTTBridge) Close() {
	b.client.Disconnect(1000)
	log.Printf("[mqtt] disconnected")
}
