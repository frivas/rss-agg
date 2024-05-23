package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	authVal := headers.Get("Authorization")
	fmt.Println(headers)
	if authVal == "" {
		return "", errors.New("An API Key is needed to process this request")
	}
	authVals := strings.Split(authVal, " ")
	if len(authVals) != 2 {
		return "", errors.New("Malformed auth header")
	}
	if authVals[0] != "ApiKey" {
		return "", errors.New("Malformed auth header")
	}
	fmt.Println(authVals[1])
	return authVals[1], nil
}
