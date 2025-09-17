package projects_controllers

import (
	projects_services "logbull/internal/features/projects/services"
)

var projectController = &ProjectController{
	projects_services.GetProjectService(),
}

var membershipController = &MembershipController{
	projects_services.GetMembershipService(),
}

func GetProjectController() *ProjectController {
	return projectController
}

func GetMembershipController() *MembershipController {
	return membershipController
}
