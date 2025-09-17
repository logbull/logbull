package logs_querying_tests

import (
	"fmt"
	"net/http"
	"testing"

	projects_testing "logbull/internal/features/projects/testing"
	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
)

func Test_ExecuteQuery_WithDifferentUserRoles_ReturnsLogsBasedOnPermissions(t *testing.T) {
	testCases := []struct {
		name              string
		userRole          users_enums.UserRole
		projectRole       *users_enums.ProjectRole
		logCount          int
		expectedStatus    int
		needsProjectOwner bool
	}{
		{
			name:           "Project Member",
			userRole:       users_enums.UserRoleMember,
			projectRole:    &[]users_enums.ProjectRole{users_enums.ProjectRoleMember}[0],
			logCount:       5,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Project Owner",
			userRole:       users_enums.UserRoleMember,
			projectRole:    nil, // Owner is the creator
			logCount:       3,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Project Admin",
			userRole:       users_enums.UserRoleMember,
			projectRole:    &[]users_enums.ProjectRole{users_enums.ProjectRoleAdmin}[0],
			logCount:       4,
			expectedStatus: http.StatusOK,
		},
		{
			name:              "Global Admin",
			userRole:          users_enums.UserRoleAdmin,
			projectRole:       nil,
			logCount:          2,
			expectedStatus:    http.StatusOK,
			needsProjectOwner: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := CreateLogQueryTestRouter()

			var owner, testUser *users_dto.SignInResponseDTO

			if tc.needsProjectOwner {
				// Global admin test: create separate project owner
				owner = users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUser = users_testing.CreateTestUser(tc.userRole)
			} else if tc.projectRole == nil {
				// Project owner test: user is both owner and test user
				owner = users_testing.CreateTestUser(tc.userRole)
				testUser = owner
			} else {
				// Member/Admin test: create owner and separate test user
				owner = users_testing.CreateTestUser(users_enums.UserRoleMember)
				testUser = users_testing.CreateTestUser(tc.userRole)
			}

			uniqueID := uuid.New().String()
			projectName := fmt.Sprintf("%s Test %s", tc.name, uniqueID[:8])
			project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

			// Submit logs using helper function
			SubmitLogsWithCustomFields(t, router, project.ID, uniqueID, tc.logCount, map[string]any{
				"role_test": tc.name,
			})

			// Add project role if needed
			if tc.projectRole != nil {
				projects_testing.AddMemberToProject(project, testUser, *tc.projectRole, owner.Token, router)
			}

			WaitForLogsToBeIndexed(t, router, project.ID, tc.logCount, uniqueID, "Bearer "+testUser.Token)

			query := BuildSimpleConditionQuery("test_id", "equals", uniqueID)
			queryResponse := ExecuteTestQuery(t, router, project.ID, query, testUser.Token, tc.expectedStatus)

			if tc.expectedStatus == http.StatusOK {
				AssertQueryResponseValid(t, queryResponse, 1)
				AssertLogContainsUniqueID(t, queryResponse.Logs, uniqueID, tc.logCount)
			}
		})
	}
}

func Test_ExecuteQuery_WhenUserIsNotProjectMember_ReturnsForbidden(t *testing.T) {
	router, _, project, uniqueID := SetupBasicQueryTest(t, "Non-Member Test")
	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	// Submit logs using helper function
	SubmitLogsWithCustomFields(t, router, project.ID, uniqueID, 1, map[string]any{
		"access_test": "forbidden",
	})

	// Non-member should not be able to query project logs
	query := BuildSimpleConditionQuery("test_id", "equals", uniqueID)
	ExecuteTestQuery(t, router, project.ID, query, nonMember.Token, http.StatusForbidden)
}

func Test_ExecuteQuery_WithoutAuthToken_ReturnsUnauthorized(t *testing.T) {
	router, _, project, uniqueID := SetupBasicQueryTest(t, "No Auth Test")

	query := BuildSimpleConditionQuery("test_id", "equals", uniqueID)

	// Make request without Authorization header (empty token)
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/query/execute/%s", project.ID.String()),
		Body:           query,
		ExpectedStatus: http.StatusUnauthorized,
	})

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d but got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func Test_GetQueryableFields_WhenUserIsNotProjectMember_ReturnsForbidden(t *testing.T) {
	router, _, project, _ := SetupBasicQueryTest(t, "Non-Member Fields Test")
	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/logs/query/fields/%s", project.ID.String()),
		"Bearer "+nonMember.Token,
		http.StatusForbidden,
	)

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status %d but got %d", http.StatusForbidden, resp.StatusCode)
	}
}

func Test_GetQueryableFields_WithoutAuthToken_ReturnsUnauthorized(t *testing.T) {
	router, _, project, _ := SetupBasicQueryTest(t, "No Auth Fields Test")

	resp := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/logs/query/fields/%s", project.ID.String()),
		"", // No auth token
		http.StatusUnauthorized,
	)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status %d but got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}
