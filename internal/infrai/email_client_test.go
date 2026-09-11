package infrai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestSendEmailBuildsAuthenticatedIdempotentRequest(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s", request.Method)
		}
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing bearer authentication")
		}
		if request.Header.Get("Idempotency-Key") != "submission-42" {
			t.Fatal("missing idempotency key")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"ok":true,"data":{"message_id":"msg_42"},"metadata":{}}`)),
		}, nil
	})
	client, err := NewClient("test-key", &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}

	result, err := client.SendEmail(context.Background(), Email{To: "team@example.org", Subject: "Contact", Body: "Hello"}, "submission-42")
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != "msg_42" {
		t.Fatalf("message ID = %q", result.MessageID)
	}
}
