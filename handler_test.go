package main

import (
	"bytes"
	"encoding/json"
	"lambdahttpgw/stats"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	stats.Init()
	os.Exit(m.Run())
}

// setupMockLambda starts an HTTP server that mimics the AWS Lambda Invoke API
// and configures the global lambdaClient to use it.
func setupMockLambda(handler http.HandlerFunc) (*httptest.Server, func()) {
	server := httptest.NewServer(handler)
	origClient := lambdaClient

	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Endpoint:    aws.String(server.URL),
		Credentials: credentials.NewStaticCredentials("fake", "fake", "fake"),
	}))
	lambdaClient = lambda.New(sess)

	return server, func() {
		lambdaClient = origClient
		server.Close()
	}
}

// --- parsePathRequest tests ---

func TestParsePathRequest(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		expectedFunction string
		expectedPath     string
		expectErr        bool
	}{
		{
			name:             "function with single path segment",
			path:             "/myFunction/pets",
			expectedFunction: "myFunction",
			expectedPath:     "/pets",
		},
		{
			name:             "function with deep path",
			path:             "/myFunction/some/deep/path",
			expectedFunction: "myFunction",
			expectedPath:     "/some/deep/path",
		},
		{
			name:             "function only, no trailing slash",
			path:             "/myFunction",
			expectedFunction: "myFunction",
			expectedPath:     "/",
		},
		{
			name:             "function with trailing slash",
			path:             "/myFunction/",
			expectedFunction: "myFunction",
			expectedPath:     "/",
		},
		{
			name:      "root path only",
			path:      "/",
			expectErr: true,
		},
		{
			name:      "empty path",
			path:      "",
			expectErr: true,
		},
		{
			name:             "function with query string in URL",
			path:             "/myFunction/path?key=value",
			expectedFunction: "myFunction",
			expectedPath:     "/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			fn, path, err := parsePathRequest(req)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if fn != tt.expectedFunction {
				t.Errorf("function name: got %q, want %q", fn, tt.expectedFunction)
			}
			if path != tt.expectedPath {
				t.Errorf("path: got %q, want %q", path, tt.expectedPath)
			}
		})
	}
}

// --- parseSubdomainRequest tests ---

func TestParseSubdomainRequest(t *testing.T) {
	origBaseDomain := baseDomain
	defer func() { baseDomain = origBaseDomain }()

	baseDomain = "live.mocks.cloud"

	tests := []struct {
		name             string
		host             string
		path             string
		expectedFunction string
		expectedPath     string
		expectErr        bool
		errContains      string
	}{
		{
			name:             "subdomain with single path segment",
			host:             "m7x3kq2b1p.live.mocks.cloud",
			path:             "/pets",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/pets",
		},
		{
			name:             "subdomain with deep path",
			host:             "m7x3kq2b1p.live.mocks.cloud",
			path:             "/api/v1/pets",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/api/v1/pets",
		},
		{
			name:             "subdomain with root path",
			host:             "m7x3kq2b1p.live.mocks.cloud",
			path:             "/",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/",
		},
		{
			name:             "subdomain with empty path defaults to /",
			host:             "m7x3kq2b1p.live.mocks.cloud",
			path:             "",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/",
		},
		{
			name:             "subdomain with port in host",
			host:             "m7x3kq2b1p.live.mocks.cloud:443",
			path:             "/pets",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/pets",
		},
		{
			name:             "subdomain with non-standard port",
			host:             "m7x3kq2b1p.live.mocks.cloud:8080",
			path:             "/test",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/test",
		},
		{
			name:             "hyphenated subdomain",
			host:             "brave-turing.live.mocks.cloud",
			path:             "/pets",
			expectedFunction: "brave-turing",
			expectedPath:     "/pets",
		},
		{
			name:        "no subdomain - bare domain",
			host:        "live.mocks.cloud",
			path:        "/pets",
			expectErr:   true,
			errContains: "does not match base domain",
		},
		{
			name:        "wrong domain entirely",
			host:        "m7x3kq2b1p.other.domain",
			path:        "/pets",
			expectErr:   true,
			errContains: "does not match base domain",
		},
		{
			name:        "partial domain match should fail",
			host:        "m7x3kq2b1p.notlive.mocks.cloud",
			path:        "/pets",
			expectErr:   true,
			errContains: "does not match base domain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, nil)
			req.Host = tt.host
			fn, path, err := parseSubdomainRequest(req)
			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if fn != tt.expectedFunction {
				t.Errorf("function name: got %q, want %q", fn, tt.expectedFunction)
			}
			if path != tt.expectedPath {
				t.Errorf("path: got %q, want %q", path, tt.expectedPath)
			}
		})
	}
}

func TestValidateConfig_SubdomainModeRequiresBaseDomain(t *testing.T) {
	origMode := routingMode
	origBaseDomain := baseDomain
	defer func() {
		routingMode = origMode
		baseDomain = origBaseDomain
	}()

	routingMode = "subdomain"
	baseDomain = ""

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when BASE_DOMAIN is empty in subdomain mode")
		}
	}()
	validateConfig()
}

func TestValidateConfig_SubdomainModeWithBaseDomain(t *testing.T) {
	origMode := routingMode
	origBaseDomain := baseDomain
	defer func() {
		routingMode = origMode
		baseDomain = origBaseDomain
	}()

	routingMode = "subdomain"
	baseDomain = "live.mocks.cloud"

	// Should not panic
	validateConfig()
}

func TestValidateConfig_PathModeNoBaseDomainOk(t *testing.T) {
	origMode := routingMode
	origBaseDomain := baseDomain
	defer func() {
		routingMode = origMode
		baseDomain = origBaseDomain
	}()

	routingMode = "path"
	baseDomain = ""

	// Should not panic
	validateConfig()
}

// --- parseRequest integration tests (routing mode dispatch + prefix) ---

func TestParseRequest_PathMode(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	routingMode = "path"
	functionPrefix = ""

	req, _ := http.NewRequest("GET", "/myFunction/path", nil)
	fn, path, _, _, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn != "myFunction" {
		t.Errorf("function name: got %q, want %q", fn, "myFunction")
	}
	if path != "/path" {
		t.Errorf("path: got %q, want %q", path, "/path")
	}
}

func TestParseRequest_SubdomainMode(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	origBaseDomain := baseDomain
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
		baseDomain = origBaseDomain
	}()

	routingMode = "subdomain"
	functionPrefix = ""
	baseDomain = "live.mocks.cloud"

	req, _ := http.NewRequest("GET", "/pets", nil)
	req.Host = "abc123.live.mocks.cloud"
	fn, path, _, _, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn != "abc123" {
		t.Errorf("function name: got %q, want %q", fn, "abc123")
	}
	if path != "/pets" {
		t.Errorf("path: got %q, want %q", path, "/pets")
	}
}

func TestParseRequest_FunctionPrefix_PathMode(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	routingMode = "path"
	functionPrefix = "imposter-"

	req, _ := http.NewRequest("GET", "/abc123/path", nil)
	fn, _, _, _, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn != "imposter-abc123" {
		t.Errorf("function name: got %q, want %q", fn, "imposter-abc123")
	}
}

func TestParseRequest_FunctionPrefix_SubdomainMode(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	origBaseDomain := baseDomain
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
		baseDomain = origBaseDomain
	}()

	routingMode = "subdomain"
	functionPrefix = "imposter-"
	baseDomain = "live.mocks.cloud"

	req, _ := http.NewRequest("GET", "/pets", nil)
	req.Host = "m7x3kq2b1p.live.mocks.cloud"
	fn, _, _, _, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn != "imposter-m7x3kq2b1p" {
		t.Errorf("function name: got %q, want %q", fn, "imposter-m7x3kq2b1p")
	}
}

func TestParseRequest_NoPrefix(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	routingMode = "path"
	functionPrefix = ""

	req, _ := http.NewRequest("GET", "/rawName/path", nil)
	fn, _, _, _, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn != "rawName" {
		t.Errorf("function name: got %q, want %q", fn, "rawName")
	}
}

func TestParseRequest_DefaultRoutingModeIsPath(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	// Anything that's not "subdomain" should behave as path mode
	routingMode = ""
	functionPrefix = ""

	req, _ := http.NewRequest("GET", "/myFunc/test", nil)
	fn, path, _, _, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn != "myFunc" {
		t.Errorf("function name: got %q, want %q", fn, "myFunc")
	}
	if path != "/test" {
		t.Errorf("path: got %q, want %q", path, "/test")
	}
}

// --- Header and body parsing tests ---

func TestParseRequest_HeadersParsed(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	routingMode = "path"
	functionPrefix = ""

	req, _ := http.NewRequest("GET", "/fn/path", nil)
	req.Header.Set("X-Custom-Header", "test-value")
	req.Header.Set("Content-Type", "application/json")

	_, _, headers, _, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if headers == nil {
		t.Fatal("headers should not be nil")
	}
	if (*headers)["X-Custom-Header"] != "test-value" {
		t.Errorf("X-Custom-Header: got %q, want %q", (*headers)["X-Custom-Header"], "test-value")
	}
	if (*headers)["Content-Type"] != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", (*headers)["Content-Type"], "application/json")
	}
}

func TestParseRequest_BodyParsed(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	routingMode = "path"
	functionPrefix = ""

	bodyContent := `{"key":"value"}`
	req, _ := http.NewRequest("POST", "/fn/path", bytes.NewBufferString(bodyContent))

	_, _, _, body, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body == nil {
		t.Fatal("body should not be nil")
	}
	if string(*body) != bodyContent {
		t.Errorf("body: got %q, want %q", string(*body), bodyContent)
	}
}

func TestParseRequest_EmptyBody(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	routingMode = "path"
	functionPrefix = ""

	req, _ := http.NewRequest("GET", "/fn/path", nil)

	_, _, _, body, err := parseRequest(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body == nil {
		t.Fatal("body should not be nil")
	}
	if len(*body) != 0 {
		t.Errorf("body should be empty, got %d bytes", len(*body))
	}
}

// --- Error propagation tests ---

func TestParseRequest_PathModeError(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	routingMode = "path"
	functionPrefix = ""

	req, _ := http.NewRequest("GET", "/", nil)
	_, _, _, _, err := parseRequest(req)
	if err == nil {
		t.Error("expected error for root path in path mode")
	}
}

func TestParseRequest_SubdomainModeError(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	origBaseDomain := baseDomain
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
		baseDomain = origBaseDomain
	}()

	routingMode = "subdomain"
	functionPrefix = "imposter-"
	baseDomain = "live.mocks.cloud"

	req, _ := http.NewRequest("GET", "/pets", nil)
	req.Host = "wrong.domain.com"
	_, _, _, _, err := parseRequest(req)
	if err == nil {
		t.Error("expected error for mismatched domain")
	}
}

// --- HTTP method tests ---

func TestParseRequest_DifferentMethods(t *testing.T) {
	origMode := routingMode
	origPrefix := functionPrefix
	defer func() {
		routingMode = origMode
		functionPrefix = origPrefix
	}()

	routingMode = "path"
	functionPrefix = ""

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, "/fn/path", nil)
			fn, path, _, _, err := parseRequest(req)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", method, err)
			}
			if fn != "fn" {
				t.Errorf("function name: got %q, want %q", fn, "fn")
			}
			if path != "/path" {
				t.Errorf("path: got %q, want %q", path, "/path")
			}
		})
	}
}

// --- statusHandler tests ---

func TestStatusHandler(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/system/status", nil)
	statusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ok\n" {
		t.Errorf("body: got %q, want %q", w.Body.String(), "ok\n")
	}
}

// --- getRequestId tests ---

func TestGetRequestId_FromHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "my-id-123")

	id := getRequestId("X-Request-ID", req)
	if id != "my-id-123" {
		t.Errorf("expected my-id-123, got %q", id)
	}
}

func TestGetRequestId_GeneratesUUID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	id := getRequestId("", req)
	if id == "" {
		t.Error("expected a generated UUID, got empty")
	}
	// UUID format: 8-4-4-4-12
	if len(id) != 36 {
		t.Errorf("expected UUID length 36, got %d", len(id))
	}
}

func TestGetRequestId_MissingHeaderGeneratesUUID(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	// Header name is set but header value is absent
	id := getRequestId("X-Missing-Header", req)
	if id == "" {
		t.Error("expected a generated UUID, got empty")
	}
	if len(id) != 36 {
		t.Errorf("expected UUID length 36, got %d", len(id))
	}
}

// --- setCorsHeaders tests ---

func TestSetCorsHeaders_WithOrigin(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")

	setCorsHeaders(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("Allow-Origin: got %q, want %q", got, "https://example.com")
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Allow-Credentials: got %q, want %q", got, "true")
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Error("Allow-Methods should not be empty")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Allow-Headers should not be empty")
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Error("Allow-Headers should include Authorization")
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Content-Type") {
		t.Error("Allow-Headers should include Content-Type")
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("Max-Age: got %q, want %q", got, "86400")
	}
}

func TestSetCorsHeaders_NoOrigin(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	setCorsHeaders(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Allow-Origin: got %q, want %q", got, "*")
	}
}

// --- sendResponse tests ---

func TestSendResponse(t *testing.T) {
	w := httptest.NewRecorder()
	log := logrus.WithField("test", true)

	headers := map[string]string{
		"Content-Type":  "application/json",
		"X-Custom":      "value",
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

// --- handler CORS integration tests ---

func TestHandler_CorsPreflightWhenEnabled(t *testing.T) {
	origCors := permissiveCorsEnabled
	defer func() { permissiveCorsEnabled = origCors }()

	permissiveCorsEnabled = true

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/someFunction/path", nil)
	req.Header.Set("Origin", "https://myapp.example.com")

	handler(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNoContent)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://myapp.example.com" {
		t.Errorf("Allow-Origin: got %q, want %q", got, "https://myapp.example.com")
	}
}

func TestHandler_CorsDisabled_NoHeaders(t *testing.T) {
	origCors := permissiveCorsEnabled
	origMode := routingMode
	defer func() {
		permissiveCorsEnabled = origCors
		routingMode = origMode
	}()

	permissiveCorsEnabled = false
	routingMode = "path"

	w := httptest.NewRecorder()
	// Use root path to trigger 400 before lambda invocation
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "https://myapp.example.com")

	handler(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Allow-Origin should be empty when CORS disabled, got %q", got)
	}
}

func TestHandler_CorsHeadersOnNonPreflight(t *testing.T) {
	origCors := permissiveCorsEnabled
	origMode := routingMode
	defer func() {
		permissiveCorsEnabled = origCors
		routingMode = origMode
	}()

	permissiveCorsEnabled = true
	routingMode = "path"

	w := httptest.NewRecorder()
	// Use root path so handler returns 400 before reaching Lambda invocation
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://myapp.example.com")

	handler(w, req)

	// CORS headers should still be set even when the request fails
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://myapp.example.com" {
		t.Errorf("Allow-Origin: got %q, want %q", got, "https://myapp.example.com")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandler_BadRequest_EmptyPath(t *testing.T) {
	origCors := permissiveCorsEnabled
	origMode := routingMode
	defer func() {
		permissiveCorsEnabled = origCors
		routingMode = origMode
	}()

	permissiveCorsEnabled = false
	routingMode = "path"

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

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

// --- full handler integration tests ---

func TestHandler_SuccessfulInvocation(t *testing.T) {
	lambdaResp := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{"X-Custom": "resp-value"},
		Body:       `{"result":"ok"}`,
	}
	respPayload, _ := json.Marshal(lambdaResp)

	_, cleanup := setupMockLambda(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		w.Write(respPayload)
	}))
	defer cleanup()

	origCors := permissiveCorsEnabled
	origMode := routingMode
	defer func() {
		permissiveCorsEnabled = origCors
		routingMode = origMode
	}()
	permissiveCorsEnabled = false
	routingMode = "path"

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/myFunction/path", nil)

	handler(w, req)

	if w.Code != 200 {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	if w.Body.String() != `{"result":"ok"}` {
		t.Errorf("body: got %q", w.Body.String())
	}
	if got := w.Header().Get("X-Custom"); got != "resp-value" {
		t.Errorf("X-Custom: got %q, want %q", got, "resp-value")
	}
}

func TestHandler_LambdaInvocationError(t *testing.T) {
	_, cleanup := setupMockLambda(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer cleanup()

	origCors := permissiveCorsEnabled
	origMode := routingMode
	defer func() {
		permissiveCorsEnabled = origCors
		routingMode = origMode
	}()
	permissiveCorsEnabled = false
	routingMode = "path"

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/myFunction/path", nil)

	handler(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadGateway)
	}
}

func TestHandler_CorsWithSuccessfulInvocation(t *testing.T) {
	lambdaResp := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    map[string]string{},
		Body:       "ok",
	}
	respPayload, _ := json.Marshal(lambdaResp)

	_, cleanup := setupMockLambda(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		w.Write(respPayload)
	}))
	defer cleanup()

	origCors := permissiveCorsEnabled
	origMode := routingMode
	defer func() {
		permissiveCorsEnabled = origCors
		routingMode = origMode
	}()
	permissiveCorsEnabled = true
	routingMode = "path"

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/myFunction/api", nil)
	req.Header.Set("Origin", "https://app.example.com")

	handler(w, req)

	if w.Code != 200 {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Allow-Origin: got %q, want %q", got, "https://app.example.com")
	}
}
