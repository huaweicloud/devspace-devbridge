package api

import (
	"errors"
	"net/http"

	"huawei.com/devbridge/internal/auth"
)

const (
	headerXAPIKey         = "X-API-Key"
	headerContentType     = "content-type"
	headerApplicationJson = "application/json"
)

var (
	errMissingAPIKey = errors.New("missing api key")
)

func signRequest(cred *auth.Credential, req *http.Request) error {
	if cred == nil || cred.APIKey == "" {
		return errMissingAPIKey
	}

	req.Header.Set(headerXAPIKey, cred.APIKey)
	if req.Header.Get(headerContentType) == "" {
		req.Header.Set(headerContentType, headerApplicationJson)
	}
	return nil
}
