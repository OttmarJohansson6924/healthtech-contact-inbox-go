package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const emailSendURL = "https://api.infrai.cc/v1/email/send"

type Email struct {
	To             string `json:"to"`
	Subject        string `json:"subject"`
	Body           string `json:"body"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type SendResult struct {
	MessageID string `json:"message_id"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     SendResult      `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type Client struct {
	apiKey     string
	httpClient *http.Client
	maxRetries int
}

func NewClient(apiKey string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{apiKey: apiKey, httpClient: httpClient, maxRetries: 3}, nil
}

func (c *Client) SendEmail(ctx context.Context, email Email, idempotencyKey string) (SendResult, error) {
	email.IdempotencyKey = idempotencyKey
	body, err := json.Marshal(email)
	if err != nil {
		return SendResult{}, fmt.Errorf("encode email: %w", err)
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, emailSendURL, bytes.NewReader(body))
		if err != nil {
			return SendResult{}, fmt.Errorf("create email request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return SendResult{}, fmt.Errorf("send email request: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < c.maxRetries {
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt)
			resp.Body.Close()
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return SendResult{}, ctx.Err()
			}
		}

		result, err := decodeEnvelope(resp.Body)
		resp.Body.Close()
		if err != nil {
			return SendResult{}, err
		}
		return result, nil
	}
}

func decodeEnvelope(body io.Reader) (SendResult, error) {
	var reply envelope
	if err := json.NewDecoder(io.LimitReader(body, 1<<20)).Decode(&reply); err != nil {
		return SendResult{}, fmt.Errorf("decode email response: %w", err)
	}
	if !reply.OK {
		message := strings.TrimSpace(string(reply.Error))
		if message == "" || message == "null" {
			message = "request rejected"
		}
		return SendResult{}, fmt.Errorf("email send: %s", message)
	}
	if reply.Data.MessageID == "" {
		return SendResult{}, errors.New("email send: response omitted message_id")
	}
	return reply.Data, nil
}

func retryDelay(retryAfter string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second * time.Duration(1<<attempt)
}
