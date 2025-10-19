package utils

// import (
// 	"time"

// 	"github.com/golang-jwt/jwt/v4"
// )

// func GenerateRefreshToken(email string) (string, error) {
// 	expirationTime := time.Now().Add(7 * 24 * time.Hour)
// 	claims := &jwt.RegisteredClaims{
// 		Subject:   email,
// 		ExpiresAt: jwt.NewNumericDate(expirationTime),
// 	}

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString(jwtKey)
// }
