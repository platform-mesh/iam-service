package core

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/openmfp/iam-service/pkg/layers/db"
	"github.com/openmfp/iam-service/pkg/mocks"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		db    db.Provider
		error error
	}{
		{
			name:  "new_OK",
			db:    &mocks.DB{},
			error: nil,
		},
		{
			name:  "db_not_set",
			db:    nil,
			error: errors.New("db not set"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.db, nil, nil)
			assert.Equal(t, tt.error, err)
		})
	}
}
