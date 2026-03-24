package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

// --- sendResponse tests ---

func TestSendResponse(t *testing.T) {
	w := httptest.NewRecorder()
	log := logrus.WithField("test", true)

	headers := map[string]string{
		"Content-Type": "application/json",
		"X-Custom":     "value",
	}
	body := []byte(`{"result":"ok"}`)

	err := sendResponse(log, w, &headers, http.StatusOK, &body, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != `{"result":"ok"}` {
		t.Errorf("body: got %q", w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", got, "application/json")
	}
	if got := w.Header().Get("X-Custom"); got != "value" {
		t.Errorf("X-Custom: got %q, want %q", got, "value")
	}
}

func TestSendResponse_NonOkStatus(t *testing.T) {
	w := httptest.NewRecorder()
	log := logrus.WithField("test", true)

	headers := map[string]string{}
	body := []byte("not found")

	err := sendResponse(log, w, &headers, http.StatusNotFound, &body, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSendResponse_EmptyBody(t *testing.T) {
	w := httptest.NewRecorder()
	log := logrus.WithField("test", true)

	headers := map[string]string{}
	body := []byte{}

	err := sendResponse(log, w, &headers, http.StatusNoContent, &body, "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
}
