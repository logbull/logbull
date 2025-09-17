package logs_querying_tests

import (
	"fmt"
	"net/http"
	"testing"

	logs_core "logbull/internal/features/logs/core"
	logs_receiving_tests "logbull/internal/features/logs/receiving/tests"
	projects_models "logbull/internal/features/projects/models"
	projects_testing "logbull/internal/features/projects/testing"
	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_ExecuteQuery_BasicProjectIsolation_EachProjectSeesOnlyOwnLogs(t *testing.T) {
	router, project1, project2, owner1, owner2, uniqueID1, uniqueID2 := setupTwoProjectsWithLogs(t)

	// Query first project for its own logs
	project1Query := BuildSimpleConditionQuery("test_id", "equals", uniqueID1)
	project1Response := ExecuteTestQuery(t, router, project1.ID, project1Query, owner1.Token, http.StatusOK)

	AssertQueryResponseValid(t, project1Response, 1)
	AssertLogContainsUniqueID(t, project1Response.Logs, uniqueID1, 5)

	// Verify all returned logs belong to project1
	for _, log := range project1Response.Logs {
		assertLogBelongsToProject(t, log, "project_1", uniqueID1)
	}

	// Query second project for its own logs
	project2Query := BuildSimpleConditionQuery("test_id", "equals", uniqueID2)
	project2Response := ExecuteTestQuery(t, router, project2.ID, project2Query, owner2.Token, http.StatusOK)

	AssertQueryResponseValid(t, project2Response, 1)
	AssertLogContainsUniqueID(t, project2Response.Logs, uniqueID2, 3)

	// Verify all returned logs belong to project2
	for _, log := range project2Response.Logs {
		assertLogBelongsToProject(t, log, "project_2", uniqueID2)
	}

	t.Logf("Basic isolation successful: Project 1 returned %d logs, Project 2 returned %d logs",
		len(project1Response.Logs), len(project2Response.Logs))
}

func Test_ExecuteQuery_CrossProjectAccess_ReturnsNoResults(t *testing.T) {
	router, project1, project2, owner1, owner2, uniqueID1, uniqueID2 := setupTwoProjectsWithLogs(t)

	// Owner1 tries to query project1 for project2's data
	crossQuery1 := BuildSimpleConditionQuery("test_id", "equals", uniqueID2)
	crossResponse1 := ExecuteTestQuery(t, router, project1.ID, crossQuery1, owner1.Token, http.StatusOK)

	assert.Equal(t, 0, len(crossResponse1.Logs),
		"Querying project 1 for project 2's data should return no logs")

	// Owner2 tries to query project2 for project1's data
	crossQuery2 := BuildSimpleConditionQuery("test_id", "equals", uniqueID1)
	crossResponse2 := ExecuteTestQuery(t, router, project2.ID, crossQuery2, owner2.Token, http.StatusOK)

	assert.Equal(t, 0, len(crossResponse2.Logs),
		"Querying project 2 for project 1's data should return no logs")

	t.Logf("Cross-project access prevention successful: both cross-project queries returned 0 logs")
}

func Test_ExecuteQuery_BroadSearchQueries_MaintainProjectIsolation(t *testing.T) {
	router, project1, project2, owner1, owner2, uniqueID1, uniqueID2 := setupTwoProjectsWithLogs(t)

	// Broad search query that would match both projects if not properly isolated
	broadQuery := BuildSimpleConditionQuery("log_source", "contains", "project")

	// Query project1 with broad search - should only return project1's logs
	project1BroadResponse := ExecuteTestQuery(t, router, project1.ID, broadQuery, owner1.Token, http.StatusOK)

	for _, log := range project1BroadResponse.Logs {
		assertLogDoesNotBelongToOtherProject(t, log, "project_2", uniqueID2,
			"Project 1 broad search should not return project 2's logs")
	}

	// Query project2 with broad search - should only return project2's logs
	project2BroadResponse := ExecuteTestQuery(t, router, project2.ID, broadQuery, owner2.Token, http.StatusOK)

	for _, log := range project2BroadResponse.Logs {
		assertLogDoesNotBelongToOtherProject(t, log, "project_1", uniqueID1,
			"Project 2 broad search should not return project 1's logs")
	}

	t.Logf("Broad search isolation successful: Project 1 returned %d logs, Project 2 returned %d logs",
		len(project1BroadResponse.Logs), len(project2BroadResponse.Logs))
}

func Test_ExecuteQuery_UserAccessIsolation_DeniesAccessToOtherProjects(t *testing.T) {
	router, project1, project2, owner1, owner2, uniqueID1, uniqueID2 := setupTwoProjectsWithLogs(t)

	// Owner2 attempts to access project2 using owner1's data identifiers
	unauthorizedQuery1 := BuildSimpleConditionQuery("test_id", "equals", uniqueID1)
	unauthorizedResponse1 := ExecuteTestQuery(t, router, project2.ID, unauthorizedQuery1, owner2.Token, http.StatusOK)

	assert.Equal(t, 0, len(unauthorizedResponse1.Logs),
		"Owner2 should not access owner1's data even with correct identifiers")

	// Owner1 attempts to access project1 using owner2's data identifiers
	unauthorizedQuery2 := BuildSimpleConditionQuery("test_id", "equals", uniqueID2)
	unauthorizedResponse2 := ExecuteTestQuery(t, router, project1.ID, unauthorizedQuery2, owner1.Token, http.StatusOK)

	assert.Equal(t, 0, len(unauthorizedResponse2.Logs),
		"Owner1 should not access owner2's data even with correct identifiers")

	t.Logf("User access isolation successful: both unauthorized access attempts returned 0 logs")
}

func Test_ExecuteQuery_MultipleProjectsComprehensive_AllIsolationMechanismsWork(t *testing.T) {
	router, project1, project2, owner1, owner2, uniqueID1, uniqueID2 := setupTwoProjectsWithLogs(t)

	// Create a third project to test more complex isolation scenarios
	owner3 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID3 := uuid.New().String()
	project3Name := fmt.Sprintf("Project 3 Isolation Test %s", uniqueID3[:8])
	project3, _ := projects_testing.CreateTestProjectWithToken(project3Name, owner3.Token, router)

	// Submit logs to third project
	testLogs3 := logs_receiving_tests.CreateValidLogItems(4, uniqueID3)
	for i := range testLogs3 {
		testLogs3[i].Fields["project_name"] = project3Name
		testLogs3[i].Fields["log_source"] = "project_3"
	}
	SubmitLogsAndProcess(t, router, project3.ID, testLogs3)
	WaitForLogsToBeIndexed(t, router, project3.ID, 4, uniqueID3, "Bearer "+owner3.Token)

	// Test 1: Each project returns only its own logs
	projects := []struct {
		project  *projects_models.Project
		owner    *users_dto.SignInResponseDTO
		uniqueID string
		expected int
		source   string
	}{
		{project1, owner1, uniqueID1, 5, "project_1"},
		{project2, owner2, uniqueID2, 3, "project_2"},
		{project3, owner3, uniqueID3, 4, "project_3"},
	}

	for _, p := range projects {
		query := BuildSimpleConditionQuery("test_id", "equals", p.uniqueID)
		response := ExecuteTestQuery(t, router, p.project.ID, query, p.owner.Token, http.StatusOK)

		AssertQueryResponseValid(t, response, 1)
		AssertLogContainsUniqueID(t, response.Logs, p.uniqueID, p.expected)

		for _, log := range response.Logs {
			assertLogBelongsToProject(t, log, p.source, p.uniqueID)
		}
	}

	// Test 2: Cross-project queries return no results (comprehensive matrix)
	crossProjectTests := []struct {
		queryProject *projects_models.Project
		queryOwner   *users_dto.SignInResponseDTO
		targetID     string
		description  string
	}{
		{project1, owner1, uniqueID2, "Project 1 querying for Project 2 data"},
		{project1, owner1, uniqueID3, "Project 1 querying for Project 3 data"},
		{project2, owner2, uniqueID1, "Project 2 querying for Project 1 data"},
		{project2, owner2, uniqueID3, "Project 2 querying for Project 3 data"},
		{project3, owner3, uniqueID1, "Project 3 querying for Project 1 data"},
		{project3, owner3, uniqueID2, "Project 3 querying for Project 2 data"},
	}

	for _, test := range crossProjectTests {
		query := BuildSimpleConditionQuery("test_id", "equals", test.targetID)
		response := ExecuteTestQuery(t, router, test.queryProject.ID, query, test.queryOwner.Token, http.StatusOK)

		assert.Equal(t, 0, len(response.Logs), test.description+" should return no results")
	}

	t.Logf("Comprehensive isolation test successful: all 3 projects properly isolated")
}

// setupTwoProjectsWithLogs creates two projects with test data for isolation testing
func setupTwoProjectsWithLogs(t *testing.T) (
	router *gin.Engine,
	project1, project2 *projects_models.Project,
	owner1, owner2 *users_dto.SignInResponseDTO,
	uniqueID1, uniqueID2 string,
) {
	router = CreateLogQueryTestRouter()

	// Create two separate users and projects
	owner1 = users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner2 = users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID1 = uuid.New().String()
	uniqueID2 = uuid.New().String()

	project1Name := fmt.Sprintf("Project 1 Isolation Test %s", uniqueID1[:8])
	project2Name := fmt.Sprintf("Project 2 Isolation Test %s", uniqueID2[:8])

	project1, _ = projects_testing.CreateTestProjectWithToken(project1Name, owner1.Token, router)
	project2, _ = projects_testing.CreateTestProjectWithToken(project2Name, owner2.Token, router)

	// Submit logs to first project
	testLogs1 := logs_receiving_tests.CreateValidLogItems(5, uniqueID1)
	for i := range testLogs1 {
		testLogs1[i].Fields["project_name"] = project1Name
		testLogs1[i].Fields["log_source"] = "project_1"
	}
	SubmitLogsAndProcess(t, router, project1.ID, testLogs1)
	WaitForLogsToBeIndexed(t, router, project1.ID, 5, uniqueID1, "Bearer "+owner1.Token)

	// Submit logs to second project
	testLogs2 := logs_receiving_tests.CreateValidLogItems(3, uniqueID2)
	for i := range testLogs2 {
		testLogs2[i].Fields["project_name"] = project2Name
		testLogs2[i].Fields["log_source"] = "project_2"
	}
	SubmitLogsAndProcess(t, router, project2.ID, testLogs2)
	WaitForLogsToBeIndexed(t, router, project2.ID, 3, uniqueID2, "Bearer "+owner2.Token)

	return
}

// assertLogBelongsToProject verifies a log belongs to the expected project
func assertLogBelongsToProject(t *testing.T, log logs_core.LogItemDTO, expectedSource, expectedUniqueID string) {
	if logSource, exists := log.Fields["log_source"]; exists {
		assert.Equal(t, expectedSource, logSource,
			"Log should have log_source=%s", expectedSource)
	}
	if testID, exists := log.Fields["test_id"]; exists {
		assert.Equal(t, expectedUniqueID, testID,
			"Log should have correct test_id")
	}
}

// assertLogDoesNotBelongToOtherProject verifies a log does not belong to another project
func assertLogDoesNotBelongToOtherProject(
	t *testing.T,
	log logs_core.LogItemDTO,
	excludedSource, excludedUniqueID, message string,
) {
	if logSource, exists := log.Fields["log_source"]; exists {
		assert.NotEqual(t, excludedSource, logSource, message)
	}
	if testID, exists := log.Fields["test_id"]; exists {
		assert.NotEqual(t, excludedUniqueID, testID, message)
	}
}
