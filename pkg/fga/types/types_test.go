package types

import (
	"reflect"
	"testing"
)

func TestRole_String(t *testing.T) {
	tests := []struct {
		name string
		role Role
		want string
	}{
		{
			name: "Member role",
			role: Member,
			want: "member",
		},
		{
			name: "VaultMaintainer role",
			role: VaultMaintainer,
			want: "vault_maintainer",
		},
		{
			name: "Owner role",
			role: Owner,
			want: "owner",
		},
		{
			name: "Invalid role",
			role: Invalid,
			want: "",
		},
		{
			name: "Unknown role",
			role: Role(999),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.String(); got != tt.want {
				t.Errorf("Role.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRole(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want Role
	}{
		{
			name: "Parse member",
			s:    "member",
			want: Member,
		},
		{
			name: "Parse vault_maintainer",
			s:    "vault_maintainer",
			want: VaultMaintainer,
		},
		{
			name: "Parse owner",
			s:    "owner",
			want: Owner,
		},
		{
			name: "Parse invalid role",
			s:    "invalid_role",
			want: Invalid,
		},
		{
			name: "Parse empty string",
			s:    "",
			want: Invalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseRole(tt.s); got != tt.want {
				t.Errorf("ParseRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllRoles(t *testing.T) {
	want := []Role{Member, VaultMaintainer, Owner}
	got := AllRoles()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AllRoles() = %v, want %v", got, want)
	}
}

func TestAllRoleStrings(t *testing.T) {
	want := []string{"member", "vault_maintainer", "owner"}
	got := AllRoleStrings()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("AllRoleStrings() = %v, want %v", got, want)
	}
}
