package projects_controllers

import (
	"fmt"
	"net/http"
	"testing"

	audit_logs "logbull/internal/features/audit_logs"
	projects_dto "logbull/internal/features/projects/dto"
	projects_models "logbull/internal/features/projects/models"
	projects_services "logbull/internal/features/projects/services"
	projects_testing "logbull/internal/features/projects/testing"

	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_models "logbull/internal/features/users/models"
	users_services "logbull/internal/features/users/services"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_CreateProjectViaMember_WhenMemberProjectsEnabled_ProjectCreated(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	users_testing.EnableMemberProjectCreation()
	defer users_testing.ResetSettingsToDefaults()

	user := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := projects_dto.CreateProjectRequestDTO{
		Name: "Test Project",
	}

	var response projects_dto.ProjectResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects",
		"Bearer "+user.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, "Test Project", response.Name)
	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.Equal(t, users_enums.ProjectRoleOwner, *response.UserRole)
}

func Test_CreateProjectViaMember_WhenMemberProjectsDisabled_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	defer users_testing.ResetSettingsToDefaults()

	user := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()[:8]
	request := projects_dto.CreateProjectRequestDTO{
		Name: fmt.Sprintf("Test Project %s", uniqueID),
	}

	users_testing.DisableMemberProjectCreation()

	settingsService := users_services.GetSettingsService()
	settings, err := settingsService.GetSettings()
	assert.NoError(t, err)

	if settings.IsMemberAllowedToCreateProjects {
		t.Fatal("RACE CONDITION DETECTED: Member project creation should be disabled but was enabled by another test")
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects",
		"Bearer "+user.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to create projects")
}

func Test_CreateProjectViaGlobalAdmin_WhenMemberProjectsDisabled_ProjectCreated(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	users_testing.DisableMemberProjectCreation()
	defer users_testing.ResetSettingsToDefaults()

	globalAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	request := projects_dto.CreateProjectRequestDTO{
		Name: "GlobalAdmin Project",
	}

	var response projects_dto.ProjectResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects",
		"Bearer "+globalAdmin.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, "GlobalAdmin Project", response.Name)
	assert.Equal(t, users_enums.ProjectRoleOwner, *response.UserRole)
}

func Test_CreateProject_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	user := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/projects",
		Body:           "invalid json",
		AuthToken:      "Bearer " + user.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_CreateProject_WithoutAuthToken_ReturnsUnauthorized(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())

	request := projects_dto.CreateProjectRequestDTO{
		Name: "Test Project",
	}

	test_utils.MakePostRequest(t, router, "/api/v1/projects", "", request, http.StatusUnauthorized)
}

func Test_GetUserProjects_WhenUserHasProjects_ReturnsProjectsList(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project1, _ := projects_testing.CreateTestProjectWithToken("Project 1", user.Token, router)
	project2, _ := projects_testing.CreateTestProjectWithToken("Project 2", user.Token, router)

	var response projects_dto.ListProjectsResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects",
		"Bearer "+user.Token,
		http.StatusOK,
		&response,
	)

	assert.GreaterOrEqual(t, len(response.Projects), 2)

	projectNames := make([]string, len(response.Projects))
	for i, p := range response.Projects {
		projectNames[i] = p.Name
	}
	assert.Contains(t, projectNames, project1.Name)
	assert.Contains(t, projectNames, project2.Name)
}

func Test_GetUserProjects_WithoutAuthToken_ReturnsUnauthorized(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	test_utils.MakeGetRequest(t, router, "/api/v1/projects", "", http.StatusUnauthorized)
}

func Test_GetSingleProject_WhenUserIsProjectMember_ReturnsProject(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", user.Token, router)

	var response projects_models.Project
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+user.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, project.ID, response.ID)
	assert.Equal(t, "Test Project", response.Name)
}

func Test_GetSingleProject_WhenUserIsNotProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", owner.Token, router)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+nonMember.Token,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to view project")
}

func Test_GetSingleProject_WhenUserIsGlobalAdmin_ReturnsProject(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	globalAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", owner.Token, router)

	var response projects_models.Project
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+globalAdmin.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, project.ID, response.ID)
}

func Test_GetSingleProject_WithInvalidProjectID_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/projects/invalid-uuid",
		"Bearer "+user.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid project ID")
}

func Test_UpdateProject_WhenUserIsProjectOwner_ProjectUpdated(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Original Name", user.Token, router)

	updateRequest := projects_models.Project{
		Name:               project.Name,
		IsApiKeyRequired:   true,
		IsFilterByDomain:   true,
		IsFilterByIP:       false,
		AllowedDomains:     []string{"example.com", "test.com"},
		AllowedIPs:         []string{},
		LogsPerSecondLimit: 2000,
		MaxLogsAmount:      20_000_000,
		MaxLogsSizeMB:      2000,
		MaxLogsLifeDays:    180,
		MaxLogSizeKB:       128,
	}

	var response projects_models.Project
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+user.Token,
		updateRequest,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, project.ID, response.ID)
	assert.True(t, response.IsApiKeyRequired)
	assert.True(t, response.IsFilterByDomain)
	assert.Equal(t, 2000, response.LogsPerSecondLimit)
	assert.Contains(t, response.AllowedDomains, "example.com")
	assert.Contains(t, response.AllowedDomains, "test.com")
}

func Test_UpdateProject_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", owner.Token, router)
	projects_testing.AddMemberToProject(project, member, users_enums.ProjectRoleMember, owner.Token, router)

	updateRequest := projects_models.Project{
		Name: "Updated Name",
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+member.Token,
		updateRequest,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to update project")
}

func Test_DeleteProject_WhenUserIsProjectOwner_ProjectDeleted(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", user.Token, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/" + project.ID.String(),
		AuthToken:      "Bearer " + user.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Project deleted successfully")
}

func Test_DeleteProject_WhenUserIsGlobalAdmin_ProjectDeleted(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	globalAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", owner.Token, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/" + project.ID.String(),
		AuthToken:      "Bearer " + globalAdmin.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Project deleted successfully")
}

func Test_DeleteProject_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", owner.Token, router)
	projects_testing.AddMemberToProject(project, member, users_enums.ProjectRoleMember, owner.Token, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/" + project.ID.String(),
		AuthToken:      "Bearer " + member.Token,
		ExpectedStatus: http.StatusForbidden,
	})

	assert.Contains(t, string(resp.Body), "only project owner or admin can delete project")
}

func Test_UpdateProject_WhenUserIsProjectAdmin_ProjectUpdated(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", owner.Token, router)
	projects_testing.AddMemberToProject(project, projectAdmin, users_enums.ProjectRoleAdmin, owner.Token, router)

	updateRequest := projects_models.Project{
		Name:               project.Name,
		IsApiKeyRequired:   true,
		LogsPerSecondLimit: 3000,
	}

	var response projects_models.Project
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+projectAdmin.Token,
		updateRequest,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, project.ID, response.ID)
	assert.True(t, response.IsApiKeyRequired)
	assert.Equal(t, 3000, response.LogsPerSecondLimit)
}

func Test_GetSingleProject_WhenUserIsProjectAdmin_ReturnsProject(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", owner.Token, router)
	projects_testing.AddMemberToProject(project, projectAdmin, users_enums.ProjectRoleAdmin, owner.Token, router)

	var response projects_models.Project
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+projectAdmin.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, project.ID, response.ID)
	assert.Equal(t, "Test Project", response.Name)
}

func Test_GetProjectAuditLogs_WhenUserIsProjectAdmin_ReturnsAuditLogs(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New()
	projectName := fmt.Sprintf("ProjectAdmin Test %s", uniqueID.String()[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	projects_testing.AddMemberToProject(project, projectAdmin, users_enums.ProjectRoleAdmin, owner.Token, router)
	var response audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/projects/"+project.ID.String()+"/audit-logs",
		"Bearer "+projectAdmin.Token, http.StatusOK, &response)

	assert.GreaterOrEqual(t, len(response.AuditLogs), 2) // Create + Add member
	for _, log := range response.AuditLogs {
		assert.Equal(t, &project.ID, log.ProjectID)
	}
}

func Test_GetProjectAuditLogs_WithMultipleProjects_ReturnsOnlyProjectSpecificLogs(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner2 := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID1 := uuid.New()
	uniqueID2 := uuid.New()
	projectName1 := fmt.Sprintf("Project Test %s", uniqueID1.String()[:8])
	projectName2 := fmt.Sprintf("Project Test %s", uniqueID2.String()[:8])

	project1, _ := projects_testing.CreateTestProjectWithToken(projectName1, owner1.Token, router)
	project2, _ := projects_testing.CreateTestProjectWithToken(projectName2, owner2.Token, router)
	updateProject1 := projects_models.Project{
		Name:               project1.Name,
		IsApiKeyRequired:   true,
		LogsPerSecondLimit: 1500,
	}
	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/"+project1.ID.String(),
		"Bearer "+owner1.Token,
		updateProject1,
		http.StatusOK,
	)

	updateProject2 := projects_models.Project{
		Name:               project2.Name,
		IsFilterByDomain:   true,
		LogsPerSecondLimit: 2000,
	}
	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/"+project2.ID.String(),
		"Bearer "+owner2.Token,
		updateProject2,
		http.StatusOK,
	)

	var project1Response audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/projects/"+project1.ID.String()+"/audit-logs?limit=50",
		"Bearer "+owner1.Token, http.StatusOK, &project1Response)

	var project2Response audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/projects/"+project2.ID.String()+"/audit-logs?limit=50",
		"Bearer "+owner2.Token, http.StatusOK, &project2Response)

	assert.GreaterOrEqual(t, len(project1Response.AuditLogs), 2)
	for _, log := range project1Response.AuditLogs {
		assert.Equal(t, &project1.ID, log.ProjectID)
		assert.Contains(t, log.Message, projectName1)
	}

	assert.GreaterOrEqual(t, len(project2Response.AuditLogs), 2)
	for _, log := range project2Response.AuditLogs {
		assert.Equal(t, &project2.ID, log.ProjectID)
		assert.Contains(t, log.Message, projectName2)
	}

	project1Messages := extractAuditLogMessages(project1Response.AuditLogs)
	project2Messages := extractAuditLogMessages(project2Response.AuditLogs)

	for _, msg := range project1Messages {
		assert.NotContains(t, msg, projectName2, "Project1 logs should not contain Project2 name")
	}

	for _, msg := range project2Messages {
		assert.NotContains(t, msg, projectName1, "Project2 logs should not contain Project1 name")
	}
}

func Test_GetProjectAuditLogs_WithDifferentUserRoles_EnforcesPermissionsCorrectly(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	globalAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New()
	projectName := fmt.Sprintf("Audit Test Project %s", uniqueID.String()[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	projects_testing.AddMemberToProject(project, member, users_enums.ProjectRoleMember, owner.Token, router)
	var ownerResponse audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/projects/"+project.ID.String()+"/audit-logs",
		"Bearer "+owner.Token, http.StatusOK, &ownerResponse)

	assert.GreaterOrEqual(t, len(ownerResponse.AuditLogs), 2)
	var memberResponse audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/projects/"+project.ID.String()+"/audit-logs",
		"Bearer "+member.Token, http.StatusOK, &memberResponse)

	assert.GreaterOrEqual(t, len(memberResponse.AuditLogs), 2)

	var globalAdminResponse audit_logs.GetAuditLogsResponse
	test_utils.MakeGetRequestAndUnmarshal(t, router,
		"/api/v1/projects/"+project.ID.String()+"/audit-logs",
		"Bearer "+globalAdmin.Token, http.StatusOK, &globalAdminResponse)

	assert.GreaterOrEqual(t, len(globalAdminResponse.AuditLogs), 2)

	resp := test_utils.MakeGetRequest(t, router,
		"/api/v1/projects/"+project.ID.String()+"/audit-logs",
		"Bearer "+nonMember.Token, http.StatusForbidden)

	assert.Contains(t, string(resp.Body), "insufficient permissions to view project audit logs")
}

func Test_GetProjectAuditLogs_WithoutAuthToken_ReturnsUnauthorized(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectWithToken("Test Project", owner.Token, router)

	test_utils.MakeGetRequest(t, router,
		"/api/v1/projects/"+project.ID.String()+"/audit-logs",
		"", http.StatusUnauthorized)
}

func Test_GetProjectWithCache_WhenProjectExists_ReturnsCachedProject(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectWithToken("Cache Test Project", owner.Token, router)
	projectService := projects_services.GetProjectService()

	cachedProject1, err := projectService.GetProjectWithCache(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, project.ID, cachedProject1.ID)
	assert.Equal(t, "Cache Test Project", cachedProject1.Name)
	assert.False(t, cachedProject1.IsNotExists)

	cachedProject2, err := projectService.GetProjectWithCache(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, project.ID, cachedProject2.ID)
	assert.Equal(t, "Cache Test Project", cachedProject2.Name)
	assert.False(t, cachedProject2.IsNotExists)
}

func Test_GetProjectWithCache_WhenProjectNotExists_CachesNotFound(t *testing.T) {
	projectService := projects_services.GetProjectService()
	nonExistentID := uuid.New()

	_, err := projectService.GetProjectWithCache(nonExistentID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project not found")

	_, err2 := projectService.GetProjectWithCache(nonExistentID)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "project not found")
}

func Test_CreateProject_PrewarmsCacheAutomatically(t *testing.T) {
	users_testing.EnableMemberProjectCreation()
	defer users_testing.ResetSettingsToDefaults()

	ownerResponse := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner := getUserFromSignInResponse(ownerResponse)
	projectService := projects_services.GetProjectService()

	request := &projects_dto.CreateProjectRequestDTO{
		Name: "Prewarmed Cache Project",
	}

	response, err := projectService.CreateProject(request, owner)
	assert.NoError(t, err)
	assert.NotNil(t, response)

	cachedProject, err := projectService.GetProjectWithCache(response.ID)
	assert.NoError(t, err)
	assert.Equal(t, response.ID, cachedProject.ID)
	assert.Equal(t, "Prewarmed Cache Project", cachedProject.Name)
	assert.False(t, cachedProject.IsNotExists)
}

func Test_UpdateProject_InvalidatesCache(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	ownerResponse := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner := getUserFromSignInResponse(ownerResponse)
	project, _ := projects_testing.CreateTestProjectWithToken("Cache Invalidation Test", ownerResponse.Token, router)
	projectService := projects_services.GetProjectService()

	cachedProject1, err := projectService.GetProjectWithCache(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Cache Invalidation Test", cachedProject1.Name)

	updateProject := &projects_models.Project{
		Name:               "Updated Cache Test Project",
		IsApiKeyRequired:   true,
		IsFilterByDomain:   true,
		IsFilterByIP:       false,
		AllowedDomains:     []string{"example.com"},
		AllowedIPs:         []string{},
		LogsPerSecondLimit: 2000,
		MaxLogsAmount:      20_000_000,
		MaxLogsSizeMB:      2000,
		MaxLogsLifeDays:    180,
		MaxLogSizeKB:       128,
	}

	updatedProject, err := projectService.UpdateProject(project.ID, updateProject, owner)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Cache Test Project", updatedProject.Name)

	cachedProject2, err := projectService.GetProjectWithCache(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Cache Test Project", cachedProject2.Name)
	assert.True(t, cachedProject2.IsApiKeyRequired)
	assert.True(t, cachedProject2.IsFilterByDomain)
	assert.Contains(t, cachedProject2.AllowedDomains, "example.com")
	assert.Equal(t, 2000, cachedProject2.LogsPerSecondLimit)
}

func Test_DeleteProject_InvalidatesCache(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	ownerResponse := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner := getUserFromSignInResponse(ownerResponse)
	project, _ := projects_testing.CreateTestProjectWithToken("Delete Cache Test", ownerResponse.Token, router)
	projectService := projects_services.GetProjectService()

	cachedProject, err := projectService.GetProjectWithCache(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Delete Cache Test", cachedProject.Name)

	err = projectService.DeleteProject(project.ID, owner)
	assert.NoError(t, err)

	_, err = projectService.GetProjectWithCache(project.ID)
	assert.Error(t, err)
}

func Test_GetProjectWithCache_CachesAllProjectFields(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	ownerResponse := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner := getUserFromSignInResponse(ownerResponse)
	project, _ := projects_testing.CreateTestProjectWithToken("Full Fields Test", ownerResponse.Token, router)
	projectService := projects_services.GetProjectService()

	updateProject := &projects_models.Project{
		Name:               "Complete Project Settings",
		IsApiKeyRequired:   true,
		IsFilterByDomain:   true,
		IsFilterByIP:       true,
		AllowedDomains:     []string{"domain1.com", "domain2.com"},
		AllowedIPs:         []string{"192.168.1.1", "10.0.0.1"},
		LogsPerSecondLimit: 5000,
		MaxLogsAmount:      50_000_000,
		MaxLogsSizeMB:      5000,
		MaxLogsLifeDays:    365,
		MaxLogSizeKB:       256,
	}

	_, err := projectService.UpdateProject(project.ID, updateProject, owner)
	assert.NoError(t, err)

	cachedProject, err := projectService.GetProjectWithCache(project.ID)
	assert.NoError(t, err)

	assert.Equal(t, "Complete Project Settings", cachedProject.Name)
	assert.True(t, cachedProject.IsApiKeyRequired)
	assert.True(t, cachedProject.IsFilterByDomain)
	assert.True(t, cachedProject.IsFilterByIP)
	assert.Contains(t, cachedProject.AllowedDomains, "domain1.com")
	assert.Contains(t, cachedProject.AllowedDomains, "domain2.com")
	assert.Contains(t, cachedProject.AllowedIPs, "192.168.1.1")
	assert.Contains(t, cachedProject.AllowedIPs, "10.0.0.1")
	assert.Equal(t, 5000, cachedProject.LogsPerSecondLimit)
	assert.Equal(t, int64(50_000_000), cachedProject.MaxLogsAmount)
	assert.Equal(t, 5000, cachedProject.MaxLogsSizeMB)
	assert.Equal(t, 365, cachedProject.MaxLogsLifeDays)
	assert.Equal(t, 256, cachedProject.MaxLogSizeKB)
	assert.False(t, cachedProject.IsNotExists)
}

func Test_GetProjectWithCache_MultipleNonExistentProjects_CachesEachSeparately(t *testing.T) {
	projectService := projects_services.GetProjectService()
	nonExistentID1 := uuid.New()
	nonExistentID2 := uuid.New()

	_, err1 := projectService.GetProjectWithCache(nonExistentID1)
	assert.Error(t, err1)
	assert.Contains(t, err1.Error(), "project not found")

	_, err2 := projectService.GetProjectWithCache(nonExistentID2)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "project not found")

	_, err3 := projectService.GetProjectWithCache(nonExistentID1)
	assert.Error(t, err3)

	_, err4 := projectService.GetProjectWithCache(nonExistentID2)
	assert.Error(t, err4)
}

func extractAuditLogMessages(logs []*audit_logs.AuditLogDTO) []string {
	messages := make([]string, len(logs))
	for i, log := range logs {
		messages[i] = log.Message
	}
	return messages
}

func getUserFromSignInResponse(response *users_dto.SignInResponseDTO) *users_models.User {
	userService := users_services.GetUserService()
	user, err := userService.GetUserByID(response.UserID)
	if err != nil {
		panic(err)
	}
	return user
}
