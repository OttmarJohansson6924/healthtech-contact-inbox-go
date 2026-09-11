package contact

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/infrai-examples/healthtech-contact-inbox/internal/infrai"
)

type recordingSender struct {
	email infrai.Email
	key   string
}

func (s *recordingSender) SendEmail(_ context.Context, email infrai.Email, key string) (infrai.SendResult, error) {
	s.email = email
	s.key = key
	return infrai.SendResult{MessageID: "msg_contact_42"}, nil
}

func TestHandlerRoutesContactToTeamInbox(t *testing.T) {
	sender := &recordingSender{}
	handler, err := NewHandler(sender, allowedInbox, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	body := `{"name":"Mina Patel","email":"mina@clinic.example","organization":"North Clinic","message":"We need an intake integration."}`
	request := httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader(body))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if sender.email.To != allowedInbox {
		t.Fatalf("recipient = %q", sender.email.To)
	}
	if sender.key == "" || !strings.Contains(sender.email.Body, "mina@clinic.example") {
		t.Fatal("delivery lacks idempotency key or contact address")
	}
}

func TestNewHandlerRejectsUnapprovedInbox(t *testing.T) {
	_, err := NewHandler(&recordingSender{}, "care-team@example.org", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected unapproved inbox to be rejected")
	}
}
