package projects_testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"logbull/internal/features/audit_logs"
	projects_dto "logbull/internal/features/projects/dto"
	projects_models "logbull/internal/features/projects/models"
	projects_repositories "logbull/internal/features/projects/repositories"
	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_middleware "logbull/internal/features/users/middleware"
	users_services "logbull/internal/features/users/services"
	users_testing "logbull/internal/features/users/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectConfigurationDTO struct {
	IsApiKeyRequired   bool     `json:"isApiKeyRequired"`
	IsFilterByDomain   bool     `json:"isFilterByDomain"`
	AllowedDomains     []string `json:"allowedDomains"`
	IsFilterByIP       bool     `json:"isFilterByIP"`
	AllowedIPs         []string `json:"allowedIPs"`
	LogsPerSecondLimit int      `json:"logsPerSecondLimit"`
	MaxLogSizeKB       int      `json:"maxLogSizeKB"`
}

func CreateTestRouter(controllers ...ControllerInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	for _, controller := range controllers {
		if routerGroup, ok := protected.(*gin.RouterGroup); ok {
			controller.RegisterRoutes(routerGroup)
		}
	}

	audit_logs.SetupDependencies()

	return router
}

func CreateTestProject(name string, owner *users_dto.SignInResponseDTO, router *gin.Engine) *projects_models.Project {
	project, _ := CreateTestProjectViaAPI(name, owner, router)
	return project
}

func CreateTestProjectViaAPI(
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*projects_models.Project, string) {
	return createTestProjectViaAPI(name, owner, router, true)
}

func CreateTestProjectViaAPIWithoutSettingsChange(
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) (*projects_models.Project, string) {
	return createTestProjectViaAPI(name, owner, router, false)
}

func createTestProjectViaAPI(
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
	enableMemberCreation bool,
) (*projects_models.Project, string) {
	if enableMemberCreation {
		users_testing.EnableMemberProjectCreation()
		defer users_testing.ResetSettingsToDefaults()
	}

	request := projects_dto.CreateProjectRequestDTO{Name: name}
	w := MakeAPIRequest(router, "POST", "/api/v1/projects", "Bearer "+owner.Token, request)

	if w.Code != http.StatusOK {
		panic(fmt.Sprintf("Failed to create project. Status: %d, Body: %s", w.Code, w.Body.String()))
	}

	var response projects_dto.ProjectResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	project := &projects_models.Project{
		ID:   response.ID,
		Name: response.Name,
	}

	return project, owner.Token
}

func CreateTestProjectWithToken(name string, token string, router *gin.Engine) (*projects_models.Project, string) {
	return createTestProjectWithToken(name, token, router, true)
}

func CreateTestProjectWithTokenWithoutSettingsChange(
	name string,
	token string,
	router *gin.Engine,
) (*projects_models.Project, string) {
	return createTestProjectWithToken(name, token, router, false)
}

func createTestProjectWithToken(
	name string,
	token string,
	router *gin.Engine,
	enableMemberCreation bool,
) (*projects_models.Project, string) {
	if enableMemberCreation {
		users_testing.EnableMemberProjectCreation()
		defer users_testing.ResetSettingsToDefaults()
	}

	request := projects_dto.CreateProjectRequestDTO{Name: name}
	w := MakeAPIRequest(router, "POST", "/api/v1/projects", "Bearer "+token, request)

	if w.Code != http.StatusOK {
		panic(fmt.Sprintf("Failed to create project. Status: %d, Body: %s", w.Code, w.Body.String()))
	}

	var response projects_dto.ProjectResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	project := &projects_models.Project{
		ID:   response.ID,
		Name: response.Name,
	}

	return project, token
}

func AddMemberToProject(
	project *projects_models.Project,
	member *users_dto.SignInResponseDTO,
	role users_enums.ProjectRole,
	ownerToken string,
	router *gin.Engine,
) {
	request := projects_dto.AddMemberRequestDTO{
		Email: member.Email,
		Role:  role,
	}

	w := MakeAPIRequest(
		router,
		"POST",
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+ownerToken,
		request,
	)

	if w.Code != http.StatusOK {
		panic("Failed to add member to project via API: " + w.Body.String())
	}
}

func AddMemberToProjectViaOwner(
	project *projects_models.Project,
	member *users_dto.SignInResponseDTO,
	role users_enums.ProjectRole,
	router *gin.Engine,
) {
	membershipRepo := &projects_repositories.MembershipRepository{}
	projectMembers, err := membershipRepo.GetProjectMembers(project.ID)
	if err != nil {
		panic("Failed to get project members: " + err.Error())
	}

	var ownerToken string
	for _, m := range projectMembers {
		if m.Role == users_enums.ProjectRoleOwner {
			userService := users_services.GetUserService()

			owner, err := userService.GetUserByID(m.UserID)
			if err != nil {
				panic("Failed to get owner user: " + err.Error())
			}

			tokenResponse, err := userService.GenerateAccessToken(owner)
			if err != nil {
				panic("Failed to generate owner token: " + err.Error())
			}

			ownerToken = tokenResponse.Token

			break
		}
	}

	if ownerToken == "" {
		panic("No project owner found")
	}

	AddMemberToProject(project, member, role, ownerToken, router)
}

func InviteMemberToProject(
	project *projects_models.Project,
	email string,
	role users_enums.ProjectRole,
	inviterToken string,
	router *gin.Engine,
) *projects_dto.AddMemberResponseDTO {
	request := projects_dto.AddMemberRequestDTO{
		Email: email,
		Role:  role,
	}

	w := MakeAPIRequest(
		router,
		"POST",
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+inviterToken,
		request,
	)

	if w.Code != http.StatusOK {
		panic("Failed to invite member to project via API: " + w.Body.String())
	}

	var response projects_dto.AddMemberResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	return &response
}

func ChangeMemberRole(
	project *projects_models.Project,
	memberUserID uuid.UUID,
	newRole users_enums.ProjectRole,
	changerToken string,
	router *gin.Engine,
) {
	request := projects_dto.ChangeMemberRoleRequestDTO{
		Role: newRole,
	}

	w := MakeAPIRequest(
		router,
		"PUT",
		fmt.Sprintf("/api/v1/projects/memberships/%s/members/%s/role", project.ID.String(), memberUserID.String()),
		"Bearer "+changerToken,
		request,
	)

	if w.Code != http.StatusOK {
		panic("Failed to change member role via API: " + w.Body.String())
	}
}

func RemoveMemberFromProject(
	project *projects_models.Project,
	memberUserID uuid.UUID,
	removerToken string,
	router *gin.Engine,
) {
	w := MakeAPIRequest(
		router,
		"DELETE",
		fmt.Sprintf("/api/v1/projects/memberships/%s/members/%s", project.ID.String(), memberUserID.String()),
		"Bearer "+removerToken,
		nil,
	)

	if w.Code != http.StatusOK {
		panic("Failed to remove member from project via API: " + w.Body.String())
	}
}

func GetProjectMembers(
	project *projects_models.Project,
	requesterToken string,
	router *gin.Engine,
) *projects_dto.GetMembersResponseDTO {
	w := MakeAPIRequest(
		router,
		"GET",
		"/api/v1/projects/memberships/"+project.ID.String()+"/members",
		"Bearer "+requesterToken,
		nil,
	)

	if w.Code != http.StatusOK {
		panic("Failed to get project members via API: " + w.Body.String())
	}

	var response projects_dto.GetMembersResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	return &response
}

func UpdateProject(
	project *projects_models.Project,
	updateData *projects_models.Project,
	updaterToken string,
	router *gin.Engine,
) *projects_models.Project {
	w := MakeAPIRequest(
		router,
		"PUT",
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+updaterToken,
		updateData,
	)

	if w.Code != http.StatusOK {
		panic("Failed to update project via API: " + w.Body.String())
	}

	var response projects_models.Project
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	return &response
}

func DeleteProject(project *projects_models.Project, deleterToken string, router *gin.Engine) {
	w := MakeAPIRequest(
		router,
		"DELETE",
		"/api/v1/projects/"+project.ID.String(),
		"Bearer "+deleterToken,
		nil,
	)

	if w.Code != http.StatusOK {
		panic("Failed to delete project via API: " + w.Body.String())
	}
}

func MakeAPIRequest(router *gin.Engine, method, url, authToken string, body any) *httptest.ResponseRecorder {
	var requestBody *bytes.Buffer
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		requestBody = bytes.NewBuffer(bodyJSON)
	} else {
		requestBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		panic(err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func CreateTestProjectWithConfiguration(
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
	config *ProjectConfigurationDTO,
) *projects_models.Project {
	project := CreateTestProject(name, owner, router)

	updateData := &projects_models.Project{
		Name:               project.Name,
		IsApiKeyRequired:   config.IsApiKeyRequired,
		IsFilterByDomain:   config.IsFilterByDomain,
		AllowedDomains:     config.AllowedDomains,
		IsFilterByIP:       config.IsFilterByIP,
		AllowedIPs:         config.AllowedIPs,
		LogsPerSecondLimit: config.LogsPerSecondLimit,
		MaxLogSizeKB:       config.MaxLogSizeKB,
	}

	return UpdateProject(project, updateData, owner.Token, router)
}

func CreateBasicTestProject(
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) *projects_models.Project {
	config := &ProjectConfigurationDTO{
		IsApiKeyRequired:   false,
		IsFilterByDomain:   false,
		AllowedDomains:     nil,
		IsFilterByIP:       false,
		AllowedIPs:         nil,
		LogsPerSecondLimit: 1000,
		MaxLogSizeKB:       64,
	}
	return CreateTestProjectWithConfiguration(name, owner, router, config)
}

func CreateSecureTestProject(
	name string,
	owner *users_dto.SignInResponseDTO,
	router *gin.Engine,
) *projects_models.Project {
	config := &ProjectConfigurationDTO{
		IsApiKeyRequired:   true,
		IsFilterByDomain:   true,
		AllowedDomains:     []string{"example.com"},
		IsFilterByIP:       true,
		AllowedIPs:         []string{"192.168.1.0/24"},
		LogsPerSecondLimit: 100,
		MaxLogSizeKB:       64,
	}
	return CreateTestProjectWithConfiguration(name, owner, router, config)
}
