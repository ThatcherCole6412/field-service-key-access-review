package infrai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestListKeysParsesBusinessRejectionBeforeStatus(t *testing.T) {
	client := testClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/account/keys/list" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		return response(http.StatusBadRequest, `{"ok":false,"data":null,"error":{"code":"INVALID_ARGUMENT","message":"invalid filter"},"metadata":{}}`), nil
	})

	_, err := client.ListKeys(context.Background())
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "INVALID_ARGUMENT" || apiErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("error = %#v, want parsed API error", err)
	}
}

func TestGenerateReviewRetries429(t *testing.T) {
	attempts := 0
	client := testClient(func(req *http.Request) (*http.Response, error) {
		attempts++
		if req.Method != http.MethodPost || req.URL.Path != "/v1/pdf/generate" {
			t.Fatalf("request = %s %s", req.Method, req.URL.Path)
		}
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing bearer authorization")
		}
		if attempts == 1 {
			res := response(http.StatusTooManyRequests, `{"ok":false,"data":null,"error":{"code":"RATE_LIMITED","message":"retry later"},"metadata":{}}`)
			res.Header.Set("Retry-After", "0")
			return res, nil
		}
		return response(http.StatusOK, `{"ok":true,"data":{"url":"https://example.test/review.pdf"},"error":null,"metadata":{}}`), nil
	})
	client.sleep = func(context.Context, time.Duration) error { return nil }

	pdf, err := client.GenerateReview(context.Background(), "# Review")
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || pdf.URL != "https://example.test/review.pdf" {
		t.Fatalf("attempts = %d, pdf = %#v", attempts, pdf)
	}
}

func testClient(fn roundTripFunc) *Client {
	client := New("test-key")
	client.baseURL = "https://example.test"
	client.httpClient = &http.Client{Transport: fn}
	return client
}

func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
