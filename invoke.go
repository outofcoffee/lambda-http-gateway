package main

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/sirupsen/logrus"
)

func invoke(
	log *logrus.Entry,
	functionName string,
	httpMethod string,
	path string,
	requestHeaders *map[string]string,
	requestBody *[]byte,
) (statusCode int, responseBody *[]byte, responseHeaders *map[string]string, err error) {
	log.Debugf("invoking function %v with %v %v [body: %v bytes]", functionName, httpMethod, path, len(*requestBody))

	encodedBody := b64.StdEncoding.EncodeToString(*requestBody)
	request := events.APIGatewayProxyRequest{
		HTTPMethod:      httpMethod,
		Path:            path,
		Headers:         *requestHeaders,
		Body:            encodedBody,
		IsBase64Encoded: true,
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("error marshalling request: %v", err)
	}

	result, err := lambdaClient.Invoke(context.Background(), &lambda.InvokeInput{FunctionName: &functionName, Payload: payload})
	if err != nil {
		return 0, nil, nil, fmt.Errorf("error calling %v: %v", functionName, err)
	}

	var resp events.APIGatewayProxyResponse

	err = json.Unmarshal(result.Payload, &resp)
	statusCode = resp.StatusCode
	if err != nil || statusCode == 0 {
		return statusCode, nil, nil, fmt.Errorf("error unmarshalling response: %v", err)
	}

	var respBody []byte
	if resp.IsBase64Encoded {
		respBody, err = b64.StdEncoding.DecodeString(resp.Body)
		if err != nil {
			return statusCode, nil, nil, fmt.Errorf("error decoding body %v: %v", resp.Body, err)
		}
	} else {
		respBody = []byte(resp.Body)
	}

	log.Debugf("received response from function %v [code: %v, body: %v bytes]", functionName, statusCode, len(respBody))
	return statusCode, &respBody, &resp.Headers, nil
}
