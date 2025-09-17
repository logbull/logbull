package users_controllers

import (
	"net/http"
	"testing"
	"time"

	"logbull/internal/features/audit_logs"
	projects_controllers "logbull/internal/features/projects/controllers"
	projects_dto "logbull/internal/features/projects/dto"
	projects_testing "logbull/internal/features/projects/testing"
	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_middleware "logbull/internal/features/users/middleware"
	users_services "logbull/internal/features/users/services"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_GetUsersList_WhenUserIsAdmin_ReturnsUsers(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	var response users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users",
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	assert.NotNil(t, response.Users)
	assert.GreaterOrEqual(t, response.Total, int64(1)) // At least the test user should exist
}

func Test_GetUsersList_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create member user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	resp := test_utils.MakeGetRequest(t, router, "/api/v1/users", "Bearer "+testUser.Token, http.StatusForbidden)
	assert.Contains(t, string(resp.Body), "permissions")
}

func Test_GetUsersList_WithPagination_RespectsLimits(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	var response users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users?limit=5&offset=0",
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	assert.NotNil(t, response.Users)
	assert.LessOrEqual(t, len(response.Users), 5) // Should respect limit
}

func Test_GetUsersList_WithBeforeDateFilter_ReturnsFilteredUsers(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// Test with beforeDate filter
	beforeDate := "2024-01-01T00:00:00Z"
	var response users_dto.ListUsersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users?beforeDate="+beforeDate,
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	assert.NotNil(t, response.Users)
	// All returned users should have been created before the specified date
	for _, user := range response.Users {
		assert.True(t, user.CreatedAt.Before(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
	}
}

func Test_GetUsersList_WithInvalidDateFilter_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// Test with invalid date format
	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users?beforeDate=invalid-date",
		"Bearer "+testUser.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid query parameters")
}

func Test_GetUsersList_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	router := createManagementTestRouter()

	test_utils.MakeGetRequest(t, router, "/api/v1/users", "", http.StatusUnauthorized)
}

func Test_GetUserProfile_WhenAccessingOwnProfile_ReturnsProfile(t *testing.T) {
	router := createManagementTestRouter()

	// Create member user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	var response users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/"+testUser.UserID.String(),
		"Bearer "+testUser.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, testUser.UserID, response.ID)
	assert.Equal(t, users_enums.UserRoleMember, response.Role)
}

func Test_GetUserProfile_WhenUserIsAdmin_ReturnsProfile(t *testing.T) {
	router := createManagementTestRouter()

	// Create both admin and regular user
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	regularUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	var response users_dto.UserProfileResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/"+regularUser.UserID.String(),
		"Bearer "+adminUser.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, regularUser.UserID, response.ID)
}

func Test_GetUserProfile_WhenAccessingOtherUserAsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create two member users
	user1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(users_enums.UserRoleMember)

	test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users/"+user2.UserID.String(),
		"Bearer "+user1.Token,
		http.StatusForbidden,
	)
}

func Test_GetUserProfile_WithNonExistentUser_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// Try to access non-existent user
	test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users/00000000-0000-0000-0000-000000000000",
		"Bearer "+testUser.Token,
		http.StatusForbidden,
	)
}

func Test_GetUserProfile_WithInvalidUserID_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user and get token
	testUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/users/invalid-uuid",
		"Bearer "+testUser.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid user ID")
}

func Test_DeactivateUser_WhenUserIsAdmin_UserDeactivated(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin and target user
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	targetUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/deactivate",
		"Bearer "+adminUser.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User deactivated successfully")
}

func Test_DeactivateUser_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create two member users
	user1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(users_enums.UserRoleMember)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+user2.UserID.String()+"/deactivate",
		"Bearer "+user1.Token,
		nil,
		http.StatusForbidden,
	)
}

func Test_DeactivateUser_WhenDeactivatingOwnAccount_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminUser.UserID.String()+"/deactivate",
		"Bearer "+adminUser.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot deactivate your own account")
}

func Test_ActivateUser_WhenUserIsAdmin_UserActivated(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin and target user
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	targetUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	// First deactivate the user
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/deactivate",
		"Bearer "+adminUser.Token,
		nil,
		http.StatusOK,
	)

	// Now activate the user
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/activate",
		"Bearer "+adminUser.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User activated successfully")
}

func Test_ActivateUser_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create two member users
	user1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(users_enums.UserRoleMember)

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+user2.UserID.String()+"/activate",
		"Bearer "+user1.Token,
		nil,
		http.StatusForbidden,
	)
}

func Test_ChangeUserRole_WhenUserIsRootAdmin_RoleChanged(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and target user
	rootAdmin := users_testing.ReacreateInitAdminAndGetAccess()
	targetUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users_enums.UserRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User role changed successfully")
}

func Test_ChangeUserRole_WhenUserIsMember_ReturnsForbidden(t *testing.T) {
	router := createManagementTestRouter()

	// Create two member users
	user1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users_enums.UserRoleAdmin,
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+user2.UserID.String()+"/role",
		"Bearer "+user1.Token,
		request,
		http.StatusForbidden,
	)
}

func Test_ChangeUserRole_WhenChangingOwnRole_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin user
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users_enums.UserRoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+adminUser.UserID.String()+"/role",
		"Bearer "+adminUser.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot change your own role")
}

func Test_ChangeUserRole_WithInvalidRole_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin and target user
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	targetUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	// Test with invalid JSON structure containing invalid role
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "PUT",
		URL:            "/api/v1/users/" + targetUser.UserID.String() + "/role",
		Body:           map[string]string{"role": "INVALID_ROLE"},
		AuthToken:      "Bearer " + adminUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.NotEmpty(t, resp.Body)
}

func Test_ChangeUserRole_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create admin and target user
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	targetUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "PUT",
		URL:            "/api/v1/users/" + targetUser.UserID.String() + "/role",
		Body:           "invalid json",
		AuthToken:      "Bearer " + adminUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

// Tests for root admin restrictions
func Test_ChangeUserRole_WhenRegularAdminPromotesToAdmin_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create regular admin and target user
	regularAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	targetUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users_enums.UserRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/role",
		"Bearer "+regularAdmin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "only the root admin user can promote users to admin or demote admin users")
}

func Test_ChangeUserRole_WhenRegularAdminDemotesAdmin_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create regular admin and admin target user
	regularAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	adminTargetUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users_enums.UserRoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/role",
		"Bearer "+regularAdmin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "only the root admin user can promote users to admin or demote admin users")
}

func Test_DeactivateUser_WhenRegularAdminDeactivatesAdmin_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create regular admin and admin target user
	regularAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	adminTargetUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/deactivate",
		"Bearer "+regularAdmin.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "only the root admin user can deactivate admin accounts")
}

func Test_ActivateUser_WhenRegularAdminActivatesAdmin_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create regular admin and admin target user
	regularAdmin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	adminTargetUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/activate",
		"Bearer "+regularAdmin.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "only the root admin user can activate admin accounts")
}

func Test_ChangeUserRole_WhenRootAdminPromotesToAdmin_RoleChanged(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and target user
	rootAdmin := users_testing.ReacreateInitAdminAndGetAccess()
	targetUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users_enums.UserRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+targetUser.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User role changed successfully")
}

func Test_ChangeUserRole_WhenRootAdminDemotesAdmin_RoleChanged(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and admin target user
	rootAdmin := users_testing.ReacreateInitAdminAndGetAccess()
	adminTargetUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users_enums.UserRoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User role changed successfully")
}

func Test_DeactivateUser_WhenRootAdminDeactivatesAdmin_UserDeactivated(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and admin target user
	rootAdmin := users_testing.ReacreateInitAdminAndGetAccess()
	adminTargetUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User deactivated successfully")
}

func Test_ActivateUser_WhenRootAdminActivatesAdmin_UserActivated(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin and admin target user
	rootAdmin := users_testing.ReacreateInitAdminAndGetAccess()
	adminTargetUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// First deactivate the admin user
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)

	// Now activate the admin user
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+adminTargetUser.UserID.String()+"/activate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User activated successfully")
}

func Test_ChangeUserRole_WhenRootAdminChangesOwnRole_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin
	rootAdmin := users_testing.ReacreateInitAdminAndGetAccess()

	request := users_dto.ChangeUserRoleRequestDTO{
		Role: users_enums.UserRoleMember,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/"+rootAdmin.UserID.String()+"/role",
		"Bearer "+rootAdmin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot change your own role")
}

func Test_DeactivateUser_WhenRootAdminDeactivatesOwnAccount_ReturnsBadRequest(t *testing.T) {
	router := createManagementTestRouter()

	// Create root admin
	rootAdmin := users_testing.ReacreateInitAdminAndGetAccess()

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/"+rootAdmin.UserID.String()+"/deactivate",
		"Bearer "+rootAdmin.Token,
		nil,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot deactivate your own account")
}

func Test_InviteUserToProject_MembershipReceivedAfterSignUp(t *testing.T) {
	// Setup router with required controllers
	router := createInviteProjectTestRouter()
	users_testing.EnableMemberInvitations()
	users_testing.EnableExternalRegistrations()
	defer users_testing.ResetSettingsToDefaults()

	// 1. Create project owner and project
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Invite Test Project", owner, router)

	// 2. Invite non-existing user to project
	inviteEmail := "invited-" + uuid.New().String() + "@example.com"
	inviteResponse := projects_testing.InviteMemberToProject(
		project,
		inviteEmail,
		users_enums.ProjectRoleMember,
		owner.Token,
		router,
	)

	assert.Equal(t, projects_dto.AddStatusInvited, inviteResponse.Status)

	// 3. Sign up the invited user
	signUpRequest := users_dto.SignUpRequestDTO{
		Email:    inviteEmail,
		Password: "testpassword123",
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/signup",
		"",
		signUpRequest,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "User created successfully")

	// 4. Sign in the newly registered user
	signInRequest := users_dto.SignInRequestDTO{
		Email:    inviteEmail,
		Password: "testpassword123",
	}

	var signInResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		signInRequest,
		http.StatusOK,
		&signInResponse,
	)

	// 5. Verify user is automatically added as member to project
	var membersResponse projects_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+signInResponse.Token,
		http.StatusOK,
		&membersResponse,
	)

	// Check if the invited user is now a member
	var foundMember *projects_dto.ProjectMemberResponseDTO
	for _, member := range membersResponse.Members {
		if member.UserID == signInResponse.UserID {
			foundMember = &member
			break
		}
	}

	assert.NotNil(t, foundMember, "Invited user should be automatically added as project member after sign up")
	assert.Equal(t, users_enums.ProjectRoleMember, foundMember.Role)
	assert.Equal(t, inviteEmail, foundMember.Email)
}

func createInviteProjectTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")

	// Register unprotected routes (signup, signin)
	GetUserController().RegisterRoutes(v1)

	// Register protected routes with auth middleware
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	// Register all necessary controllers for the test
	GetManagementController().RegisterRoutes(protected.(*gin.RouterGroup))
	GetUserController().RegisterProtectedRoutes(protected.(*gin.RouterGroup))
	projects_controllers.GetProjectController().RegisterRoutes(protected.(*gin.RouterGroup))
	projects_controllers.GetMembershipController().RegisterRoutes(protected.(*gin.RouterGroup))
	audit_logs.SetupDependencies()

	return router
}
