package users_controllers

import (
	"fmt"
	"net/http"
	"testing"

	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_services "logbull/internal/features/users/services"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_SignUpUser_WithValidData_UserCreated(t *testing.T) {
	router := createUserTestRouter()

	request := users_dto.SignUpRequestDTO{
		Email:    "test" + uuid.New().String() + "@example.com",
		Password: "testpassword123",
	}

	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", request, http.StatusOK)
}

func Test_SignUpUser_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/signup",
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_SignUpUser_WithDuplicateEmail_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	email := "duplicate" + uuid.New().String() + "@example.com"

	request := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: "testpassword123",
	}

	// First signup
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", request, http.StatusOK)

	// Second signup with same email
	resp := test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", request, http.StatusBadRequest)
	assert.Contains(t, string(resp.Body), "already exists")
}

func Test_SignUpUser_WithValidationErrors_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	testCases := []struct {
		name    string
		request users_dto.SignUpRequestDTO
	}{
		{
			name: "missing email",
			request: users_dto.SignUpRequestDTO{
				Password: "testpassword123",
			},
		},
		{
			name: "missing password",
			request: users_dto.SignUpRequestDTO{
				Email: "test@example.com",
			},
		},
		{
			name: "short password",
			request: users_dto.SignUpRequestDTO{
				Email:    "test@example.com",
				Password: "short",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", tc.request, http.StatusBadRequest)
		})
	}
}

func Test_SignInUser_WithValidCredentials_ReturnsToken(t *testing.T) {
	router := createUserTestRouter()
	email := "signin" + uuid.New().String() + "@example.com"
	password := "testpassword123"

	// First create a user
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: password,
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Now sign in
	signinRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: password,
	}

	var response users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		signinRequest,
		http.StatusOK,
		&response,
	)

	assert.NotEmpty(t, response.Token)
	assert.NotEqual(t, uuid.Nil, response.UserID)
}

func Test_SignInUser_WithWrongPassword_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	email := "signin2" + uuid.New().String() + "@example.com"

	// First create a user
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: "testpassword123",
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Now sign in with wrong password
	signinRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: "wrongpassword",
	}

	resp := test_utils.MakePostRequest(t, router, "/api/v1/users/signin", "", signinRequest, http.StatusBadRequest)
	assert.Contains(t, string(resp.Body), "password is incorrect")
}

func Test_SignInUser_WithNonExistentUser_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	signinRequest := users_dto.SignInRequestDTO{
		Email:    "nonexistent" + uuid.New().String() + "@example.com",
		Password: "testpassword123",
	}

	resp := test_utils.MakePostRequest(t, router, "/api/v1/users/signin", "", signinRequest, http.StatusBadRequest)
	assert.Contains(t, string(resp.Body), "does not exist")
}

func Test_SignInUser_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/signin",
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_CheckAdminHasPassword_WhenAdminHasNoPassword_ReturnsFalse(t *testing.T) {
	router := createUserTestRouter()

	users_testing.RecreateInitialAdmin()

	var response users_dto.IsAdminHasPasswordResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/users/admin/has-password", "", http.StatusOK, &response)

	assert.False(t, response.HasPassword)
}

func Test_SetAdminPassword_WithValidPassword_PasswordSet(t *testing.T) {
	router := createUserTestRouter()

	users_testing.RecreateInitialAdmin()

	request := users_dto.SetAdminPasswordRequestDTO{
		Password: "adminpassword123",
	}

	test_utils.MakePostRequest(t, router, "/api/v1/users/admin/set-password", "", request, http.StatusOK)

	// Now check that admin has password
	var hasPasswordResponse users_dto.IsAdminHasPasswordResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/admin/has-password",
		"",
		http.StatusOK,
		&hasPasswordResponse,
	)

	assert.True(t, hasPasswordResponse.HasPassword)
}

func Test_SetAdminPassword_WithInvalidPassword_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	testCases := []struct {
		name     string
		password string
	}{
		{
			name:     "short password",
			password: "short",
		},
		{
			name:     "empty password",
			password: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := users_dto.SetAdminPasswordRequestDTO{
				Password: tc.password,
			}

			test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/users/admin/set-password",
				"",
				request,
				http.StatusBadRequest,
			)
		})
	}
}

func Test_SetAdminPassword_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/admin/set-password",
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_ChangeUserPassword_WithValidData_PasswordChanged(t *testing.T) {
	router := createUserTestRouter()
	email := "changepass" + uuid.New().String() + "@example.com"
	oldPassword := "oldpassword123"
	newPassword := "newpassword123"

	// Create user via signup
	signupRequest := users_dto.SignUpRequestDTO{
		Email:    email,
		Password: oldPassword,
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signup", "", signupRequest, http.StatusOK)

	// Sign in to get token
	signinRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: oldPassword,
	}
	var signinResponse users_dto.SignInResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/signin",
		"",
		signinRequest,
		http.StatusOK,
		&signinResponse,
	)

	// Change password
	changePasswordRequest := users_dto.ChangePasswordRequestDTO{
		NewPassword: newPassword,
	}

	test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/users/change-password",
		"Bearer "+signinResponse.Token,
		changePasswordRequest,
		http.StatusOK,
	)

	// Verify old password no longer works
	oldSigninRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: oldPassword,
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signin", "", oldSigninRequest, http.StatusBadRequest)

	// Verify new password works
	newSigninRequest := users_dto.SignInRequestDTO{
		Email:    email,
		Password: newPassword,
	}
	test_utils.MakePostRequest(t, router, "/api/v1/users/signin", "", newSigninRequest, http.StatusOK)
}

func Test_ChangeUserPassword_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	router := createUserTestRouter()

	request := users_dto.ChangePasswordRequestDTO{
		NewPassword: "newpassword123",
	}

	test_utils.MakePutRequest(t, router, "/api/v1/users/change-password", "", request, http.StatusUnauthorized)
}

func Test_ChangeUserPassword_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	testUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "PUT",
		URL:            "/api/v1/users/change-password",
		Body:           "invalid json",
		AuthToken:      "Bearer " + testUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_ChangeUserPassword_WithValidationErrors_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	testUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	testCases := []struct {
		name    string
		request users_dto.ChangePasswordRequestDTO
	}{
		{
			name:    "missing new password",
			request: users_dto.ChangePasswordRequestDTO{},
		},
		{
			name: "short new password",
			request: users_dto.ChangePasswordRequestDTO{
				NewPassword: "short",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test_utils.MakePutRequest(
				t,
				router,
				"/api/v1/users/change-password",
				"Bearer "+testUser.Token,
				tc.request,
				http.StatusBadRequest,
			)
		})
	}
}

func Test_InviteUser_WhenUserIsAdmin_UserInvited(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	projectID := uuid.New()
	projectRole := users_enums.ProjectRoleAdmin

	request := users_dto.InviteUserRequestDTO{
		Email:               "invited" + uuid.New().String() + "@example.com",
		IntendedProjectID:   &projectID,
		IntendedProjectRole: &projectRole,
	}

	var response users_dto.InviteUserResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+adminUser.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, request.Email, response.Email)
	assert.Equal(t, request.IntendedProjectID, response.IntendedProjectID)
	assert.Equal(t, request.IntendedProjectRole, response.IntendedProjectRole)
	assert.NotEqual(t, uuid.Nil, response.ID)
}

func Test_InviteUser_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	router := createUserTestRouter()

	request := users_dto.InviteUserRequestDTO{
		Email: "invited@example.com",
	}

	test_utils.MakePostRequest(t, router, "/api/v1/users/invite", "", request, http.StatusUnauthorized)
}

func Test_InviteUser_WithoutPermission_ReturnsForbidden(t *testing.T) {
	router := createUserTestRouter()
	defer users_testing.ResetSettingsToDefaults()

	memberUser := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()[:8]
	request := users_dto.InviteUserRequestDTO{
		Email: fmt.Sprintf("invited_%s@example.com", uniqueID),
	}

	users_testing.DisableMemberInvitations()

	settingsService := users_services.GetSettingsService()
	settings, err := settingsService.GetSettings()
	assert.NoError(t, err)

	if settings.IsAllowMemberInvitations {
		t.Fatal("RACE CONDITION DETECTED: Member invitations should be disabled but were enabled by another test")
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+memberUser.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions")
}

func Test_InviteUser_WithInvalidJSON_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	// Test with invalid JSON structure
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            "/api/v1/users/invite",
		Body:           "invalid json",
		AuthToken:      "Bearer " + adminUser.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_InviteUser_WithValidationErrors_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)

	testCases := []struct {
		name    string
		request users_dto.InviteUserRequestDTO
	}{
		{
			name: "missing email",
			request: users_dto.InviteUserRequestDTO{
				IntendedProjectID: &uuid.UUID{},
			},
		},
		{
			name: "invalid email",
			request: users_dto.InviteUserRequestDTO{
				Email: "invalid-email",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test_utils.MakePostRequest(
				t,
				router,
				"/api/v1/users/invite",
				"Bearer "+adminUser.Token,
				tc.request,
				http.StatusBadRequest,
			)
		})
	}
}

func Test_InviteUser_WithDuplicateEmail_ReturnsBadRequest(t *testing.T) {
	router := createUserTestRouter()
	adminUser := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	email := "duplicate-invite" + uuid.New().String() + "@example.com"

	request := users_dto.InviteUserRequestDTO{
		Email: email,
	}

	// First invitation
	test_utils.MakePostRequest(t, router, "/api/v1/users/invite", "Bearer "+adminUser.Token, request, http.StatusOK)

	// Second invitation with same email
	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/users/invite",
		"Bearer "+adminUser.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "already exists")
}
