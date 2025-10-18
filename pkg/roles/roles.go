package roles

import (
	"fmt"
	"os"
	"path/filepath"

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
	GetAvailableRoles(groupResource string) ([]string, error)
	GetRoleDefinitions(groupResource string) ([]RoleDefinition, error)
	Reload() error
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

	err := retriever.Reload()
	if err != nil {
		return nil, fmt.Errorf("failed to load roles from %s: %w", filePath, err)
	}

	return retriever, nil
}

// NewDefaultRolesRetriever creates a roles retriever with the default input/roles.yaml path
func NewDefaultRolesRetriever() (*FileBasedRolesRetriever, error) {
	// Get the current working directory and construct the path to input/roles.yaml
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}

	filePath := filepath.Join(cwd, "input", "roles.yaml")
	return NewFileBasedRolesRetriever(filePath)
}

// Reload reloads the roles configuration from the file
func (r *FileBasedRolesRetriever) Reload() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to read roles file: %w", err)
	}

	var config RolesConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal roles YAML: %w", err)
	}

	r.config = &config
	return nil
}

// GetAvailableRoles returns the list of available role IDs for a given group resource
func (r *FileBasedRolesRetriever) GetAvailableRoles(groupResource string) ([]string, error) {
	if r.config == nil {
		return nil, fmt.Errorf("roles configuration not loaded")
	}

	for _, groupRoles := range r.config.Roles {
		if groupRoles.GroupResource == groupResource {
			var roleIDs []string
			for _, role := range groupRoles.Roles {
				roleIDs = append(roleIDs, role.ID)
			}
			return roleIDs, nil
		}
	}

	// Return empty slice if no roles found for this group resource
	return []string{}, nil
}

// GetRoleDefinitions returns the full role definitions for a given group resource
func (r *FileBasedRolesRetriever) GetRoleDefinitions(groupResource string) ([]RoleDefinition, error) {
	if r.config == nil {
		return nil, fmt.Errorf("roles configuration not loaded")
	}

	for _, groupRoles := range r.config.Roles {
		if groupRoles.GroupResource == groupResource {
			return groupRoles.Roles, nil
		}
	}

	// Return empty slice if no roles found for this group resource
	return []RoleDefinition{}, nil
}
