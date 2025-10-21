package entity

import "time"

type ErrorInfo struct {
	ID      string
	Code    string
	Message string
	Details map[string]string

	Service   string
	Operation string

	Frequency        int
	FrequencyMinutes int

	CreatedAt time.Time
	Alerted   bool
}
