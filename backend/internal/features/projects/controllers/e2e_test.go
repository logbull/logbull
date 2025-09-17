package projects_controllers

import (
	"net/http"
	"testing"

	projects_dto "logbull/internal/features/projects/dto"
	projects_models "logbull/internal/features/projects/models"
	projects_testing "logbull/internal/features/projects/testing"

	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_ProjectLifecycleE2E_CompletesSuccessfully(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	users_testing.EnableMemberProjectCreation()
	defer users_testing.ResetSettingsToDefaults()

	// 1. Create project owner
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	// 2. Owner creates project
	createRequest := projects_dto.CreateProjectRequestDTO{
		Name: "E2E Test Project",
	}

	var projectResponse projects_dto.ProjectResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects",
		"Bearer "+owner.Token,
		createRequest,
		http.StatusOK,
		&projectResponse,
	)

	assert.Equal(t, "E2E Test Project", projectResponse.Name)
	assert.Equal(t, users_enums.ProjectRoleOwner, *projectResponse.UserRole)
	projectID := projectResponse.ID

	// 3. Owner invites a new user
	inviteRequest := projects_dto.AddMemberRequestDTO{
		Email: "invited" + uuid.New().String() + "@example.com",
		Role:  users_enums.ProjectRoleMember,
	}

	var inviteResponse projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+projectID.String()+"/members",
		"Bearer "+owner.Token,
		inviteRequest,
		http.StatusOK,
		&inviteResponse,
	)

	assert.True(t, inviteResponse.Status == projects_dto.AddStatusInvited)

	// 4. Add existing user to project
	existingMember := users_testing.CreateTestUser(users_enums.UserRoleMember)
	addMemberRequest := projects_dto.AddMemberRequestDTO{
		Email: existingMember.Email,
		Role:  users_enums.ProjectRoleMember,
	}

	var addMemberResponse projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+projectID.String()+"/members",
		"Bearer "+owner.Token,
		addMemberRequest,
		http.StatusOK,
		&addMemberResponse,
	)

	assert.True(t, addMemberResponse.Status == projects_dto.AddStatusAdded)

	// 5. List project members
	var membersResponse projects_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+projectID.String()+"/members",
		"Bearer "+owner.Token,
		http.StatusOK,
		&membersResponse,
	)

	assert.GreaterOrEqual(t, len(membersResponse.Members), 2) // owner + added member

	roles := make([]users_enums.ProjectRole, len(membersResponse.Members))
	for i, m := range membersResponse.Members {
		roles[i] = m.Role
	}
	assert.Contains(t, roles, users_enums.ProjectRoleOwner)
	assert.Contains(t, roles, users_enums.ProjectRoleMember)

	// 6. Promote member to admin
	promoteRequest := projects_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.ProjectRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+projectID.String()+"/members/"+existingMember.UserID.String()+"/role",
		"Bearer "+owner.Token,
		promoteRequest,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Member role changed successfully")

	// 7. Update project settings
	updateRequest := projects_models.Project{
		Name:               "Updated E2E Project",
		IsApiKeyRequired:   true,
		IsFilterByDomain:   true,
		AllowedDomains:     []string{"example.com"},
		LogsPerSecondLimit: 5000,
		MaxLogsAmount:      50000000,
		MaxLogsSizeMB:      5000,
		MaxLogsLifeDays:    365,
		MaxLogSizeKB:       256,
	}

	var updateResponse projects_models.Project
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/"+projectID.String(),
		"Bearer "+owner.Token,
		updateRequest,
		http.StatusOK,
		&updateResponse,
	)

	assert.Equal(t, "Updated E2E Project", updateResponse.Name)
	assert.True(t, updateResponse.IsApiKeyRequired)
	assert.True(t, updateResponse.IsFilterByDomain)
	assert.Equal(t, 5000, updateResponse.LogsPerSecondLimit)
	assert.Contains(t, updateResponse.AllowedDomains, "example.com")

	// 8. Transfer ownership
	transferRequest := projects_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: existingMember.Email,
	}

	resp = test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+projectID.String()+"/transfer-ownership",
		"Bearer "+owner.Token,
		transferRequest,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Ownership transferred successfully")

	// 9. New owner can now manage project
	var finalProject projects_models.Project
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/"+projectID.String(),
		"Bearer "+existingMember.Token,
		http.StatusOK,
		&finalProject,
	)

	assert.Equal(t, projectID, finalProject.ID)
	assert.Equal(t, "Updated E2E Project", finalProject.Name)

	// 10. New owner can delete project
	resp = test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/" + projectID.String(),
		AuthToken:      "Bearer " + existingMember.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Project deleted successfully")
}

func Test_AdminProjectManagementE2E_CompletesSuccessfully(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())

	// 1. Create admin and regular user
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	regularUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	// 2. Regular user creates project (with member creation disabled)
	users_testing.DisableMemberProjectCreation()
	defer users_testing.ResetSettingsToDefaults()

	// Regular user cannot create project
	createRequest := projects_dto.CreateProjectRequestDTO{
		Name: "Regular User Project",
	}

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects",
		"Bearer "+regularUser.Token,
		createRequest,
		http.StatusForbidden,
	)

	// 3. Admin can create project regardless of settings
	adminCreateRequest := projects_dto.CreateProjectRequestDTO{
		Name: "Admin Project",
	}

	var adminProjectResponse projects_dto.ProjectResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects",
		"Bearer "+admin.Token,
		adminCreateRequest,
		http.StatusOK,
		&adminProjectResponse,
	)

	assert.Equal(t, "Admin Project", adminProjectResponse.Name)
	adminProjectID := adminProjectResponse.ID

	// 4. Admin can view any project (even not a member)
	regularUser2 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	users_testing.EnableMemberProjectCreation()

	regularUserCreateRequest := projects_dto.CreateProjectRequestDTO{
		Name: "Regular User Project 2",
	}

	var regularProjectResponse projects_dto.ProjectResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects",
		"Bearer "+regularUser2.Token,
		regularUserCreateRequest,
		http.StatusOK,
		&regularProjectResponse,
	)

	regularProjectID := regularProjectResponse.ID

	// Admin can view regular user's project
	var adminViewResponse projects_models.Project
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/"+regularProjectID.String(),
		"Bearer "+admin.Token,
		http.StatusOK,
		&adminViewResponse,
	)

	assert.Equal(t, regularProjectID, adminViewResponse.ID)

	// 5. Admin can delete any project
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/" + regularProjectID.String(),
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Project deleted successfully")

	// 6. Clean up admin's project
	test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "DELETE",
		URL:            "/api/v1/projects/" + adminProjectID.String(),
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusOK,
	})
}
