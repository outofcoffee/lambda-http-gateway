package main

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

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
			name:             "function with query string in path",
			path:             "/myFunction/path?key=value",
			expectedFunction: "myFunction",
			expectedPath:     "/path?key=value",
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
			errContains: "no subdomain",
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

func TestParseSubdomainRequest_BaseDomainRequired(t *testing.T) {
	origBaseDomain := baseDomain
	defer func() { baseDomain = origBaseDomain }()

	baseDomain = ""

	req, _ := http.NewRequest("GET", "/pets", nil)
	req.Host = "test.live.mocks.cloud"
	_, _, err := parseSubdomainRequest(req)
	if err == nil {
		t.Error("expected error when BASE_DOMAIN is empty")
	}
	if !strings.Contains(err.Error(), "BASE_DOMAIN") {
		t.Errorf("error should mention BASE_DOMAIN, got: %v", err)
	}
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
