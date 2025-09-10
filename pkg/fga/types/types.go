package types

type UserIDToRoles map[string][]string

type Role int

const (
	Invalid Role = -1
	Member  Role = iota
	VaultMaintainer
	Owner
)

func (r Role) String() string {
	switch r {
	case Member:
		return "member"
	case VaultMaintainer:
		return "vault_maintainer"
	case Owner:
		return "owner"
	default:
		return ""
	}
}

func ParseRole(s string) Role {
	switch s {
	case "member":
		return Member
	case "vault_maintainer":
		return VaultMaintainer
	case "owner":
		return Owner
	default:
		return Invalid
	}
}

func AllRoles() []Role {
	return []Role{Member, VaultMaintainer, Owner}
}

func AllRoleStrings() []string {
	roles := AllRoles()
	result := make([]string, len(roles))
	for i, role := range roles {
		result[i] = role.String()
	}
	return result
}
