package helpers

import (
	"net/http"
	"strings"
)

func GetBearerToken(r *http.Request) (string, error) {
	bearerToken := r.Header.Get("Authorization")
	if bearerToken == "" {
		return "", nil
	}
	tokens := strings.Split(bearerToken, "Bearer ")
	if len(tokens) != 2 {
		return "", nil
	}
	return tokens[1], nil
}
