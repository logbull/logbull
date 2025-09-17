package users_controllers

import (
	users_services "logbull/internal/features/users/services"

	"golang.org/x/time/rate"
)

var userController = &UserController{
	userService:   users_services.GetUserService(),
	signinLimiter: rate.NewLimiter(rate.Limit(3), 3), // 3 RPS with burst of 3
}

var settingsController = &SettingsController{
	settingsService: users_services.GetSettingsService(),
}

var managementController = &ManagementController{
	managementService: users_services.GetManagementService(),
}

func GetUserController() *UserController {
	return userController
}

func GetSettingsController() *SettingsController {
	return settingsController
}

func GetManagementController() *ManagementController {
	return managementController
}
