package roles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileBasedRolesRetriever_Success(t *testing.T) {
	// Create a temporary YAML file
	content := `roles:
  - groupResource: core_platform-mesh_io_account
    roles:
      - id: owner
        displayName: Owner
        description: Full access
      - id: member
        displayName: Member
        description: Limited access
  - groupResource: apps.v1/deployments
    roles:
      - id: admin
        displayName: Admin
        description: Admin access`

	tmpFile := createTempYAMLFile(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	retriever, err := NewFileBasedRolesRetriever(tmpFile)

	require.NoError(t, err)
	assert.NotNil(t, retriever)
	assert.NotNil(t, retriever.config)
	assert.Len(t, retriever.config.Roles, 2)
}

func TestNewFileBasedRolesRetriever_FileNotFound(t *testing.T) {
	retriever, err := NewFileBasedRolesRetriever("/nonexistent/path/roles.yaml")

	assert.Error(t, err)
	assert.Nil(t, retriever)
	assert.Contains(t, err.Error(), "failed to load roles")
}

func TestNewFileBasedRolesRetriever_InvalidYAML(t *testing.T) {
	content := `invalid yaml content: [unclosed bracket`
	tmpFile := createTempYAMLFile(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	retriever, err := NewFileBasedRolesRetriever(tmpFile)

	assert.Error(t, err)
	assert.Nil(t, retriever)
	assert.Contains(t, err.Error(), "failed to unmarshal roles YAML")
}

func TestGetAvailableRoleIDs_Success(t *testing.T) {
	roleDefinitions := []RoleDefinition{
		{ID: "owner", DisplayName: "Owner", Description: "Full access"},
		{ID: "member", DisplayName: "Member", Description: "Limited access"},
		{ID: "admin", DisplayName: "Admin", Description: "Admin access"},
	}

	roleIDs := GetAvailableRoleIDs(roleDefinitions)
	assert.Equal(t, []string{"owner", "member", "admin"}, roleIDs)
}

func TestGetAvailableRoleIDs_EmptySlice(t *testing.T) {
	roleDefinitions := []RoleDefinition{}
	roleIDs := GetAvailableRoleIDs(roleDefinitions)
	assert.Equal(t, []string{}, roleIDs)
}

func TestGetRoleDefinitions_Success(t *testing.T) {
	content := `roles:
  - groupResource: core_platform-mesh_io_account
    roles:
      - id: owner
        displayName: Owner
        description: Full access to all resources
      - id: member
        displayName: Member
        description: Limited access to resources`

	tmpFile := createTempYAMLFile(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	retriever, err := NewFileBasedRolesRetriever(tmpFile)
	require.NoError(t, err)

	definitions, err := retriever.GetRoleDefinitions("core_platform-mesh_io_account")
	assert.NoError(t, err)
	assert.Len(t, definitions, 2)

	// Check first role definition
	assert.Equal(t, "owner", definitions[0].ID)
	assert.Equal(t, "Owner", definitions[0].DisplayName)
	assert.Equal(t, "Full access to all resources", definitions[0].Description)

	// Check second role definition
	assert.Equal(t, "member", definitions[1].ID)
	assert.Equal(t, "Member", definitions[1].DisplayName)
	assert.Equal(t, "Limited access to resources", definitions[1].Description)
}

func TestGetRoleDefinitions_GroupResourceNotFound(t *testing.T) {
	content := `roles:
  - groupResource: core_platform-mesh_io_account
    roles:
      - id: owner
        displayName: Owner`

	tmpFile := createTempYAMLFile(t, content)
	defer func() { _ = os.Remove(tmpFile) }()

	retriever, err := NewFileBasedRolesRetriever(tmpFile)
	require.NoError(t, err)

	definitions, err := retriever.GetRoleDefinitions("nonexistent_resource")
	assert.NoError(t, err)
	assert.Empty(t, definitions)
}

func TestGetRoleDefinitions_NoConfigLoaded(t *testing.T) {
	retriever := &FileBasedRolesRetriever{}

	definitions, err := retriever.GetRoleDefinitions("any_resource")
	assert.Error(t, err)
	assert.Nil(t, definitions)
	assert.Contains(t, err.Error(), "roles configuration not loaded")
}

func TestNewDefaultRolesRetriever_IntegrationTest(t *testing.T) {
	// This test checks if the default roles.yaml exists and can be loaded
	// It's more of an integration test to ensure the actual file structure works

	// Save current directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	// Change to project root (assuming tests run from pkg/roles)
	projectRoot := filepath.Join(originalWd, "..", "..")
	err = os.Chdir(projectRoot)
	require.NoError(t, err)

	// Check if input/roles.yaml exists
	rolesFile := filepath.Join("input", "roles.yaml")
	if _, err := os.Stat(rolesFile); os.IsNotExist(err) {
		t.Skip("input/roles.yaml does not exist, skipping integration test")
	}

	retriever, err := NewDefaultRolesRetriever()
	assert.NoError(t, err)
	assert.NotNil(t, retriever)

	// Try to get role definitions for the existing group resource
	roleDefinitions, err := retriever.GetRoleDefinitions("core_platform-mesh_io_account")
	assert.NoError(t, err)
	assert.NotEmpty(t, roleDefinitions)

	// Test the helper function
	roleIDs := GetAvailableRoleIDs(roleDefinitions)
	assert.Contains(t, roleIDs, "owner")
	assert.Contains(t, roleIDs, "member")
}

// Helper function to create a temporary YAML file for testing
func createTempYAMLFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "roles_test_*.yaml")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}
