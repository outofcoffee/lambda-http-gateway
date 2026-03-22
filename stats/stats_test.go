package stats

import (
	"fmt"
	"lambdahttpgw/config"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestInit_Disabled(t *testing.T) {
	os.Unsetenv("STATS_RECORDER")
	os.Unsetenv("STATS_REPORT_URL")
	// Should not panic
	Init()
}

func TestRecordHit_Disabled(t *testing.T) {
	os.Unsetenv("STATS_RECORDER")
	os.Unsetenv("STATS_REPORT_URL")
	// Should be a no-op when disabled
	RecordHit(Invocation{FunctionName: "test", Duration: time.Second})
}

func TestStatsHolder(t *testing.T) {
	functionStats = map[string]*statsHolder{}

	functionStats["funcA"] = &statsHolder{Hits: 0, LastReport: 0}
	functionStats["funcA"].Hits++
	functionStats["funcA"].Hits++

	functionStats["funcB"] = &statsHolder{Hits: 0, LastReport: 0}
	functionStats["funcB"].Hits++

	stats := GetAllStats()

	if stats["funcA"].Hits != 2 {
		t.Errorf("funcA hits: got %d, want 2", stats["funcA"].Hits)
	}
	if stats["funcB"].Hits != 1 {
		t.Errorf("funcB hits: got %d, want 1", stats["funcB"].Hits)
	}
}

func TestGetAllStats_Empty(t *testing.T) {
	functionStats = map[string]*statsHolder{}
	stats := GetAllStats()
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d entries", len(stats))
	}
}

func TestReportStats_NoPending(t *testing.T) {
	functionStats = map[string]*statsHolder{}
	reportStats()
}

func TestReportStats_WithPending(t *testing.T) {
	var receivedPath string
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	origUrl := config.StatsUrl
	config.StatsUrl = server.URL
	defer func() { config.StatsUrl = origUrl }()

	functionStats = map[string]*statsHolder{
		"myFunc": {Hits: 5, LastReport: 0},
	}

	reportStats()

	if receivedPath != "/hits/myFunc" {
		t.Errorf("path: got %q, want %q", receivedPath, "/hits/myFunc")
	}
	if receivedBody != "5" {
		t.Errorf("body: got %q, want %q", receivedBody, "5")
	}

	holder := functionStats["myFunc"]
	if holder.LastReport != 5 {
		t.Errorf("LastReport: got %d, want 5", holder.LastReport)
	}
}

func TestReportStats_AlreadyReported(t *testing.T) {
	functionStats = map[string]*statsHolder{
		"myFunc": {Hits: 5, LastReport: 5},
	}
	reportStats()
}

func TestSendStat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	origUrl := config.StatsUrl
	config.StatsUrl = server.URL
	defer func() { config.StatsUrl = origUrl }()

	if !sendStat("testFunc", 10) {
		t.Error("expected success")
	}
}

func TestSendStat_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origUrl := config.StatsUrl
	config.StatsUrl = server.URL
	defer func() { config.StatsUrl = origUrl }()

	if sendStat("testFunc", 10) {
		t.Error("expected failure for 500 response")
	}
}

func TestSendStat_ConnectionError(t *testing.T) {
	origUrl := config.StatsUrl
	config.StatsUrl = "http://localhost:1"
	defer func() { config.StatsUrl = origUrl }()

	if sendStat("testFunc", 10) {
		t.Error("expected failure for connection error")
	}
}

func TestSendStat_InvalidURL(t *testing.T) {
	origUrl := config.StatsUrl
	config.StatsUrl = "://invalid"
	defer func() { config.StatsUrl = origUrl }()

	if sendStat("testFunc", 10) {
		t.Error("expected failure for invalid URL")
	}
}

func TestEnableReporter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	origUrl := config.StatsUrl
	config.StatsUrl = server.URL
	defer func() { config.StatsUrl = origUrl }()

	os.Setenv("STATS_REPORT_INTERVAL", "50ms")
	defer os.Unsetenv("STATS_REPORT_INTERVAL")

	functionStats = map[string]*statsHolder{
		"tickFunc": {Hits: 3, LastReport: 0},
	}

	done := enableReporter()
	time.Sleep(150 * time.Millisecond)
	done <- true

	holder := functionStats["tickFunc"]
	if holder.LastReport != 3 {
		t.Errorf("LastReport: got %d, want 3 (reporter should have reported)", holder.LastReport)
	}
}

func TestReportStats_PartialDue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	origUrl := config.StatsUrl
	config.StatsUrl = server.URL
	defer func() { config.StatsUrl = origUrl }()

	functionStats = map[string]*statsHolder{
		"fn": {Hits: 5, LastReport: 3},
	}

	reportStats()

	if functionStats["fn"].LastReport != 5 {
		t.Errorf("LastReport after report: got %d, want 5", functionStats["fn"].LastReport)
	}
}

func TestSendStat_Status300(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(300)
	}))
	defer server.Close()

	origUrl := config.StatsUrl
	config.StatsUrl = server.URL
	defer func() { config.StatsUrl = origUrl }()

	if sendStat("testFunc", 10) {
		t.Error(fmt.Sprintf("expected failure for 300 response"))
	}
}

func TestEnableRecorder(t *testing.T) {
	// Reset state
	functionStats = map[string]*statsHolder{}
	hitCh = nil
	activeReqCh = nil
	funcInvocations = nil
	funcDuration = nil

	enableRecorder()

	if hitCh == nil {
		t.Fatal("hitCh should be initialized")
	}
	if activeReqCh == nil {
		t.Fatal("activeReqCh should be initialized")
	}
}

func TestIncDecActiveRequests_Enabled(t *testing.T) {
	// Ensure channels are set up
	if activeReqCh == nil {
		activeReqCh = make(chan int, 100)
		go func() {
			for {
				activeRequests += <-activeReqCh
			}
		}()
	}

	IncActiveRequests()
	time.Sleep(10 * time.Millisecond)
	if activeRequests < 1 {
		t.Errorf("activeRequests should be >= 1, got %d", activeRequests)
	}

	DecActiveRequests()
	time.Sleep(10 * time.Millisecond)
}

func TestIncDecActiveRequests_Disabled(t *testing.T) {
	origCh := activeReqCh
	activeReqCh = nil
	defer func() { activeReqCh = origCh }()

	// Should not panic or block
	IncActiveRequests()
	DecActiveRequests()
}

func TestRecordHit_Enabled(t *testing.T) {
	// Ensure recorder is set up
	if hitCh == nil {
		functionStats = map[string]*statsHolder{}
		enableRecorder()
	}

	origEnabled := config.StatsRecorderEnabled
	config.StatsRecorderEnabled = true
	defer func() { config.StatsRecorderEnabled = origEnabled }()

	RecordHit(Invocation{FunctionName: "testFunc", Duration: 100 * time.Millisecond})
	time.Sleep(50 * time.Millisecond)

	stats := GetAllStats()
	if holder, ok := stats["testFunc"]; !ok || holder.Hits < 1 {
		t.Error("expected testFunc to have at least 1 hit after RecordHit")
	}
}

func TestRecord_NewAndExisting(t *testing.T) {
	// Ensure prometheus counters are set up
	if funcInvocations == nil {
		enableRecorder()
	}
	functionStats = map[string]*statsHolder{}

	// First call creates a new holder
	record(Invocation{FunctionName: "newFunc", Duration: 100 * time.Millisecond})
	if holder, ok := functionStats["newFunc"]; !ok || holder.Hits != 1 {
		t.Error("expected newFunc to have 1 hit")
	}

	// Second call increments existing
	record(Invocation{FunctionName: "newFunc", Duration: 200 * time.Millisecond})
	if functionStats["newFunc"].Hits != 2 {
		t.Errorf("expected 2 hits, got %d", functionStats["newFunc"].Hits)
	}
}

func TestReportStats_SendFails(t *testing.T) {
	origUrl := config.StatsUrl
	config.StatsUrl = "http://localhost:1"
	defer func() { config.StatsUrl = origUrl }()

	functionStats = map[string]*statsHolder{
		"fn": {Hits: 5, LastReport: 0},
	}

	reportStats()

	// LastReport should NOT be updated on failure
	if functionStats["fn"].LastReport != 0 {
		t.Errorf("LastReport should remain 0 on failure, got %d", functionStats["fn"].LastReport)
	}
}
