package stats

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"lambdahttpgw/config"
	"time"
)

type Invocation struct {
	FunctionName string
	Duration     time.Duration
}

type statsHolder struct {
	Hits       int64 `json:"hits"`
	LastReport int64 `json:"lastReport"`
}

var (
	functionStats   = map[string]*statsHolder{}
	hitCh           chan Invocation
	activeReqCh     chan int
	funcInvocations *prometheus.CounterVec
	funcDuration    *prometheus.CounterVec
	activeRequests  int
)

// enableRecorder starts a goroutine that ensures single concurrency
// when mutating functionStats.
func enableRecorder() {
	logrus.Debugf("enabling stats recorder")

	funcInvocations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "functions_invoked_count",
		Help: "Total number of invocations (per function).",
	}, []string{"function"})

	funcDuration = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "functions_duration_sum",
		Help: "Sum of invocation durations in seconds (per function).",
	}, []string{"function"})

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "active_requests",
		Help: "Number of currently active requests.",
	}, func() float64 {
		return float64(activeRequests)
	})

	// buffer to reduce likelihood of blocking caller
	hitCh = make(chan Invocation, 100)
	go func() {
		for true {
			invocation := <-hitCh
			record(invocation)
		}
	}()

	activeReqCh = make(chan int, 100)
	go func() {
		for true {
			activeRequests += <-activeReqCh
		}
	}()
}

func record(invocation Invocation) {
	holder, exist := functionStats[invocation.FunctionName]
	if !exist {
		holder = &statsHolder{
			Hits:       0,
			LastReport: 0,
		}
		functionStats[invocation.FunctionName] = holder
	}
	holder.Hits++
	funcInvocations.WithLabelValues(invocation.FunctionName).Inc()
	funcDuration.WithLabelValues(invocation.FunctionName).Add(invocation.Duration.Seconds())
}

func RecordHit(invocation Invocation) {
	if !config.StatsRecorderEnabled {
		return
	}
	hitCh <- invocation
}

func GetAllStats() map[string]*statsHolder {
	return functionStats
}

func IncActiveRequests() {
	if activeReqCh == nil {
		return
	}
	activeReqCh <- 1
}

func DecActiveRequests() {
	if activeReqCh == nil {
		return
	}
	activeReqCh <- -1
}
