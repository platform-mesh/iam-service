package utils

import (
	"strings"

	"github.com/openmfp/iam-service/pkg/graph"
)

func CheckRolesFilter(s string, rolesfilter []*graph.RoleInput) bool {
	contains := false
	for _, filterRole := range rolesfilter {
		if strings.Contains(s, filterRole.TechnicalName) {
			contains = true
			break
		}
	}
	return contains
}
