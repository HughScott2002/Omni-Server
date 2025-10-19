package models

import (
	"time"

	"golang.org/x/time/rate"
)

type Visitor struct {
	Limiter  *rate.Limiter
	LastSeen time.Time
}
