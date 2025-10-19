package models

import (
	"time"
)

type Session struct {
	ID          string    `json:"id"`
	UserEmail   string    `json:"userEmail"`
	DeviceInfo  string    `json:"deviceInfo"`
	IPAddress   string    `json:"ipAddress"`
	Country     string    `json:"country"`
	Browser     string    `json:"browser"`
	Token       string    `json:"token"`
	LastLoginAt time.Time `json:"lastLoginAt"`
	CreatedAt   time.Time `json:"createdAt"`
}
