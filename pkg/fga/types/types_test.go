package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRole_String(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected string
	}{
		{
			name:     "Member role",
			role:     Member,
			expected: "member",
		},
		{
			name:     "VaultMaintainer role",
			role:     VaultMaintainer,
			expected: "vault_maintainer",
		},
		{
			name:     "Owner role",
			role:     Owner,
			expected: "owner",
		},
		{
			name:     "Invalid role",
			role:     Invalid,
			expected: "",
		},
		{
			name:     "Unknown role",
			role:     Role(999),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.role.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseRole(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Role
	}{
		{
			name:     "Parse member",
			input:    "member",
			expected: Member,
		},
		{
			name:     "Parse vault_maintainer",
			input:    "vault_maintainer",
			expected: VaultMaintainer,
		},
		{
			name:     "Parse owner",
			input:    "owner",
			expected: Owner,
		},
		{
			name:     "Parse invalid string",
			input:    "invalid_role",
			expected: Invalid,
		},
		{
			name:     "Parse empty string",
			input:    "",
			expected: Invalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseRole(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAllRoles(t *testing.T) {
	roles := AllRoles()
	expected := []Role{Member, VaultMaintainer, Owner}

	assert.Equal(t, expected, roles)
	assert.Len(t, roles, 3)
}

func TestAllRoleStrings(t *testing.T) {
	roleStrings := AllRoleStrings()
	expected := []string{"member", "vault_maintainer", "owner"}

	assert.Equal(t, expected, roleStrings)
	assert.Len(t, roleStrings, 3)
}

func TestUserIDToRoles(t *testing.T) {
	// Test that UserIDToRoles can be created and used
	userRoles := make(UserIDToRoles)
	userRoles["user1"] = []string{"member", "owner"}
	userRoles["user2"] = []string{"vault_maintainer"}

	assert.Equal(t, []string{"member", "owner"}, userRoles["user1"])
	assert.Equal(t, []string{"vault_maintainer"}, userRoles["user2"])
	assert.Len(t, userRoles, 2)
}
