package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
)

func sendResponse(log *logrus.Entry, w http.ResponseWriter, headers *map[string]string, statusCode int, body *[]byte, client string) (err error) {
	for responseHeaderKey, responseHeaderValue := range *headers {
		w.Header().Add(responseHeaderKey, responseHeaderValue)
	}
	w.WriteHeader(statusCode)
	_, err = w.Write(*body)
	if err != nil {
		return fmt.Errorf("error writing response: %v", err)
	}

	log.Debugf("wrote response [code: %v, body %v bytes] to client %v", statusCode, len(*body), client)
	return nil
}
