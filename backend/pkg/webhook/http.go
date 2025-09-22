package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// HTTPWebhook implements webhook delivery over HTTP/HTTPS
type HTTPWebhook struct {
	config WebhookConfig
	client *http.Client
	logger *logrus.Logger
}

// NewHTTPWebhook creates a new HTTP webhook
func NewHTTPWebhook(config WebhookConfig, logger *logrus.Logger) *HTTPWebhook {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Create HTTP client with proper configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: !config.VerifySSL,
		},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &HTTPWebhook{
		config: config,
		client: client,
		logger: logger,
	}
}

// Send sends a webhook payload via HTTP
func (w *HTTPWebhook) Send(ctx context.Context, payload *Payload) error {
	startTime := time.Now()

	// Serialize payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", w.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Docker-Auto-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", payload.ID)
	req.Header.Set("X-Webhook-Event", payload.Event)
	req.Header.Set("X-Webhook-Timestamp", payload.Timestamp.Format(time.RFC3339))
	req.Header.Set("X-Webhook-Source", payload.Source)

	// Add custom headers
	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	// Add signature if secret is configured
	if w.config.Secret != "" {
		signature := w.generateSignature(jsonData, w.config.Secret)
		headerName := w.config.SignatureHeader
		if headerName == "" {
			headerName = "X-Hub-Signature-256"
		}
		req.Header.Set(headerName, "sha256="+signature)
	}

	// Add retry count header
	if payload.RetryCount > 0 {
		req.Header.Set("X-Webhook-Retry-Count", fmt.Sprintf("%d", payload.RetryCount))
	}

	w.logger.WithFields(logrus.Fields{
		"webhook_id": payload.ID,
		"event":      payload.Event,
		"url":        w.config.URL,
		"retry":      payload.RetryCount,
	}).Debug("Sending webhook")

	// Send the request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	// Read response body (limit to reasonable size)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024)) // 10KB limit
	if err != nil {
		w.logger.WithError(err).Warn("Failed to read response body")
	}

	// Log the result
	logFields := logrus.Fields{
		"webhook_id":    payload.ID,
		"event":         payload.Event,
		"url":           w.config.URL,
		"status_code":   resp.StatusCode,
		"duration_ms":   duration.Milliseconds(),
		"retry_count":   payload.RetryCount,
		"response_size": len(bodyBytes),
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.logger.WithFields(logFields).Info("Webhook delivered successfully")
		return nil
	}

	// Log error response
	logFields["response_body"] = string(bodyBytes)
	w.logger.WithFields(logFields).Warn("Webhook delivery failed")

	return fmt.Errorf("webhook delivery failed with status %d: %s", resp.StatusCode, string(bodyBytes))
}

// ValidateConfig validates the webhook configuration
func (w *HTTPWebhook) ValidateConfig() error {
	if w.config.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	if w.config.Timeout <= 0 {
		return fmt.Errorf("webhook timeout must be positive")
	}

	if w.config.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}

	if w.config.RetryDelay < 0 {
		return fmt.Errorf("retry delay cannot be negative")
	}

	return nil
}

// GetName returns the webhook name
func (w *HTTPWebhook) GetName() string {
	return "http"
}

// generateSignature generates HMAC-SHA256 signature for the payload
func (w *HTTPWebhook) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature verifies the HMAC-SHA256 signature
func (w *HTTPWebhook) VerifySignature(payload []byte, signature, secret string) bool {
	if secret == "" {
		return true // No secret configured, skip verification
	}

	expected := w.generateSignature(payload, secret)

	// Remove "sha256=" prefix if present
	if len(signature) > 7 && signature[:7] == "sha256=" {
		signature = signature[7:]
	}

	return hmac.Equal([]byte(expected), []byte(signature))
}

// TestConnection tests the webhook endpoint
func (w *HTTPWebhook) TestConnection(ctx context.Context) error {
	// Create a test payload
	testPayload := &Payload{
		ID:        "test-" + fmt.Sprintf("%d", time.Now().Unix()),
		Event:     "test.connection",
		Timestamp: time.Now(),
		Source:    "docker-auto",
		Data: map[string]interface{}{
			"test": true,
			"message": "This is a test webhook to verify connectivity",
		},
		Metadata: map[string]string{
			"test": "true",
		},
	}

	// Try to send the test payload
	return w.Send(ctx, testPayload)
}

// GetDeliveryInfo returns information about the last delivery attempt
func (w *HTTPWebhook) GetDeliveryInfo(ctx context.Context, payload *Payload) (*DeliveryResult, error) {
	startTime := time.Now()

	// Serialize payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", w.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers (same as Send method)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Docker-Auto-Webhook/1.0")
	req.Header.Set("X-Webhook-ID", payload.ID)
	req.Header.Set("X-Webhook-Event", payload.Event)
	req.Header.Set("X-Webhook-Timestamp", payload.Timestamp.Format(time.RFC3339))
	req.Header.Set("X-Webhook-Source", payload.Source)

	// Add custom headers
	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	// Add signature if secret is configured
	if w.config.Secret != "" {
		signature := w.generateSignature(jsonData, w.config.Secret)
		headerName := w.config.SignatureHeader
		if headerName == "" {
			headerName = "X-Hub-Signature-256"
		}
		req.Header.Set(headerName, "sha256="+signature)
	}

	// Send the request
	resp, err := w.client.Do(req)
	deliveredAt := time.Now()
	duration := deliveredAt.Sub(startTime)

	result := &DeliveryResult{
		ID:          payload.ID,
		URL:         w.config.URL,
		DeliveredAt: deliveredAt,
		Duration:    duration,
		RetryCount:  payload.RetryCount,
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	defer resp.Body.Close()

	// Read response
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024))
	if err != nil {
		result.Error = fmt.Sprintf("failed to read response: %v", err)
	} else {
		result.ResponseBody = string(bodyBytes)
	}

	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// Convert headers
	result.ResponseHeaders = make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			result.ResponseHeaders[key] = values[0]
		}
	}

	if !result.Success {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, result.ResponseBody)
	}

	return result, nil
}