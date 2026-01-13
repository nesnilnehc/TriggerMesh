package models

import (
	"time"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	APIKey    string    `json:"api_key"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	JobName   string    `json:"job_name"`
	Params    string    `json:"params"`
	Result    string    `json:"result"`
	Error     string    `json:"error,omitempty"`
}
