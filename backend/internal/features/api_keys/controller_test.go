package api_keys

import (
	"net/http"
	"testing"

	projects_controllers "logbull/internal/features/projects/controllers"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// CreateApiKey Tests

func Test_CreateApiKey_WhenUserIsProjectOwner_ApiKeyCreated(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := CreateApiKeyRequestDTO{
		Name: "Test API Key",
	}

	var response ApiKey
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, "Test API Key", response.Name)
	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.NotEmpty(t, response.Token)
	assert.NotEmpty(t, response.TokenPrefix)
	assert.Contains(t, response.Token, "lb_")
	assert.Contains(t, response.TokenPrefix, "lb_")
	assert.Contains(t, response.TokenPrefix, "...")
}

func Test_CreateApiKey_WhenUserIsProjectAdmin_ApiKeyCreated(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	admin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProject(project, admin, users_enums.ProjectRoleAdmin, owner.Token, router)

	request := CreateApiKeyRequestDTO{
		Name: "Admin API Key",
	}

	var response ApiKey
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+admin.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, "Admin API Key", response.Name)
	assert.NotEmpty(t, response.Token)
}

func Test_CreateApiKey_WhenUserIsGlobalAdmin_ApiKeyCreated(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	globalAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := CreateApiKeyRequestDTO{
		Name: "Global Admin API Key",
	}

	var response ApiKey
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+globalAdmin.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, "Global Admin API Key", response.Name)
}

func Test_CreateApiKey_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProject(project, member, users_enums.ProjectRoleMember, owner.Token, router)

	request := CreateApiKeyRequestDTO{
		Name: "Member API Key",
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+member.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to create API keys")
}

func Test_CreateApiKey_WhenUserIsNotProjectMember_ReturnsForbidden(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := CreateApiKeyRequestDTO{
		Name: "Non-member API Key",
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+nonMember.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to create API keys")
}

func Test_CreateApiKey_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/projects/api-keys/" + project.ID.String(),
		Body:           "invalid json",
		AuthToken:      "Bearer " + owner.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_CreateApiKey_WithInvalidProjectID_ReturnsBadRequest(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := CreateApiKeyRequestDTO{
		Name: "Test API Key",
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/api-keys/invalid-uuid",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid project ID")
}

// GetApiKeys Tests

func Test_GetApiKeys_WhenUserIsProjectOwner_ReturnsApiKeys(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	// Create API keys
	CreateTestApiKey("API Key 1", project.ID, owner.Token, router)
	CreateTestApiKey("API Key 2", project.ID, owner.Token, router)

	var response GetApiKeysResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.GreaterOrEqual(t, len(response.ApiKeys), 2)

	// Verify keys are returned
	apiKeyNames := make([]string, len(response.ApiKeys))
	for i, key := range response.ApiKeys {
		apiKeyNames[i] = key.Name
		assert.NotEqual(t, uuid.Nil, key.ID)
		assert.NotEmpty(t, key.TokenPrefix)
		assert.Contains(t, key.TokenPrefix, "lb_")
		assert.Contains(t, key.TokenPrefix, "...")
		assert.Equal(t, ApiKeyStatusActive, key.Status)
	}
	assert.Contains(t, apiKeyNames, "API Key 1")
	assert.Contains(t, apiKeyNames, "API Key 2")
}

func Test_GetApiKeys_WhenUserIsProjectMember_ReturnsApiKeys(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProject(project, member, users_enums.ProjectRoleMember, owner.Token, router)

	CreateTestApiKey("Member View Key", project.ID, owner.Token, router)

	var response GetApiKeysResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+member.Token,
		http.StatusOK,
		&response,
	)

	assert.GreaterOrEqual(t, len(response.ApiKeys), 1)
}

func Test_GetApiKeys_WhenUserIsGlobalAdmin_ReturnsApiKeys(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	globalAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	CreateTestApiKey("Admin View Key", project.ID, owner.Token, router)

	var response GetApiKeysResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+globalAdmin.Token,
		http.StatusOK,
		&response,
	)

	assert.GreaterOrEqual(t, len(response.ApiKeys), 1)
}

func Test_GetApiKeys_WhenUserIsNotProjectMember_ReturnsForbidden(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String(),
		"Bearer "+nonMember.Token,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to view API keys")
}

// UpdateApiKey Tests

func Test_UpdateApiKey_WhenUserIsProjectOwner_ApiKeyUpdated(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	apiKey := CreateTestApiKey("Original Key", project.ID, owner.Token, router)

	newName := "Updated Key Name"
	status := ApiKeyStatusDisabled
	request := UpdateApiKeyRequestDTO{
		Name:   &newName,
		Status: &status,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String()+"/"+apiKey.ID.String(),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "API key updated successfully")
}

func Test_UpdateApiKey_WhenUserIsProjectAdmin_ApiKeyUpdated(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	admin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProject(project, admin, users_enums.ProjectRoleAdmin, owner.Token, router)
	apiKey := CreateTestApiKey("Admin Update Key", project.ID, owner.Token, router)

	newName := "Admin Updated Key"
	request := UpdateApiKeyRequestDTO{
		Name: &newName,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String()+"/"+apiKey.ID.String(),
		"Bearer "+admin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "API key updated successfully")
}

func Test_UpdateApiKey_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProject(project, member, users_enums.ProjectRoleMember, owner.Token, router)
	apiKey := CreateTestApiKey("Member Update Key", project.ID, owner.Token, router)

	newName := "Member Updated Key"
	request := UpdateApiKeyRequestDTO{
		Name: &newName,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String()+"/"+apiKey.ID.String(),
		"Bearer "+member.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to update API keys")
}

func Test_UpdateApiKey_WithNonExistentApiKey_ReturnsBadRequest(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	newName := "Non-existent Key"
	request := UpdateApiKeyRequestDTO{
		Name: &newName,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/api-keys/"+project.ID.String()+"/"+uuid.New().String(),
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "API key not found")
}

// DeleteApiKey Tests

func Test_DeleteApiKey_WhenUserIsProjectOwner_ApiKeyDeleted(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	apiKey := CreateTestApiKey("Delete Key", project.ID, owner.Token, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/api-keys/" + project.ID.String() + "/" + apiKey.ID.String(),
		AuthToken:      "Bearer " + owner.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "API key deleted successfully")
}

func Test_DeleteApiKey_WhenUserIsProjectAdmin_ApiKeyDeleted(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	admin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProject(project, admin, users_enums.ProjectRoleAdmin, owner.Token, router)
	apiKey := CreateTestApiKey("Admin Delete Key", project.ID, owner.Token, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/api-keys/" + project.ID.String() + "/" + apiKey.ID.String(),
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "API key deleted successfully")
}

func Test_DeleteApiKey_WhenUserIsGlobalAdmin_ApiKeyDeleted(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	globalAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	apiKey := CreateTestApiKey("Global Delete Key", project.ID, owner.Token, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/api-keys/" + project.ID.String() + "/" + apiKey.ID.String(),
		AuthToken:      "Bearer " + globalAdmin.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "API key deleted successfully")
}

func Test_DeleteApiKey_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProject(project, member, users_enums.ProjectRoleMember, owner.Token, router)
	apiKey := CreateTestApiKey("Member Delete Key", project.ID, owner.Token, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/api-keys/" + project.ID.String() + "/" + apiKey.ID.String(),
		AuthToken:      "Bearer " + member.Token,
		ExpectedStatus: http.StatusForbidden,
	})

	assert.Contains(t, string(resp.Body), "insufficient permissions to delete API keys")
}

func Test_DeleteApiKey_WithNonExistentApiKey_ReturnsBadRequest(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/api-keys/" + project.ID.String() + "/" + uuid.New().String(),
		AuthToken:      "Bearer " + owner.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "API key not found")
}

// Cross-project security tests

func Test_UpdateApiKey_WithApiKeyFromDifferentProject_ReturnsBadRequest(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner2 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project1, _ := projects_testing.CreateTestProjectViaAPI("Project 1", owner1, router)
	project2, _ := projects_testing.CreateTestProjectViaAPI("Project 2", owner2, router)

	// Create API key in project1
	apiKey := CreateTestApiKey("Cross Project Key", project1.ID, owner1.Token, router)

	// Try to update it via project2 endpoint
	newName := "Hacked Key"
	request := UpdateApiKeyRequestDTO{
		Name: &newName,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/api-keys/"+project2.ID.String()+"/"+apiKey.ID.String(),
		"Bearer "+owner2.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "API key does not belong to this project")
}

func Test_DeleteApiKey_WithApiKeyFromDifferentProject_ReturnsBadRequest(t *testing.T) {
	router := CreateApiKeyTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner2 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project1, _ := projects_testing.CreateTestProjectViaAPI("Project 1", owner1, router)
	project2, _ := projects_testing.CreateTestProjectViaAPI("Project 2", owner2, router)

	// Create API key in project1
	apiKey := CreateTestApiKey("Cross Project Delete Key", project1.ID, owner1.Token, router)

	// Try to delete it via project2 endpoint
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/api-keys/" + project2.ID.String() + "/" + apiKey.ID.String(),
		AuthToken:      "Bearer " + owner2.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "API key does not belong to this project")
}
