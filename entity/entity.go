package entity

import "time"

type ErrorInfo struct {
	ID      string
	Code    string
	Message string
	Details map[string]string

	Service   string
	Operation string

	CreatedAt time.Time
	Alerted   bool
}
