package api_keys

import (
	"encoding/json"
	"fmt"
	"net/http"

	projects_testing "logbull/internal/features/projects/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CreateApiKeyTestRouter(additionalControllers ...projects_testing.ControllerInterface) *gin.Engine {
	controllers := []projects_testing.ControllerInterface{GetApiKeyController()}
	controllers = append(controllers, additionalControllers...)
	return projects_testing.CreateTestRouter(controllers...)
}

func CreateTestApiKey(name string, projectID uuid.UUID, ownerToken string, router *gin.Engine) *ApiKey {
	request := CreateApiKeyRequestDTO{
		Name: name,
	}

	w := projects_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/projects/api-keys/"+projectID.String(),
		"Bearer "+ownerToken,
		request,
	)

	if w.Code != http.StatusOK {
		fmt.Printf("Failed to create API key. Status: %d, Body: %s\n", w.Code, w.Body.String())
		panic("Failed to create API key via API")
	}

	var response ApiKey
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		panic(err)
	}

	return &response
}
