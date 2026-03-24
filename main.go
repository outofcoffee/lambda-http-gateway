package main

import (
	"context"
	"fmt"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
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
	region                = config.GetRegion()
	requestIdHeader       = config.GetRequestIdHeader()
	routingMode           = config.GetRoutingMode()
	baseDomain            = config.GetBaseDomain()
	functionPrefix        = config.GetFunctionPrefix()
	permissiveCorsEnabled = config.IsPermissiveCorsEnabled()
	version               = "dev"
	lambdaClient          *lambda.Client
)

func main() {
	logrus.SetLevel(config.GetConfigLevel())
	validateConfig()

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(region),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load AWS config: %v", err))
	}
	lambdaClient = lambda.NewFromConfig(cfg)

	stats.Init()

	http.Handle("/system/metrics", promhttp.Handler())
	http.HandleFunc("/system/status", statusHandler)
	http.HandleFunc("/", handler)

	port := config.GetPort()
	logrus.Infof("starting http lambda gateway %v for region %v on port %v (routing: %v)", version, region, port, routingMode)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
