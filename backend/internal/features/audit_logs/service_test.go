package audit_logs

import (
	"testing"
	"time"

	user_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_AuditLogs_ProjectSpecificLogs(t *testing.T) {
	service := GetAuditLogService()
	user1 := users_testing.CreateTestUser(user_enums.UserRoleMember)
	user2 := users_testing.CreateTestUser(user_enums.UserRoleMember)
	project1ID, project2ID := uuid.New(), uuid.New()

	// Create test logs for projects
	createAuditLog(service, "Test project1 log first", &user1.UserID, &project1ID)
	createAuditLog(service, "Test project1 log second", &user2.UserID, &project1ID)
	createAuditLog(service, "Test project2 log first", &user1.UserID, &project2ID)
	createAuditLog(service, "Test project2 log second", &user2.UserID, &project2ID)
	createAuditLog(service, "Test no project log", &user1.UserID, nil)

	request := &GetAuditLogsRequest{Limit: 10, Offset: 0}

	// Test project 1 logs
	project1Response, err := service.GetProjectAuditLogs(project1ID, request)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(project1Response.AuditLogs))

	messages := extractMessages(project1Response.AuditLogs)
	assert.Contains(t, messages, "Test project1 log first")
	assert.Contains(t, messages, "Test project1 log second")
	for _, log := range project1Response.AuditLogs {
		assert.Equal(t, &project1ID, log.ProjectID)
	}

	// Test project 2 logs
	project2Response, err := service.GetProjectAuditLogs(project2ID, request)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(project2Response.AuditLogs))

	messages2 := extractMessages(project2Response.AuditLogs)
	assert.Contains(t, messages2, "Test project2 log first")
	assert.Contains(t, messages2, "Test project2 log second")

	// Test pagination
	limitedResponse, err := service.GetProjectAuditLogs(project1ID,
		&GetAuditLogsRequest{Limit: 1, Offset: 0})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(limitedResponse.AuditLogs))
	assert.Equal(t, 1, limitedResponse.Limit)

	// Test beforeDate filter
	beforeTime := time.Now().UTC().Add(-1 * time.Minute)
	filteredResponse, err := service.GetProjectAuditLogs(project1ID,
		&GetAuditLogsRequest{Limit: 10, BeforeDate: &beforeTime})
	assert.NoError(t, err)
	for _, log := range filteredResponse.AuditLogs {
		assert.True(t, log.CreatedAt.Before(beforeTime))
		assert.NotNil(t, log.UserEmail, "User email should be present for logs with user_id")
		assert.NotNil(t, log.ProjectName, "Project name should be present for logs with project_id")
	}
}

func createAuditLog(service *AuditLogService, message string, userID, projectID *uuid.UUID) {
	service.WriteAuditLog(message, userID, projectID)
}

func extractMessages(logs []*AuditLogDTO) []string {
	messages := make([]string, len(logs))
	for i, log := range logs {
		messages[i] = log.Message
	}
	return messages
}
