package utils_test

import (
	"testing"

	"github.com/openmfp/iam-service/internal/pkg/utils"
	"github.com/openmfp/iam-service/pkg/graph"
)

func TestCheckRolesFilter(t *testing.T) {
	tests := []struct {
		name        string
		inputString string
		roles       []*graph.RoleInput
		want        bool
	}{
		{
			name:        "no roles in filter returns false",
			inputString: "some text",
			roles:       []*graph.RoleInput{},
			want:        false,
		},
		{
			name:        "role matches substring",
			inputString: "this is an admin account",
			roles: []*graph.RoleInput{
				{TechnicalName: "admin"},
			},
			want: true,
		},
		{
			name:        "role does not match substring",
			inputString: "user account",
			roles: []*graph.RoleInput{
				{TechnicalName: "admin"},
			},
			want: false,
		},
		{
			name:        "multiple roles, one matches",
			inputString: "example with manager role",
			roles: []*graph.RoleInput{
				{TechnicalName: "admin"},
				{TechnicalName: "manager"},
			},
			want: true,
		},
		{
			name:        "multiple roles, none match",
			inputString: "example with no valid roles",
			roles: []*graph.RoleInput{
				{TechnicalName: "admin"},
				{TechnicalName: "manager"},
			},
			want: false,
		},
		{
			name:        "case sensitivity test",
			inputString: "Example With ADMIN Role",
			roles: []*graph.RoleInput{
				{TechnicalName: "admin"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.CheckRolesFilter(tt.inputString, tt.roles)
			if got != tt.want {
				t.Errorf("CheckRolesFilter(%q, roles) = %v; want %v", tt.inputString, got, tt.want)
			}
		})
	}
}
