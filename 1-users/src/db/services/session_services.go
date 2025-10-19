package services

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"omni/src/db"
	"omni/src/models"
	"github.com/google/uuid"
)

// Parse browser info from User-Agent
func ParseBrowser(userAgent string) string {
	userAgent = strings.ToLower(userAgent)
	for _, browser := range models.BrowserPatterns {
		if strings.Contains(userAgent, strings.ToLower(browser.Pattern)) {
			return browser.Name
		}
	}
	return "Unknown Browser"
}

// Format time for session display
func FormatSessionTime(t time.Time) string {
	now := time.Now()
	if t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day() {
		return "Current Session"
	}
	return t.Format("Jan 2 at 3:04 PM")
}

// Create and save a new session
func CreateSession(r *http.Request, email string) (*models.Session, error) {
	// Get the IP address
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	// Get the User-Agent
	userAgent := r.Header.Get("User-Agent")

	// Create a new session
	session := &models.Session{
		ID:          uuid.New().String(),
		UserEmail:   email,
		DeviceInfo:  userAgent,
		IPAddress:   ipAddress,
		Browser:     ParseBrowser(userAgent),
		Country:     "United States", // TODO: Use a geo-IP service here
		Token:       "",              //TODO:  Set this as needed
		LastLoginAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	// Save the session
	err := db.AddSession(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}
func CreateUserSession(r *http.Request, user *models.User, refreshToken string) (*models.Session, error) {
	// Get the IP address
	ipAddress := r.Header.Get("X-Forwarded-For")
	if ipAddress == "" {
		ipAddress = r.RemoteAddr
	}

	// Get the User-Agent
	userAgent := r.Header.Get("User-Agent")

	// Create a new session
	session := &models.Session{
		ID:          uuid.New().String(),
		UserEmail:   user.Email,
		DeviceInfo:  userAgent,
		IPAddress:   ipAddress,
		Browser:     ParseBrowser(userAgent),
		Country:     "United States", //TODO Use a geo-IP service here
		Token:       refreshToken,
		LastLoginAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	// Save the session
	if err := db.AddSession(session); err != nil {
		return nil, fmt.Errorf("failed to add session: %w", err)
	}

	// Add refresh token info
	if err := db.AddRefreshToken(refreshToken, db.RefreshTokenInfo{
		UserEmail:  user.Email,
		DeviceInfo: userAgent,
		CreatedAt:  time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("failed to add refresh token: %w", err)
	}

	return session, nil
}
