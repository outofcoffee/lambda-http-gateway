package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/sirupsen/logrus"
)

// --- invoke tests ---

func TestInvoke_Success(t *testing.T) {
	lambdaResp := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       `{"message":"hello"}`,
	}
	respPayload, _ := json.Marshal(lambdaResp)

	_, cleanup := setupMockLambda(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		w.Write(respPayload)
	}))
	defer cleanup()

	log := logrus.WithField("test", true)
	headers := map[string]string{"Content-Type": "application/json"}
	body := []byte(`{"key":"value"}`)

	code, respBody, respHeaders, err := invoke(log, "testFunc", "POST", "/path", &headers, &body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 200 {
		t.Errorf("status: got %d, want 200", code)
	}
	if string(*respBody) != `{"message":"hello"}` {
		t.Errorf("body: got %q", string(*respBody))
	}
	if (*respHeaders)["Content-Type"] != "application/json" {
		t.Errorf("Content-Type header: got %q", (*respHeaders)["Content-Type"])
	}
}

func TestInvoke_Base64Response(t *testing.T) {
	lambdaResp := events.APIGatewayProxyResponse{
		StatusCode:      200,
		Headers:         map[string]string{},
		Body:            "aGVsbG8=", // "hello" base64
		IsBase64Encoded: true,
	}
	respPayload, _ := json.Marshal(lambdaResp)

	_, cleanup := setupMockLambda(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		w.Write(respPayload)
	}))
	defer cleanup()

	log := logrus.WithField("test", true)
	headers := map[string]string{}
	body := []byte{}

	code, respBody, _, err := invoke(log, "testFunc", "GET", "/path", &headers, &body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 200 {
		t.Errorf("status: got %d, want 200", code)
	}
	if string(*respBody) != "hello" {
		t.Errorf("body: got %q, want %q", string(*respBody), "hello")
	}
}

func TestInvoke_LambdaError(t *testing.T) {
	_, cleanup := setupMockLambda(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer cleanup()

	log := logrus.WithField("test", true)
	headers := map[string]string{}
	body := []byte{}

	_, _, _, err := invoke(log, "testFunc", "GET", "/path", &headers, &body)
	if err == nil {
		t.Error("expected error for failed Lambda invocation")
	}
}

func TestInvoke_InvalidResponse(t *testing.T) {
	_, cleanup := setupMockLambda(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer cleanup()

	log := logrus.WithField("test", true)
	headers := map[string]string{}
	body := []byte{}

	_, _, _, err := invoke(log, "testFunc", "GET", "/path", &headers, &body)
	if err == nil {
		t.Error("expected error for invalid response payload")
	}
}

func TestInvoke_BadBase64Response(t *testing.T) {
	lambdaResp := events.APIGatewayProxyResponse{
		StatusCode:      200,
		Headers:         map[string]string{},
		Body:            "!!!not-base64!!!",
		IsBase64Encoded: true,
	}
	respPayload, _ := json.Marshal(lambdaResp)

	_, cleanup := setupMockLambda(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		w.Write(respPayload)
	}))
	defer cleanup()

	log := logrus.WithField("test", true)
	headers := map[string]string{}
	body := []byte{}

	_, _, _, err := invoke(log, "testFunc", "GET", "/path", &headers, &body)
	if err == nil {
		t.Error("expected error for bad base64 body")
	}
}
