package main

import (
	"net/http"
	"testing"
)

func TestParsePathRequest(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		expectedFunction string
		expectedPath     string
		expectErr        bool
	}{
		{
			name:             "function with path",
			path:             "/myFunction/some/path",
			expectedFunction: "myFunction",
			expectedPath:     "/some/path",
		},
		{
			name:             "function only",
			path:             "/myFunction",
			expectedFunction: "myFunction",
			expectedPath:     "/",
		},
		{
			name:      "empty path",
			path:      "/",
			expectErr: true,
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

func TestParseSubdomainRequest(t *testing.T) {
	// Save and restore global
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
	}{
		{
			name:             "subdomain with path",
			host:             "m7x3kq2b1p.live.mocks.cloud",
			path:             "/pets",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/pets",
		},
		{
			name:             "subdomain with root path",
			host:             "m7x3kq2b1p.live.mocks.cloud",
			path:             "/",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/",
		},
		{
			name:             "subdomain with port",
			host:             "m7x3kq2b1p.live.mocks.cloud:443",
			path:             "/pets",
			expectedFunction: "m7x3kq2b1p",
			expectedPath:     "/pets",
		},
		{
			name:      "no subdomain",
			host:      "live.mocks.cloud",
			path:      "/pets",
			expectErr: true,
		},
		{
			name:      "wrong domain",
			host:      "m7x3kq2b1p.other.domain",
			path:      "/pets",
			expectErr: true,
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

func TestFunctionPrefix(t *testing.T) {
	// Save and restore globals
	origPrefix := functionPrefix
	origMode := routingMode
	defer func() {
		functionPrefix = origPrefix
		routingMode = origMode
	}()

	functionPrefix = "imposter-"
	routingMode = "path"

	req, _ := http.NewRequest("GET", "/myFunction/path", nil)
	fn, _, _, _, err := parseRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if fn != "imposter-myFunction" {
		t.Errorf("function name: got %q, want %q", fn, "imposter-myFunction")
	}
}

func TestSubdomainBaseDomainRequired(t *testing.T) {
	origBaseDomain := baseDomain
	defer func() { baseDomain = origBaseDomain }()

	baseDomain = ""

	req, _ := http.NewRequest("GET", "/pets", nil)
	req.Host = "test.live.mocks.cloud"
	_, _, err := parseSubdomainRequest(req)
	if err == nil {
		t.Error("expected error when BASE_DOMAIN is empty")
	}
}
