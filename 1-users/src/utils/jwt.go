package utils

import "os"

func getJWTKey() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-secret-key"
	}
	return []byte(secret)
}

var jwtKey = getJWTKey()
