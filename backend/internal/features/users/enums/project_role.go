package users_enums

type ProjectRole string

const (
	ProjectRoleOwner  ProjectRole = "OWNER"
	ProjectRoleAdmin  ProjectRole = "PROJECT_ADMIN"
	ProjectRoleMember ProjectRole = "PROJECT_MEMBER"
)

// IsValid validates the ProjectRole
func (r ProjectRole) IsValid() bool {
	switch r {
	case ProjectRoleOwner, ProjectRoleAdmin, ProjectRoleMember:
		return true
	default:
		return false
	}
}
