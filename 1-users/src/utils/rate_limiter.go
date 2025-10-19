package utils

import (
	"time"

	"omni/src/db"
	"omni/src/models"
	"golang.org/x/time/rate"
)

// Retrieve and return the rate limiter for the current visitor if it
// already exists. Otherwise create a new rate limiter and add it to
// the visitors map, using the IP address as the key.
func GetVisitor(ip string) *rate.Limiter {
	db.Mu.Lock()
	defer db.Mu.Unlock()

	v, exists := db.Visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(1, 3) // Allow 1 request per second with burst of 3
		// Include the current time when creating a new visitor.
		db.Visitors[ip] = &models.Visitor{Limiter: limiter, LastSeen: time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.LastSeen = time.Now()
	return v.Limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func CleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		db.Mu.Lock()
		for ip, v := range db.Visitors {
			if time.Since(v.LastSeen) > 3*time.Minute {
				delete(db.Visitors, ip)
			}
		}
		db.Mu.Unlock()
	}
}

func init() {
	go CleanupVisitors()
}
