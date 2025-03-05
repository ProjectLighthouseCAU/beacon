package jwt

import (
	"crypto/rand"
	"fmt"
	"log"

	"github.com/ProjectLighthouseCAU/beacon/config"
	"github.com/golang-jwt/jwt/v4"
)

var (
	jwtPrivateKey []byte = setJWTPrivateKey()
)

func setJWTPrivateKey() []byte {
	if len(config.JWTPrivateKey) > 0 {
		return config.JWTPrivateKey
	}
	return NewRandomKey()
}

// --- JWT Authentication ---

// ValidateJWT parses and validates an HMAC signed JWT and returns its claims or an error if invalid
func ValidateJWT(tokenStr string) (map[string]any, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
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
func NewRandomKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Println(err)
	}
	return key
}
