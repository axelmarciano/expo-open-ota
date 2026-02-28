package helpers

import (
	"expo-open-ota/internal/types"
	"net/http"
	"strings"
)

func GetEoasAuth(r *http.Request) types.EoasAuth {
	bearerToken, _ := GetBearerToken(r)
	if bearerToken != "" {
		return types.EoasAuth{
			Token: &bearerToken,
		}
	}
	return types.EoasAuth{}
}

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
