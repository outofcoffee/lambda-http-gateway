package main

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"lambdahttpgw/config"
	"lambdahttpgw/stats"
	"net/http"
)

func validateConfig() {
	if routingMode == "subdomain" && baseDomain == "" {
		panic(fmt.Sprintf("BASE_DOMAIN must be set when using subdomain routing mode"))
	}
}

var (
	region          = config.GetRegion()
	requestIdHeader = config.GetRequestIdHeader()
	routingMode     = config.GetRoutingMode()
	baseDomain      = config.GetBaseDomain()
	functionPrefix  = config.GetFunctionPrefix()
	version         = "dev"
)

func main() {
	logrus.SetLevel(config.GetConfigLevel())
	validateConfig()
	stats.Init()

	http.Handle("/system/metrics", promhttp.Handler())
	http.HandleFunc("/system/status", statusHandler)
	http.HandleFunc("/", handler)

	port := config.GetPort()
	logrus.Infof("starting http lambda gateway %v for region %v on port %v (routing: %v)", version, region, port, routingMode)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
