// ABOUTME: REST API client for the Reachy Mini's HTTP endpoints.
// ABOUTME: Provides on-demand queries and periodic state polling published to MQTT.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type RESTClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewRESTClient(apiURL string) *RESTClient {
	return &RESTClient{
		baseURL: apiURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Do executes an HTTP request and returns the raw response body.
func (r *RESTClient) Do(ctx context.Context, method, endpoint string, body []byte) ([]byte, int, error) {
	url := r.baseURL + endpoint

	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return data, resp.StatusCode, nil
}

// GetFullState fetches the complete robot state from the REST API.
func (r *RESTClient) GetFullState(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/state/full", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/state/full returned %d", status)
	}
	return data, nil
}

// GetDaemonStatus fetches the daemon status from the REST API.
func (r *RESTClient) GetDaemonStatus(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/daemon/status", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/daemon/status returned %d", status)
	}
	return data, nil
}

// GetMotorStatus fetches the motor status from the REST API.
func (r *RESTClient) GetMotorStatus(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/motors/status", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/motors/status returned %d", status)
	}
	return data, nil
}

// RunPoller periodically fetches state from the REST API and publishes to MQTT.
// Blocks until ctx is cancelled.
func (r *RESTClient) RunPoller(ctx context.Context, interval time.Duration, mqtt *MQTTBridge) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[rest] polling every %v", interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.pollAndPublish(ctx, mqtt)
		}
	}
}

func (r *RESTClient) pollAndPublish(ctx context.Context, mqttBridge *MQTTBridge) {
	// Poll full state
	if data, err := r.GetFullState(ctx); err == nil {
		mqttBridge.Publish("state", data)
	} else {
		log.Printf("[rest] poll state error: %v", err)
	}

	// Poll motor status
	if data, err := r.GetMotorStatus(ctx); err == nil {
		mqttBridge.Publish("motors", data)
	} else {
		log.Printf("[rest] poll motors error: %v", err)
	}

	// Poll Direction of Arrival
	if data, err := r.GetDoA(ctx); err == nil {
		mqttBridge.Publish("doa", data)
	} else {
		log.Printf("[rest] poll doa error: %v", err)
	}

	// Poll current app status
	if data, err := r.GetCurrentAppStatus(ctx); err == nil {
		mqttBridge.Publish("app_status", data)
	} else {
		log.Printf("[rest] poll app status error: %v", err)
	}

	// Poll speaker volume
	if data, err := r.GetVolumeREST(ctx); err == nil {
		mqttBridge.Publish("volume", data)
	} else {
		log.Printf("[rest] poll volume error: %v", err)
	}

	// Poll microphone volume
	if data, err := r.GetMicVolumeREST(ctx); err == nil {
		mqttBridge.Publish("mic_volume", data)
	} else {
		log.Printf("[rest] poll mic volume error: %v", err)
	}

	// Poll media status
	if data, err := r.MediaStatus(ctx); err == nil {
		mqttBridge.Publish("media_status", data)
	} else {
		log.Printf("[rest] poll media status error: %v", err)
	}
}

// GetDoA fetches the Direction of Arrival from the microphone array.
func (r *RESTClient) GetDoA(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/state/doa", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/state/doa returned %d", status)
	}
	return data, nil
}

// GetCameraSpecs fetches camera intrinsics for CV pipelines.
func (r *RESTClient) GetCameraSpecs(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/camera/specs", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/camera/specs returned %d", status)
	}
	return data, nil
}

// PlayEmotion triggers a pre-built emotion animation by name.
func (r *RESTClient) PlayEmotion(ctx context.Context, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/move/play/recorded-move-dataset/pollen-robotics/reachy-mini-emotions-library/%s", name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d: %s", endpoint, status, string(data))
	}
	return data, nil
}

// PlayDance triggers a pre-built dance animation by name.
func (r *RESTClient) PlayDance(ctx context.Context, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/move/play/recorded-move-dataset/pollen-robotics/reachy-mini-dances-library/%s", name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d: %s", endpoint, status, string(data))
	}
	return data, nil
}

// ListEmotions lists available emotion animations.
func (r *RESTClient) ListEmotions(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/move/recorded-move-datasets/list/pollen-robotics/reachy-mini-emotions-library", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list emotions returned %d", status)
	}
	return data, nil
}

// ListDances lists available dance animations.
func (r *RESTClient) ListDances(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/move/recorded-move-datasets/list/pollen-robotics/reachy-mini-dances-library", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("list dances returned %d", status)
	}
	return data, nil
}

// GotoMove sends a smooth interpolated movement via REST.
func (r *RESTClient) GotoMove(ctx context.Context, body []byte) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/move/goto", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/move/goto returned %d: %s", status, string(data))
	}
	return data, nil
}

// StopMove stops a running move by UUID.
func (r *RESTClient) StopMove(ctx context.Context, uuid string) ([]byte, error) {
	body, _ := json.Marshal(map[string]string{"uuid": uuid})
	data, status, err := r.Do(ctx, http.MethodPost, "/api/move/stop", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/move/stop returned %d", status)
	}
	return data, nil
}

// ListRunningMoves lists currently running move tasks.
func (r *RESTClient) ListRunningMoves(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/move/running", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/move/running returned %d", status)
	}
	return data, nil
}

// DaemonStart starts the robot daemon.
func (r *RESTClient) DaemonStart(ctx context.Context, wakeUp bool) ([]byte, error) {
	body, _ := json.Marshal(map[string]bool{"wake_up": wakeUp})
	data, status, err := r.Do(ctx, http.MethodPost, "/api/daemon/start", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/daemon/start returned %d", status)
	}
	return data, nil
}

// DaemonStop stops the robot daemon.
func (r *RESTClient) DaemonStop(ctx context.Context, gotoSleep bool) ([]byte, error) {
	body, _ := json.Marshal(map[string]bool{"goto_sleep": gotoSleep})
	data, status, err := r.Do(ctx, http.MethodPost, "/api/daemon/stop", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/daemon/stop returned %d", status)
	}
	return data, nil
}

// DaemonRestart restarts the robot daemon.
func (r *RESTClient) DaemonRestart(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/daemon/restart", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/daemon/restart returned %d", status)
	}
	return data, nil
}

// GetCurrentAppStatus gets the status of the currently running app.
func (r *RESTClient) GetCurrentAppStatus(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/apps/current-app-status", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/apps/current-app-status returned %d", status)
	}
	return data, nil
}

// ListApps lists all available apps.
func (r *RESTClient) ListApps(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/apps/list-available", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/apps/list-available returned %d", status)
	}
	return data, nil
}

// StartApp starts an app by name.
func (r *RESTClient) StartApp(ctx context.Context, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/apps/start-app/%s", name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d", endpoint, status)
	}
	return data, nil
}

// StopApp stops the currently running app.
func (r *RESTClient) StopApp(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/apps/stop-current-app", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/apps/stop-current-app returned %d", status)
	}
	return data, nil
}

// RestartApp restarts the currently running app.
func (r *RESTClient) RestartApp(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/apps/restart-current-app", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/apps/restart-current-app returned %d", status)
	}
	return data, nil
}

// InstallApp installs an app from a HuggingFace space.
func (r *RESTClient) InstallApp(ctx context.Context, spaceID string) ([]byte, error) {
	body, _ := json.Marshal(map[string]string{"space_id": spaceID})
	data, status, err := r.Do(ctx, http.MethodPost, "/api/apps/install", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/apps/install returned %d", status)
	}
	return data, nil
}

// RemoveApp removes an installed app by name.
func (r *RESTClient) RemoveApp(ctx context.Context, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/apps/remove/%s", name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d", endpoint, status)
	}
	return data, nil
}

// CheckAppUpdates checks for available app updates.
func (r *RESTClient) CheckAppUpdates(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/apps/check-updates", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/apps/check-updates returned %d", status)
	}
	return data, nil
}

// UpdateApp updates an app to its latest version.
func (r *RESTClient) UpdateApp(ctx context.Context, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/apps/update/%s", name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d", endpoint, status)
	}
	return data, nil
}

// GetAppJobStatus gets the status and logs of an install/update job.
func (r *RESTClient) GetAppJobStatus(ctx context.Context, jobID string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/apps/job-status/%s", jobID)
	data, status, err := r.Do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %d", endpoint, status)
	}
	return data, nil
}

// GetDaemonLockStatus checks the robot-app lock status.
func (r *RESTClient) GetDaemonLockStatus(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/daemon/robot-app-lock-status", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/daemon/robot-app-lock-status returned %d", status)
	}
	return data, nil
}

// ListSounds lists available sound files on the robot.
func (r *RESTClient) ListSounds(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/media/sounds", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/media/sounds returned %d", status)
	}
	return data, nil
}

// DeleteSound deletes a sound file from the robot.
func (r *RESTClient) DeleteSound(ctx context.Context, filename string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/media/sounds/%s", filename)
	data, status, err := r.Do(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("DELETE %s returned %d", endpoint, status)
	}
	return data, nil
}

// StopSound stops the currently playing sound.
func (r *RESTClient) StopSound(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/media/stop_sound", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/media/stop_sound returned %d", status)
	}
	return data, nil
}

// TestSound plays a test sound on the robot.
func (r *RESTClient) TestSound(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/volume/test-sound", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/volume/test-sound returned %d", status)
	}
	return data, nil
}

// MediaRelease releases camera/audio hardware for external use.
func (r *RESTClient) MediaRelease(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/media/release", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/media/release returned %d", status)
	}
	return data, nil
}

// MediaAcquire re-acquires camera/audio hardware.
func (r *RESTClient) MediaAcquire(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodPost, "/api/media/acquire", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST /api/media/acquire returned %d", status)
	}
	return data, nil
}

// MediaStatus gets the current media backend status.
func (r *RESTClient) MediaStatus(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/media/status", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/media/status returned %d", status)
	}
	return data, nil
}

// GetVolumeREST fetches the current speaker volume via REST.
func (r *RESTClient) GetVolumeREST(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/volume/current", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/volume/current returned %d", status)
	}
	return data, nil
}

// GetMicVolumeREST fetches the current microphone volume via REST.
func (r *RESTClient) GetMicVolumeREST(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/volume/microphone/current", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/volume/microphone/current returned %d", status)
	}
	return data, nil
}

// GetKinematicsInfo fetches kinematics information.
func (r *RESTClient) GetKinematicsInfo(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/kinematics/info", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/kinematics/info returned %d", status)
	}
	return data, nil
}

// GetKinematicsURDF fetches the URDF model.
func (r *RESTClient) GetKinematicsURDF(ctx context.Context) ([]byte, error) {
	data, status, err := r.Do(ctx, http.MethodGet, "/api/kinematics/urdf", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET /api/kinematics/urdf returned %d", status)
	}
	return data, nil
}

// PlayDataset plays an arbitrary recorded-move dataset entry.
func (r *RESTClient) PlayDataset(ctx context.Context, dataset, name string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/move/play/recorded-move-dataset/%s/%s", dataset, name)
	data, status, err := r.Do(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("POST %s returned %d: %s", endpoint, status, string(data))
	}
	return data, nil
}

// ListDataset lists entries in an arbitrary recorded-move dataset.
func (r *RESTClient) ListDataset(ctx context.Context, dataset string) ([]byte, error) {
	endpoint := fmt.Sprintf("/api/move/recorded-move-datasets/list/%s", dataset)
	data, status, err := r.Do(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %d", endpoint, status)
	}
	return data, nil
}

// HandleAPIRequest processes an MQTT API bridge request and returns the response.
func (r *RESTClient) HandleAPIRequest(ctx context.Context, reqData []byte) []byte {
	var req APIRequest
	if err := json.Unmarshal(reqData, &req); err != nil {
		resp := APIResponse{Error: fmt.Sprintf("invalid request: %v", err)}
		data, _ := json.Marshal(resp)
		return data
	}

	if req.Method == "" {
		req.Method = http.MethodGet
	}

	data, status, err := r.Do(ctx, req.Method, req.Endpoint, req.Body)

	var resp APIResponse
	resp.Endpoint = req.Endpoint
	resp.StatusCode = status

	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Data = data
	}

	result, _ := json.Marshal(resp)
	return result
}
