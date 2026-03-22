package config

import (
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestGetConfigLevel_Default(t *testing.T) {
	os.Unsetenv("LOG_LEVEL")
	if level := GetConfigLevel(); level != logrus.DebugLevel {
		t.Errorf("expected DebugLevel, got %v", level)
	}
}

func TestGetConfigLevel_Set(t *testing.T) {
	os.Setenv("LOG_LEVEL", "warn")
	defer os.Unsetenv("LOG_LEVEL")
	if level := GetConfigLevel(); level != logrus.WarnLevel {
		t.Errorf("expected WarnLevel, got %v", level)
	}
}

func TestGetConfigLevel_Invalid(t *testing.T) {
	os.Setenv("LOG_LEVEL", "invalid")
	defer os.Unsetenv("LOG_LEVEL")
	if level := GetConfigLevel(); level != logrus.DebugLevel {
		t.Errorf("expected DebugLevel for invalid input, got %v", level)
	}
}

func TestGetPort_Default(t *testing.T) {
	os.Unsetenv("PORT")
	if port := GetPort(); port != "8090" {
		t.Errorf("expected 8090, got %v", port)
	}
}

func TestGetPort_Set(t *testing.T) {
	os.Setenv("PORT", "3000")
	defer os.Unsetenv("PORT")
	if port := GetPort(); port != "3000" {
		t.Errorf("expected 3000, got %v", port)
	}
}

func TestGetRegion_Default(t *testing.T) {
	os.Unsetenv("AWS_REGION")
	if region := GetRegion(); region != "eu-west-1" {
		t.Errorf("expected eu-west-1, got %v", region)
	}
}

func TestGetRegion_Set(t *testing.T) {
	os.Setenv("AWS_REGION", "us-east-1")
	defer os.Unsetenv("AWS_REGION")
	if region := GetRegion(); region != "us-east-1" {
		t.Errorf("expected us-east-1, got %v", region)
	}
}

func TestGetRequestIdHeader_Default(t *testing.T) {
	os.Unsetenv("REQUEST_ID_HEADER")
	if header := GetRequestIdHeader(); header != "" {
		t.Errorf("expected empty, got %v", header)
	}
}

func TestGetRequestIdHeader_Set(t *testing.T) {
	os.Setenv("REQUEST_ID_HEADER", "x-correlation-id")
	defer os.Unsetenv("REQUEST_ID_HEADER")
	if header := GetRequestIdHeader(); header != "x-correlation-id" {
		t.Errorf("expected x-correlation-id, got %v", header)
	}
}

func TestGetRoutingMode_Default(t *testing.T) {
	os.Unsetenv("ROUTING_MODE")
	if mode := GetRoutingMode(); mode != "path" {
		t.Errorf("expected path, got %v", mode)
	}
}

func TestGetRoutingMode_Set(t *testing.T) {
	os.Setenv("ROUTING_MODE", "subdomain")
	defer os.Unsetenv("ROUTING_MODE")
	if mode := GetRoutingMode(); mode != "subdomain" {
		t.Errorf("expected subdomain, got %v", mode)
	}
}

func TestGetBaseDomain(t *testing.T) {
	os.Unsetenv("BASE_DOMAIN")
	if domain := GetBaseDomain(); domain != "" {
		t.Errorf("expected empty, got %v", domain)
	}

	os.Setenv("BASE_DOMAIN", "live.mocks.cloud")
	defer os.Unsetenv("BASE_DOMAIN")
	if domain := GetBaseDomain(); domain != "live.mocks.cloud" {
		t.Errorf("expected live.mocks.cloud, got %v", domain)
	}
}

func TestGetFunctionPrefix(t *testing.T) {
	os.Unsetenv("FUNCTION_PREFIX")
	if prefix := GetFunctionPrefix(); prefix != "" {
		t.Errorf("expected empty, got %v", prefix)
	}

	os.Setenv("FUNCTION_PREFIX", "imposter-")
	defer os.Unsetenv("FUNCTION_PREFIX")
	if prefix := GetFunctionPrefix(); prefix != "imposter-" {
		t.Errorf("expected imposter-, got %v", prefix)
	}
}

func TestIsPermissiveCorsEnabled(t *testing.T) {
	os.Unsetenv("CORS_PERMISSIVE")
	if IsPermissiveCorsEnabled() {
		t.Error("expected false when unset")
	}

	os.Setenv("CORS_PERMISSIVE", "false")
	if IsPermissiveCorsEnabled() {
		t.Error("expected false for 'false'")
	}

	os.Setenv("CORS_PERMISSIVE", "true")
	defer os.Unsetenv("CORS_PERMISSIVE")
	if !IsPermissiveCorsEnabled() {
		t.Error("expected true for 'true'")
	}
}

func TestIsStatsRecorderEnabled(t *testing.T) {
	os.Unsetenv("STATS_RECORDER")
	os.Unsetenv("STATS_REPORT_URL")
	if isStatsRecorderEnabled() {
		t.Error("expected false when unset")
	}

	os.Setenv("STATS_RECORDER", "true")
	defer os.Unsetenv("STATS_RECORDER")
	if !isStatsRecorderEnabled() {
		t.Error("expected true when STATS_RECORDER=true")
	}
}

func TestIsStatsRecorderEnabled_ViaReporter(t *testing.T) {
	os.Unsetenv("STATS_RECORDER")
	os.Setenv("STATS_REPORT_URL", "http://example.com")
	defer os.Unsetenv("STATS_REPORT_URL")
	if !isStatsRecorderEnabled() {
		t.Error("expected true when reporter is enabled")
	}
}

func TestGetStatsUrl(t *testing.T) {
	os.Unsetenv("STATS_REPORT_URL")
	if url := getStatsUrl(); url != "" {
		t.Errorf("expected empty, got %v", url)
	}

	os.Setenv("STATS_REPORT_URL", "http://example.com")
	defer os.Unsetenv("STATS_REPORT_URL")
	if url := getStatsUrl(); url != "http://example.com" {
		t.Errorf("expected http://example.com, got %v", url)
	}
}

func TestGetStatsInterval_Default(t *testing.T) {
	os.Unsetenv("STATS_REPORT_INTERVAL")
	if interval := GetStatsInterval(); interval != 5*time.Second {
		t.Errorf("expected 5s, got %v", interval)
	}
}

func TestGetStatsInterval_Set(t *testing.T) {
	os.Setenv("STATS_REPORT_INTERVAL", "2m")
	defer os.Unsetenv("STATS_REPORT_INTERVAL")
	if interval := GetStatsInterval(); interval != 2*time.Minute {
		t.Errorf("expected 2m, got %v", interval)
	}
}

func TestIsStatsReporterEnabled(t *testing.T) {
	os.Unsetenv("STATS_REPORT_URL")
	if isStatsReporterEnabled() {
		t.Error("expected false when URL unset")
	}

	os.Setenv("STATS_REPORT_URL", "http://example.com")
	defer os.Unsetenv("STATS_REPORT_URL")
	if !isStatsReporterEnabled() {
		t.Error("expected true when URL set")
	}
}
