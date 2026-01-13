package storage

import (
	"testing"
)

func TestCloseNil(t *testing.T) {
	// Force db to nil
	oldDB := db
	db = nil
	defer func() { db = oldDB }()

	if err := Close(); err != nil {
		t.Errorf("Expected nil error for nil db, got %v", err)
	}
}
