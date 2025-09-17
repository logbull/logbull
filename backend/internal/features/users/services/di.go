package users_services

import (
	user_repositories "logbull/internal/features/users/repositories"
)

var secretKeyRepository = &user_repositories.SecretKeyRepository{}
var userRepository = &user_repositories.UserRepository{}
var usersSettingsRepository = &user_repositories.UsersSettingsRepository{}

var userService = &UserService{
	userRepository:      userRepository,
	secretKeyRepository: secretKeyRepository,
	settingsService:     settingsService,
}
var settingsService = &SettingsService{
	userSettingsRepository: usersSettingsRepository,
}
var managementService = &UserManagementService{
	userRepository: userRepository,
}

func GetUserService() *UserService {
	return userService
}

func GetSettingsService() *SettingsService {
	return settingsService
}

func GetManagementService() *UserManagementService {
	return managementService
}
