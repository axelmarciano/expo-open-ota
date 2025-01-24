package services

import "github.com/golang-jwt/jwt/v5"

func GenerateJWTToken(secret string, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func DecodeAndExtractJWTToken(secret string, tokenString string, claims jwt.Claims) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}
	return token, nil
}
