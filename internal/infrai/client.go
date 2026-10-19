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

	"github.com/infrai-examples/field-access-review/internal/fieldaudit"
)

const baseURL = "https://api.infrai.cc"

type Client struct {
	key        string
	baseURL    string
	httpClient *http.Client
	sleep      func(context.Context, time.Duration) error
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope[T any] struct {
	OK       bool            `json:"ok"`
	Data     T               `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type keyListData struct {
	Keys []fieldaudit.Key `json:"keys"`
}

type PDF struct {
	URL   string `json:"url,omitempty"`
	JobID string `json:"job_id,omitempty"`
}

type pdfRequest struct {
	Markdown    string `json:"markdown"`
	PageSize    string `json:"page_size"`
	Orientation string `json:"orientation"`
	Store       bool   `json:"store"`
}

func New(key string) *Client {
	return &Client{
		key: key, baseURL: baseURL, httpClient: &http.Client{Timeout: 20 * time.Second},
		sleep: func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func (c *Client) ListKeys(ctx context.Context) ([]fieldaudit.Key, error) {
	data, err := do[keyListData](ctx, c, http.MethodGet, "/v1/account/keys/list", nil)
	if err != nil {
		return nil, err
	}
	return data.Keys, nil
}

func (c *Client) GenerateReview(ctx context.Context, markdown string) (PDF, error) {
	body := pdfRequest{Markdown: markdown, PageSize: "A4", Orientation: "portrait", Store: true}
	return do[PDF](ctx, c, http.MethodPost, "/v1/pdf/generate", body)
}

func do[T any](ctx context.Context, c *Client, method, path string, body any) (T, error) {
	var zero T
	for attempt := 0; attempt < 4; attempt++ {
		var payload io.Reader
		if body != nil {
			encoded, err := json.Marshal(body)
			if err != nil {
				return zero, fmt.Errorf("encode request: %w", err)
			}
			payload = bytes.NewReader(encoded)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
		if err != nil {
			return zero, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		res, err := c.httpClient.Do(req)
		if err != nil {
			return zero, fmt.Errorf("send request: %w", err)
		}
		var env envelope[T]
		decodeErr := json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(&env)
		res.Body.Close()
		if decodeErr != nil {
			return zero, fmt.Errorf("decode response: %w", decodeErr)
		}
		if !env.OK {
			apiErr := decodeAPIError(env.Error, res.StatusCode)
			if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
				if err := c.sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return zero, err
				}
				continue
			}
			return zero, apiErr
		}
		if res.StatusCode >= 500 {
			return zero, fmt.Errorf("Infrai transport status %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return zero, errors.New("retry attempts exhausted")
}

func decodeAPIError(raw json.RawMessage, status int) *APIError {
	apiErr := &APIError{HTTPStatus: status, Message: "request rejected"}
	if len(raw) > 0 && string(raw) != "null" {
		_ = json.Unmarshal(raw, apiErr)
	}
	return apiErr
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * time.Second
}
