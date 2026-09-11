package contact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/infrai-examples/healthtech-contact-inbox/internal/infrai"
)

const maxRequestBytes = 16 << 10
const allowedInbox = "chenhua@changba.com"

type EmailSender interface {
	SendEmail(context.Context, infrai.Email, string) (infrai.SendResult, error)
}

type Handler struct {
	sender EmailSender
	inbox  string
	logger *slog.Logger
}

type submission struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Organization string `json:"organization"`
	Message      string `json:"message"`
}

func NewHandler(sender EmailSender, inbox string, logger *slog.Logger) (*Handler, error) {
	if sender == nil || logger == nil {
		return nil, fmt.Errorf("sender and logger are required")
	}
	if _, err := mail.ParseAddress(inbox); err != nil {
		return nil, fmt.Errorf("invalid TEAM_INBOX: %w", err)
	}
	if inbox != allowedInbox {
		return nil, fmt.Errorf("TEAM_INBOX must be %s", allowedInbox)
	}
	return &Handler{sender: sender, inbox: inbox, logger: logger}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var form submission
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&form); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if err := validate(form); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	key := submissionKey(form)
	result, err := h.sender.SendEmail(r.Context(), infrai.Email{
		To:      h.inbox,
		Subject: "Healthtech contact: " + form.Organization,
		Body: fmt.Sprintf("Name: %s\nEmail: %s\nOrganization: %s\n\n%s",
			form.Name, form.Email, form.Organization, form.Message),
	}, key)
	if err != nil {
		h.logger.Error("contact delivery failed", "error", err)
		http.Error(w, "could not accept form", http.StatusBadGateway)
		return
	}

	h.logger.Info("contact delivered", "message_id", result.MessageID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func validate(form submission) error {
	form.Name = strings.TrimSpace(form.Name)
	form.Email = strings.TrimSpace(form.Email)
	form.Organization = strings.TrimSpace(form.Organization)
	form.Message = strings.TrimSpace(form.Message)
	if form.Name == "" || form.Organization == "" || form.Message == "" {
		return fmt.Errorf("name, organization, and message are required")
	}
	if len(form.Name) > 120 || len(form.Organization) > 160 || len(form.Message) > 4000 {
		return fmt.Errorf("form field is too long")
	}
	address, err := mail.ParseAddress(form.Email)
	if err != nil || address.Address != form.Email {
		return fmt.Errorf("valid email is required")
	}
	return nil
}

func submissionKey(form submission) string {
	sum := sha256.Sum256([]byte(form.Name + "\x00" + form.Email + "\x00" + form.Organization + "\x00" + form.Message))
	return "health-contact-" + hex.EncodeToString(sum[:])
}
