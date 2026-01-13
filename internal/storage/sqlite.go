package storage

import (
	"database/sql"
	"time"

	"triggermesh/internal/logger"
	"triggermesh/internal/storage/models"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Init initializes the SQLite database
func Init(dbPath string) error {
	var err error

	// Open the database connection with connection pool settings
	db, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=ON")
	if err != nil {
		return err
	}

	// Configure connection pool
	// SQLite doesn't support multiple writers, but we can optimize for concurrent reads
	db.SetMaxOpenConns(25)                 // Maximum number of open connections
	db.SetMaxIdleConns(5)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum connection lifetime

	// Test the connection
	if err = db.Ping(); err != nil {
		return err
	}

	// Create the audit log table if it doesn't exist
	if err = createTables(); err != nil {
		return err
	}

	logger.Info("Database initialized successfully")
	return nil
}

// createTables creates the necessary database tables
func createTables() error {
	// Create audit log table
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		api_key TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status INTEGER NOT NULL,
		job_name TEXT,
		params TEXT,
		result TEXT,
		error TEXT
	)
	`)

	return err
}

// InsertAuditLog inserts a new audit log entry
func InsertAuditLog(log models.AuditLog) error {
	// Format timestamp as RFC3339 for better precision
	timestampStr := log.Timestamp.Format("2006-01-02 15:04:05.000000")
	_, err := db.Exec(
		`INSERT INTO audit_logs (timestamp, api_key, method, path, status, job_name, params, result, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		timestampStr,
		log.APIKey,
		log.Method,
		log.Path,
		log.Status,
		log.JobName,
		log.Params,
		log.Result,
		log.Error,
	)

	if err != nil {
		logger.Error("Failed to insert audit log", "error", err)
		return err
	}

	return nil
}

// GetAuditLogs retrieves audit logs with pagination
func GetAuditLogs(limit, offset int) ([]models.AuditLog, error) {
	rows, err := db.Query(
		`SELECT id, timestamp, api_key, method, path, status, job_name, params, result, error FROM audit_logs ORDER BY id DESC LIMIT ? OFFSET ?`,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		var timestampStr string

		// Scan the row into the log struct
		if err := rows.Scan(
			&log.ID,
			&timestampStr,
			&log.APIKey,
			&log.Method,
			&log.Path,
			&log.Status,
			&log.JobName,
			&log.Params,
			&log.Result,
			&log.Error,
		); err != nil {
			return nil, err
		}

		// Parse the timestamp string into time.Time
		// Try multiple formats for compatibility
		var timestamp time.Time
		var err error

		// Try with microseconds first
		timestamp, err = time.Parse("2006-01-02 15:04:05.000000", timestampStr)
		if err != nil {
			// Try without microseconds
			timestamp, err = time.Parse("2006-01-02 15:04:05", timestampStr)
			if err != nil {
				// If parsing fails, use current time as fallback
				timestamp = time.Now()
			}
		}
		log.Timestamp = timestamp

		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// Close closes the database connection
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
