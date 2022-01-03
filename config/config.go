package config

import (
	"github.com/sirupsen/logrus"
	"os"
)

func GetConfigLevel() logrus.Level {
	level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		level = logrus.DebugLevel
	}
	return level
}

func GetPort() string {
	port, success := os.LookupEnv("PORT")
	if !success {
		port = "8090"
	}
	return port
}

func GetRequestIdHeader() string {
	return os.Getenv("REQUEST_ID_HEADER")
}
