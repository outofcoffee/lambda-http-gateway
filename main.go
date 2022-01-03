package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"httplambda/config"
	"io/ioutil"
	"net/http"
	"strings"
)

func main() {
	logrus.SetLevel(config.GetConfigLevel())
	port := config.GetPort()

	http.HandleFunc("/", handler)

	logrus.Infof("starting http lambda gateway on port %v", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}

func handler(w http.ResponseWriter, req *http.Request) {
	requestIdHeader := config.GetRequestIdHeader()

	log := logrus.WithField("requestId", getRequestId(requestIdHeader, req))

	client := req.RemoteAddr
	log.Debugf("received request %v %v from client %v", req.Method, req.URL, client)

	functionName, path, requestHeaders, requestBody, err := parseRequest(req)
	if err != nil {
		log.Error(err)
		return
	}

	code, body, responseHeaders, err := invoke(log, functionName, req.Method, path, requestHeaders, requestBody)
	if err != nil {
		log.Error(err)
		return
	}

	err = sendResponse(log, w, responseHeaders, code, body, client)
	if err != nil {
		log.Error(err)
	}

	log.Infof("proxied request to %v [code: %v, body %v bytes] for client %v", functionName, code, len(body), client)
}

func getRequestId(headerName string, req *http.Request) string {
	var requestId string
	if headerName != "" {
		requestId = req.Header.Get(headerName)
	}
	if requestId == "" {
		requestId = uuid.NewString()
	}
	return requestId
}

func parseRequest(req *http.Request) (string, string, map[string]string, []byte, error) {
	splitPath := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/"), "/", 2)
	functionName := splitPath[0]
	path := "/" + splitPath[1]

	requestHeaders := make(map[string]string)
	for requestHeaderKey, requestHeaderValue := range req.Header {
		requestHeaders[requestHeaderKey] = requestHeaderValue[0]
	}

	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", "", map[string]string{}, nil, fmt.Errorf("error parsing request body: %v", err)
	}
	return functionName, path, requestHeaders, requestBody, err
}

func invoke(
	log *logrus.Entry,
	functionName string,
	httpMethod string,
	path string,
	requestHeaders map[string]string,
	requestBody []byte,
) (statusCode int, body []byte, responseHeaders map[string]string, err error) {
	log.Debugf("invoking function %v with %v %v [body: %v bytes]", functionName, httpMethod, path, len(requestBody))

	// Create Lambda service client
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := lambda.New(sess, &aws.Config{Region: aws.String("eu-west-1")})

	encodedBody := b64.StdEncoding.EncodeToString(requestBody)
	request := events.APIGatewayProxyRequest{
		HTTPMethod:      httpMethod,
		Path:            path,
		Headers:         requestHeaders,
		Body:            encodedBody,
		IsBase64Encoded: true,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("error marshalling request: %v", err)
	}

	result, err := client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(functionName), Payload: payload})
	if err != nil {
		return 0, nil, nil, fmt.Errorf("error calling %v: %v", functionName, err)
	}

	var resp events.APIGatewayProxyResponse

	err = json.Unmarshal(result.Payload, &resp)
	if err != nil || resp.StatusCode == 0 {
		return 0, nil, nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	var responseBody []byte
	if resp.IsBase64Encoded {
		responseBody, err = b64.StdEncoding.DecodeString(resp.Body)
		if err != nil {
			return 0, nil, nil, fmt.Errorf("error decoding body %v: %v", resp.Body, err)
		}
	} else {
		responseBody = []byte(resp.Body)
	}

	log.Debugf("received response from function %v [code: %v, body: %v bytes]", functionName, resp.StatusCode, len(responseBody))
	return resp.StatusCode, responseBody, resp.Headers, nil
}

func sendResponse(log *logrus.Entry, w http.ResponseWriter, responseHeaders map[string]string, statusCode int, body []byte, client string) (err error) {
	for responseHeaderKey, responseHeaderValue := range responseHeaders {
		w.Header().Add(responseHeaderKey, responseHeaderValue)
	}
	w.WriteHeader(statusCode)
	_, err = w.Write(body)
	if err != nil {
		return fmt.Errorf("error writing response: %v", err)
	}

	log.Debugf("wrote response [code: %v, body %v bytes] to client %v", statusCode, len(body), client)
	return nil
}
