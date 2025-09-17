package users_testing

import (
	users_repositories "logbull/internal/features/users/repositories"
)

func EnableMemberInvitations() {
	updateUsersSetting("is_allow_member_invitations", true)
}

func DisableMemberInvitations() {
	updateUsersSetting("is_allow_member_invitations", false)
}

func EnableExternalRegistrations() {
	updateUsersSetting("is_allow_external_registrations", true)
}

func DisableExternalRegistrations() {
	updateUsersSetting("is_allow_external_registrations", false)
}

func EnableMemberProjectCreation() {
	updateUsersSetting("is_member_allowed_to_create_projects", true)
}

func DisableMemberProjectCreation() {
	updateUsersSetting("is_member_allowed_to_create_projects", false)
}

func ResetSettingsToDefaults() {
	repository := &users_repositories.UsersSettingsRepository{}
	settings, err := repository.GetSettings()
	if err != nil {
		panic(err)
	}

	settings.IsAllowExternalRegistrations = true
	settings.IsAllowMemberInvitations = true
	settings.IsMemberAllowedToCreateProjects = true

	err = repository.UpdateSettings(settings)
	if err != nil {
		panic(err)
	}
}

func updateUsersSetting(column string, value bool) {
	repository := &users_repositories.UsersSettingsRepository{}
	settings, err := repository.GetSettings()
	if err != nil {
		panic(err)
	}

	switch column {
	case "is_allow_member_invitations":
		settings.IsAllowMemberInvitations = value
	case "is_allow_external_registrations":
		settings.IsAllowExternalRegistrations = value
	case "is_member_allowed_to_create_projects":
		settings.IsMemberAllowedToCreateProjects = value
	}

	err = repository.UpdateSettings(settings)
	if err != nil {
		panic(err)
	}
}
