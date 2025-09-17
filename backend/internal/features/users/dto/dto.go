package users_dto

import (
	"time"

	users_enums "logbull/internal/features/users/enums"

	"github.com/google/uuid"
)

type SignUpRequestDTO struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

type SignInRequestDTO struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SignInResponseDTO struct {
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
	Token  string    `json:"token"`
}

type SetAdminPasswordRequestDTO struct {
	Password string `json:"password" binding:"required,min=8"`
}

type IsAdminHasPasswordResponseDTO struct {
	HasPassword bool `json:"hasPassword"`
}

type ChangePasswordRequestDTO struct {
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type InviteUserRequestDTO struct {
	Email               string                   `json:"email"               binding:"required,email"`
	IntendedProjectID   *uuid.UUID               `json:"intendedProjectId"`
	IntendedProjectRole *users_enums.ProjectRole `json:"intendedProjectRole"`
}

type InviteUserResponseDTO struct {
	ID                  uuid.UUID                `json:"id"`
	Email               string                   `json:"email"`
	IntendedProjectID   *uuid.UUID               `json:"intendedProjectId"`
	IntendedProjectRole *users_enums.ProjectRole `json:"intendedProjectRole"`
	CreatedAt           time.Time                `json:"createdAt"`
}

type UserProfileResponseDTO struct {
	ID        uuid.UUID            `json:"id"`
	Email     string               `json:"email"`
	Role      users_enums.UserRole `json:"role"`
	IsActive  bool                 `json:"isActive"`
	CreatedAt time.Time            `json:"createdAt"`
}

type ListUsersResponseDTO struct {
	Users []UserProfileResponseDTO `json:"users"`
	Total int64                    `json:"total"`
}

type ChangeUserRoleRequestDTO struct {
	Role users_enums.UserRole `json:"role" binding:"required"`
}

type ListUsersRequestDTO struct {
	Limit      int        `form:"limit"      json:"limit"`
	Offset     int        `form:"offset"     json:"offset"`
	BeforeDate *time.Time `form:"beforeDate" json:"beforeDate"`
}
