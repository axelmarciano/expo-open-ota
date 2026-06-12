package helpers

import (
	"expo-open-ota/internal/types"
	"net/http"
	"strings"
)

func GetAuth(r *http.Request) types.Auth {
	bearerToken, _ := GetBearerToken(r)
	if bearerToken != "" {
		return types.Auth{
			Token: &bearerToken,
		}
	}
	sessionSecret := r.Header.Get("expo-session")
	if sessionSecret != "" {
		return types.Auth{
			SessionSecret: &sessionSecret,
		}
	}
	return types.Auth{}
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
