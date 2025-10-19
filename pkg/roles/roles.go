package roles

import (
	"os"
	"path/filepath"

	"github.com/platform-mesh/golang-commons/errors"
	"gopkg.in/yaml.v3"
)

// RoleDefinition represents a single role definition
type RoleDefinition struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"displayName"`
	Description string `yaml:"description"`
}

// GroupResourceRoles represents roles for a specific group resource
type GroupResourceRoles struct {
	GroupResource string           `yaml:"groupResource"`
	Roles         []RoleDefinition `yaml:"roles"`
}

// RolesConfig represents the entire roles configuration
type RolesConfig struct {
	Roles []GroupResourceRoles `yaml:"roles"`
}

// RolesRetriever interface for retrieving roles
type RolesRetriever interface {
	GetRoleDefinitions(groupResource string) ([]RoleDefinition, error)
}

// FileBasedRolesRetriever implements RolesRetriever by reading from a YAML file
type FileBasedRolesRetriever struct {
	filePath string
	config   *RolesConfig
}

// NewFileBasedRolesRetriever creates a new file-based roles retriever
func NewFileBasedRolesRetriever(filePath string) (*FileBasedRolesRetriever, error) {
	retriever := &FileBasedRolesRetriever{
		filePath: filePath,
	}

	err := retriever.reload()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load roles from %s", filePath)
	}

	return retriever, nil
}

// NewDefaultRolesRetriever creates a roles retriever with the default input/roles.yaml path
func NewDefaultRolesRetriever() (*FileBasedRolesRetriever, error) {
	// Get the current working directory and construct the path to input/roles.yaml
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current working directory")
	}

	filePath := filepath.Join(cwd, "input", "roles.yaml")
	return NewFileBasedRolesRetriever(filePath)
}

// reload reloads the roles configuration from the file
func (r *FileBasedRolesRetriever) reload() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return errors.Wrap(err, "failed to read roles file %s", r.filePath)
	}

	var config RolesConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal roles YAML from file %s", r.filePath)
	}

	r.config = &config
	return nil
}

// GetRoleDefinitions returns the full role definitions for a given group resource
func (r *FileBasedRolesRetriever) GetRoleDefinitions(groupResource string) ([]RoleDefinition, error) {
	if r.config == nil {
		return nil, errors.New("roles configuration not loaded")
	}

	for _, groupRoles := range r.config.Roles {
		if groupRoles.GroupResource == groupResource {
			return groupRoles.Roles, nil
		}
	}

	// Return empty slice if no roles found for this group resource
	return []RoleDefinition{}, nil
}

// GetAvailableRoleIDs is a helper function that extracts role IDs from role definitions
func GetAvailableRoleIDs(roleDefinitions []RoleDefinition) []string {
	roleIDs := make([]string, len(roleDefinitions))
	for i, role := range roleDefinitions {
		roleIDs[i] = role.ID
	}
	return roleIDs
}
