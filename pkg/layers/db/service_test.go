package db

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		db    *gorm.DB
		error error
	}{
		{
			name:  "new_OK",
			db:    &gorm.DB{},
			error: nil,
		},
		{
			name:  "conn_not_set",
			db:    nil,
			error: errors.New("connection is nil"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.db)
			assert.Equal(t, tt.error, err)
		})
	}
}
