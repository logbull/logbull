package projects_controllers

import (
	"fmt"
	"net/http"
	"testing"

	projects_dto "logbull/internal/features/projects/dto"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ListMembers Tests

func Test_GetProjectMembers_WhenUserIsProjectMember_ReturnsMembers(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	var response projects_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.GreaterOrEqual(t, len(response.Members), 2) // Owner + member

	// Check that owner and member are in the list
	memberEmails := make([]string, len(response.Members))
	memberUserIDs := make([]uuid.UUID, len(response.Members))
	for i, m := range response.Members {
		memberEmails[i] = m.Email
		memberUserIDs[i] = m.UserID
	}
	assert.Contains(t, memberUserIDs, owner.UserID)
	assert.Contains(t, memberUserIDs, member.UserID)
}

func Test_GetProjectMembers_WhenUserIsNotProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+nonMember.Token,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to view project members")
}

func Test_GetProjectMembers_WhenUserIsGlobalAdmin_ReturnsMembers(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	var response projects_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+admin.Token,
		http.StatusOK,
		&response,
	)

	assert.GreaterOrEqual(t, len(response.Members), 1) // At least the owner
}

func Test_GetProjectMembers_WhenUserIsProjectAdmin_ReturnsMembers(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, projectAdmin, users_enums.ProjectRoleAdmin, router)

	var response projects_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+projectAdmin.Token,
		http.StatusOK,
		&response,
	)

	assert.GreaterOrEqual(t, len(response.Members), 2) // Owner + ProjectAdmin
}

func Test_GetProjectMembers_WithInvalidProjectID_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/projects/memberships/invalid-uuid/members",
		"Bearer "+user.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "Invalid project ID")
}

// AddMember Tests

func Test_AddMemberToProject_WhenUserIsProjectOwner_MemberAdded(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	newMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: newMember.Email,
		Role:  users_enums.ProjectRoleMember,
	}

	var response projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.Status == projects_dto.AddStatusAdded)
}

func Test_AddMemberToProject_WhenUserIsGlobalAdmin_MemberAdded(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	newMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: newMember.Email,
		Role:  users_enums.ProjectRoleMember,
	}

	var response projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+admin.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.Status == projects_dto.AddStatusAdded)
}

func Test_AddMemberToProject_WhenUserIsProjectAdmin_MemberAdded(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	newMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, projectAdmin, users_enums.ProjectRoleAdmin, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: newMember.Email,
		Role:  users_enums.ProjectRoleMember,
	}

	var response projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+projectAdmin.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.Status == projects_dto.AddStatusAdded)
}

func Test_AddMemberToProject_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)
	newMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: newMember.Email,
		Role:  users_enums.ProjectRoleMember,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+member.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to manage members")
}

func Test_AddMemberToProject_WhenUserIsAlreadyMember_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	// Try to add the same user again
	request := projects_dto.AddMemberRequestDTO{
		Email: member.Email,
		Role:  users_enums.ProjectRoleMember,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "user is already a member of this project")
}

func Test_AddMemberToProject_WithNonExistentUser_ReturnsInvited(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: uuid.New().String() + "@example.com", // Non-existent user
		Role:  users_enums.ProjectRoleMember,
	}

	var response projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.Status == projects_dto.AddStatusInvited)
}

func Test_AddMemberToProject_WhenProjectAdminTriesToAddAdmin_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	newMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, projectAdmin, users_enums.ProjectRoleAdmin, router)

	// Project admin tries to add another admin (should fail)
	request := projects_dto.AddMemberRequestDTO{
		Email: newMember.Email,
		Role:  users_enums.ProjectRoleAdmin,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+projectAdmin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "only project owner can add/manage admins")
}

func Test_AddMemberToProject_WhenProjectAdminTriesToAddProjectAdmin_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	newMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, projectAdmin, users_enums.ProjectRoleAdmin, router)

	// ProjectAdmin tries to add another ProjectAdmin (should fail)
	request := projects_dto.AddMemberRequestDTO{
		Email: newMember.Email,
		Role:  users_enums.ProjectRoleAdmin,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+projectAdmin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "only project owner can add/manage admins")
}

func Test_InviteMemberToProject_WhenUserIsProjectOwner_MemberInvited(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	users_testing.EnableMemberInvitations()
	defer users_testing.ResetSettingsToDefaults()

	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: fmt.Sprintf("newmember-%s@example.com", uuid.New().String()),
		Role:  users_enums.ProjectRoleMember,
	}

	var response projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.Status == projects_dto.AddStatusInvited)
}

func Test_InviteMemberToProject_WhenUserIsGlobalAdmin_MemberInvited(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	users_testing.EnableMemberInvitations()
	defer users_testing.ResetSettingsToDefaults()

	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: fmt.Sprintf("admin-invite-%s@example.com", uuid.New().String()),
		Role:  users_enums.ProjectRoleMember,
	}

	var response projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+admin.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.Status == projects_dto.AddStatusInvited)
}

func Test_InviteMemberToProject_WhenUserIsProjectAdmin_MemberInvited(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	users_testing.EnableMemberInvitations()
	defer users_testing.ResetSettingsToDefaults()

	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, projectAdmin, users_enums.ProjectRoleAdmin, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: fmt.Sprintf("project-admin-invite-%s@example.com", uuid.New().String()),
		Role:  users_enums.ProjectRoleMember,
	}

	var response projects_dto.AddMemberResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+projectAdmin.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.Status == projects_dto.AddStatusInvited)
}

func Test_InviteMemberToProject_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	request := projects_dto.AddMemberRequestDTO{
		Email: fmt.Sprintf("newmember-%s@example.com", uuid.New().String()),
		Role:  users_enums.ProjectRoleMember,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+member.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to manage members")
}

// ChangeMemberRole Tests

func Test_ChangeMemberRole_WhenUserIsProjectOwner_RoleChanged(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	request := projects_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.ProjectRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/projects/memberships/%s/members/%s/role", project.ID.String(), member.UserID.String()),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Member role changed successfully")
}

func Test_ChangeMemberRole_WhenUserIsProjectAdmin_RoleChanged(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, projectAdmin, users_enums.ProjectRoleAdmin, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	request := projects_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.ProjectRoleMember, // ProjectAdmin can change member roles but not to admin level
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/projects/memberships/%s/members/%s/role", project.ID.String(), member.UserID.String()),
		"Bearer "+projectAdmin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Member role changed successfully")
}

func Test_ChangeMemberRole_WhenUserIsGlobalAdmin_RoleChanged(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	request := projects_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.ProjectRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/projects/memberships/%s/members/%s/role", project.ID.String(), member.UserID.String()),
		"Bearer "+admin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Member role changed successfully")
}

func Test_ChangeMemberRole_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member2 := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member1, users_enums.ProjectRoleMember, router)
	projects_testing.AddMemberToProjectViaOwner(project, member2, users_enums.ProjectRoleMember, router)

	request := projects_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.ProjectRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/projects/memberships/%s/members/%s/role", project.ID.String(), member2.UserID.String()),
		"Bearer "+member1.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions to manage members")
}

func Test_ChangeMemberRole_WhenChangingOwnRole_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := projects_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.ProjectRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/projects/memberships/%s/members/%s/role", project.ID.String(), owner.UserID.String()),
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot change your own role")
}

func Test_ChangeMemberRole_WhenChangingOwnerRole_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := projects_dto.ChangeMemberRoleRequestDTO{
		Role: users_enums.ProjectRoleAdmin,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/projects/memberships/%s/members/%s/role", project.ID.String(), owner.UserID.String()),
		"Bearer "+admin.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cannot change owner role")
}

// RemoveMember Tests

func Test_RemoveMemberFromProject_WhenUserIsProjectOwner_MemberRemoved(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method: "DELETE",
		URL: fmt.Sprintf(
			"/api/v1/projects/memberships/%s/members/%s",
			project.ID.String(),
			member.UserID.String(),
		),
		AuthToken:      "Bearer " + owner.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Member removed successfully")
}

func Test_RemoveMemberFromProject_WhenUserIsGlobalAdmin_MemberRemoved(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method: "DELETE",
		URL: fmt.Sprintf(
			"/api/v1/projects/memberships/%s/members/%s",
			project.ID.String(),
			member.UserID.String(),
		),
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Member removed successfully")
}

func Test_RemoveMemberFromProject_WhenUserIsProjectAdmin_MemberRemoved(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, projectAdmin, users_enums.ProjectRoleAdmin, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleMember, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method: "DELETE",
		URL: fmt.Sprintf(
			"/api/v1/projects/memberships/%s/members/%s",
			project.ID.String(),
			member.UserID.String(),
		),
		AuthToken:      "Bearer " + projectAdmin.Token,
		ExpectedStatus: http.StatusOK,
	})

	assert.Contains(t, string(resp.Body), "Member removed successfully")
}

func Test_RemoveMemberFromProject_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member2 := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member1, users_enums.ProjectRoleMember, router)
	projects_testing.AddMemberToProjectViaOwner(project, member2, users_enums.ProjectRoleMember, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method: "DELETE",
		URL: fmt.Sprintf(
			"/api/v1/projects/memberships/%s/members/%s",
			project.ID.String(),
			member2.UserID.String(),
		),
		AuthToken:      "Bearer " + member1.Token,
		ExpectedStatus: http.StatusForbidden,
	})

	assert.Contains(t, string(resp.Body), "insufficient permissions to remove members")
}

func Test_RemoveMemberFromProject_WhenRemovingOwner_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method: "DELETE",
		URL: fmt.Sprintf(
			"/api/v1/projects/memberships/%s/members/%s",
			project.ID.String(),
			owner.UserID.String(),
		),
		AuthToken:      "Bearer " + admin.Token,
		ExpectedStatus: http.StatusBadRequest,
	})

	assert.Contains(t, string(resp.Body), "cannot remove project owner, transfer ownership first")
}

// TransferOwnership Tests

func Test_TransferProjectOwnership_WhenUserIsProjectOwner_OwnershipTransferred(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleAdmin, router)

	request := projects_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: member.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/transfer-ownership",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Ownership transferred successfully")
}

func Test_TransferProjectOwnership_WhenUserIsGlobalAdmin_OwnershipTransferred(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	admin := users_testing.CreateTestUser(users_enums.UserRoleAdmin)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleAdmin, router)

	request := projects_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: member.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/transfer-ownership",
		"Bearer "+admin.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Ownership transferred successfully")

	// Verify there is only one owner and it's the new owner
	var membersResponse projects_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+member.Token,
		http.StatusOK,
		&membersResponse,
	)

	// Count owners and verify there's exactly one
	ownerCount := 0
	var currentOwner *projects_dto.ProjectMemberResponseDTO
	var previousOwner *projects_dto.ProjectMemberResponseDTO

	for i, m := range membersResponse.Members {
		if m.Role == users_enums.ProjectRoleOwner {
			ownerCount++
			currentOwner = &membersResponse.Members[i]
		}
		if m.UserID == owner.UserID {
			previousOwner = &membersResponse.Members[i]
		}
	}

	assert.Equal(t, 1, ownerCount, "There should be exactly one owner after global admin transfer")
	assert.NotNil(t, currentOwner, "Owner should exist")
	assert.Equal(t, member.UserID, currentOwner.UserID, "The new owner should be the transferred member")
	assert.NotNil(t, previousOwner, "Previous owner should still be a project member")
	assert.Equal(t, users_enums.ProjectRoleAdmin, previousOwner.Role, "Previous owner should now be admin")
}

func Test_TransferProjectOwnership_WhenUserIsProjectMember_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member2 := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member1, users_enums.ProjectRoleMember, router)
	projects_testing.AddMemberToProjectViaOwner(project, member2, users_enums.ProjectRoleAdmin, router)

	request := projects_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: member2.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/transfer-ownership",
		"Bearer "+member1.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "only project owner or admin can transfer ownership")
}

func Test_TransferProjectOwnership_WhenUserIsProjectAdmin_ReturnsForbidden(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	projectAdmin := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, projectAdmin, users_enums.ProjectRoleAdmin, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleAdmin, router)

	request := projects_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: member.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/transfer-ownership",
		"Bearer "+projectAdmin.Token,
		request,
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "only project owner or admin can transfer ownership")
}

func Test_TransferProjectOwnership_WhenNewOwnerIsNotMember_ReturnsBadRequest(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	nonMember := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

	request := projects_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: nonMember.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/transfer-ownership",
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "new owner must be a project member")
}

func Test_TransferProjectOwnership_ThereIsOnlyOneOwner_OldOwnerBecomeAdmin(t *testing.T) {
	router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	member := users_testing.CreateTestUser(users_enums.UserRoleMember)

	project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)
	projects_testing.AddMemberToProjectViaOwner(project, member, users_enums.ProjectRoleAdmin, router)

	// Transfer ownership to the member
	request := projects_dto.TransferOwnershipRequestDTO{
		NewOwnerEmail: member.Email,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/transfer-ownership",
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
	)
	assert.Contains(t, string(resp.Body), "Ownership transferred successfully")

	// Get all members using the new owner's token
	var membersResponse projects_dto.GetMembersResponseDTO
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+member.Token,
		http.StatusOK,
		&membersResponse,
	)

	// Verify there is only one owner
	ownerCount := 0
	var currentOwner *projects_dto.ProjectMemberResponseDTO
	for _, m := range membersResponse.Members {
		if m.Role == users_enums.ProjectRoleOwner {
			ownerCount++
			currentOwner = &m
		}
	}

	assert.Equal(t, 1, ownerCount, "There should be exactly one owner")
	assert.NotNil(t, currentOwner, "Owner should exist")
	assert.Equal(t, member.UserID, currentOwner.UserID, "The new owner should be the member we transferred to")
	assert.Equal(t, member.Email, currentOwner.Email, "Owner email should match the transferred member")

	// verify previous owner is now an admin
	for _, m := range membersResponse.Members {
		if m.UserID == owner.UserID {
			assert.Equal(t, users_enums.ProjectRoleAdmin, m.Role, "Previous owner should now be admin")
			break
		}
	}
}
