package auth

import (
	"crypto/rand"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"lighthouse.uni-kiel.de/lighthouse-server/config"
)

var (
	jwtPrivateKey []byte = []byte(config.GetString("JWT_PRIVATE_KEY", NewRandomKey()))
)

// --- JWT Authentication ---

// ValidateJWT parses and validates an HMAC signed JWT and returns its claims or an error if invalid
func ValidateJWT(tokenStr string) (map[string]interface{}, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtPrivateKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, fmt.Errorf("invalid JWT")
	}
}

// Generates a random 32 byte key in case the config is empty (no one being able to authenticate is better than an empty key)
func NewRandomKey() string {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic(err)
	}
	return string(key)
}
