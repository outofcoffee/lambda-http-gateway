package config

import (
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

var (
	StatsRecorderEnabled = isStatsRecorderEnabled()
	StatsReporterEnabled = isStatsReporterEnabled()
	StatsUrl             = getStatsUrl()
)

func GetConfigLevel() logrus.Level {
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = logrus.DebugLevel
	}
	return level
}

func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	return port
}

func GetRegion() string {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "eu-west-1"
	}
	return region
}

func GetRequestIdHeader() string {
	return os.Getenv("REQUEST_ID_HEADER")
}

func isStatsRecorderEnabled() bool {
	return os.Getenv("STATS_RECORDER") == "true" || isStatsReporterEnabled()
}

func getStatsUrl() string {
	return os.Getenv("STATS_REPORT_URL")
}

func GetStatsInterval() time.Duration {
	var seconds time.Duration
	interval := os.Getenv("STATS_REPORT_INTERVAL")
	if interval == "" {
		seconds = 5 * time.Second
	} else {
		seconds, _ = time.ParseDuration(interval)
	}
	return seconds
}

func isStatsReporterEnabled() bool {
	return StatsUrl != ""
}
