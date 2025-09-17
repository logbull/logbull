package users_models

import "github.com/google/uuid"

type UsersSettings struct {
	ID uuid.UUID `json:"id"                              gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	// means that any user can register via sign up form without invitation
	IsAllowExternalRegistrations bool `json:"isAllowExternalRegistrations"    gorm:"column:is_allow_external_registrations"`
	// means that any user with role MEMBER can invite other users
	IsAllowMemberInvitations bool `json:"isAllowMemberInvitations"        gorm:"column:is_allow_member_invitations"`
	// means that any user with role MEMBER can create their own projects
	IsMemberAllowedToCreateProjects bool `json:"isMemberAllowedToCreateProjects" gorm:"column:is_member_allowed_to_create_projects"`
}

func (UsersSettings) TableName() string {
	return "users_settings"
}
