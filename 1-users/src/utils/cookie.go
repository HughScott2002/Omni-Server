package utils

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func getCookieDomain() string {
	domain := os.Getenv("COOKIE_DOMAIN")
	if domain == "" {
		domain = "localhost"
	}
	return domain
}

// ExtractBearerToken pulls a token from the Authorization header.
// Returns empty string if not present.
func ExtractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// func setCookie(w http.ResponseWriter, name, value string, maxAge int) {
// 	isSecure := os.Getenv("ENVIRONMENT") == "production" || os.Getenv("ENVIRONMENT") == "prod"

//		http.SetCookie(w, &http.Cookie{
//			Name:     name,
//			Value:    value,
//			HttpOnly: true,
//			Secure:   isSecure, // Only set to true when using HTTPS
//			SameSite: http.SameSiteLaxMode,
//			Path:     "/",
//			MaxAge:   maxAge,
//		})
//	}
func SetCookie(w http.ResponseWriter, name, value string, maxAge int) {
	isSecure := os.Getenv("ENVIRONMENT") == "production" || os.Getenv("ENVIRONMENT") == "prod"

	fmt.Printf("Setting cookie: %s, MaxAge: %d\n", name, maxAge)
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Secure:   isSecure, // Only set to true in production (HTTPS)
		SameSite: http.SameSiteLaxMode,
		Path:     "/", // Set path to root for all cookies
		MaxAge:   maxAge,
		Domain:   getCookieDomain(),
	})
}
