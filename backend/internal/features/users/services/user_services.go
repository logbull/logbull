package users_services

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_interfaces "logbull/internal/features/users/interfaces"
	users_models "logbull/internal/features/users/models"
	users_repositories "logbull/internal/features/users/repositories"
)

type UserService struct {
	userRepository      *users_repositories.UserRepository
	secretKeyRepository *users_repositories.SecretKeyRepository
	settingsService     *SettingsService
	// audit log is never nil, DI always set it
	auditLogWriter users_interfaces.AuditLogWriter
}

func NewUserService(
	userRepository *users_repositories.UserRepository,
	secretKeyRepository *users_repositories.SecretKeyRepository,
	settingsService *SettingsService,
) *UserService {
	return &UserService{
		userRepository:      userRepository,
		secretKeyRepository: secretKeyRepository,
		settingsService:     settingsService,
	}
}

func (s *UserService) SetAuditLogWriter(writer users_interfaces.AuditLogWriter) {
	s.auditLogWriter = writer
}

func (s *UserService) SignUp(request *users_dto.SignUpRequestDTO) error {
	existingUser, err := s.userRepository.GetUserByEmail(request.Email)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if existingUser != nil && existingUser.Status != users_enums.UserStatusInvited {
		return errors.New("user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	hashedPasswordStr := string(hashedPassword)

	// If user exists with INVITED status, activate them and set password
	if existingUser != nil && existingUser.Status == users_enums.UserStatusInvited {
		if err := s.userRepository.UpdateUserPassword(existingUser.ID, hashedPasswordStr); err != nil {
			return fmt.Errorf("failed to set password: %w", err)
		}

		if err := s.userRepository.UpdateUserStatus(existingUser.ID, users_enums.UserStatusActive); err != nil {
			return fmt.Errorf("failed to activate user: %w", err)
		}

		s.auditLogWriter.WriteAuditLog(
			fmt.Sprintf("Invited user completed registration: %s", existingUser.Email),
			&existingUser.ID,
			nil,
		)

		return nil
	}

	// Get settings to check registration policy for new users
	settings, err := s.settingsService.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// Check if external registrations are allowed
	if !settings.IsAllowExternalRegistrations {
		return errors.New("external registration is disabled")
	}

	// Create new user
	user := &users_models.User{
		ID:                   uuid.New(),
		Email:                request.Email,
		HashedPassword:       &hashedPasswordStr,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 users_enums.UserRoleMember,
		Status:               users_enums.UserStatusActive,
		CreatedAt:            time.Now().UTC(),
	}

	if err := s.userRepository.CreateUser(user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	s.auditLogWriter.WriteAuditLog(
		fmt.Sprintf("User registered with email: %s", user.Email),
		&user.ID,
		nil,
	)

	return nil
}

func (s *UserService) SignIn(request *users_dto.SignInRequestDTO) (*users_dto.SignInResponseDTO, error) {
	user, err := s.userRepository.GetUserByEmail(request.Email)
	if err != nil {
		return nil, errors.New("user with this email does not exist")
	}

	if user == nil {
		return nil, errors.New("user with this email does not exist")
	}

	if user.Status == users_enums.UserStatusInvited {
		return nil, errors.New("user account is not passed sign up yet")
	}

	if user.Status != users_enums.UserStatusActive {
		return nil, errors.New("user account is deactivated")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.HashedPassword), []byte(request.Password))
	if err != nil {
		return nil, errors.New("password is incorrect")
	}

	response, err := s.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	s.auditLogWriter.WriteAuditLog(
		fmt.Sprintf("User signed in with email: %s", user.Email),
		&user.ID,
		nil,
	)

	return response, nil
}

func (s *UserService) GetUserFromToken(token string) (*users_models.User, error) {
	secretKey, err := s.secretKeyRepository.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok && parsedToken.Valid {
		userIDStr, ok := claims["sub"].(string)
		if !ok {
			return nil, errors.New("invalid token claims")
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, errors.New("invalid token claims")
		}

		user, err := s.userRepository.GetUserByID(userID)
		if err != nil {
			return nil, err
		}

		// Check if user is active
		if !user.IsActiveUser() {
			return nil, errors.New("user account is deactivated")
		}

		if passwordCreationTimeUnix, ok := claims["passwordCreationTime"].(float64); ok {
			tokenPasswordTime := time.Unix(int64(passwordCreationTimeUnix), 0)

			tokenTimeSeconds := tokenPasswordTime.Truncate(time.Second)
			userTimeSeconds := user.PasswordCreationTime.Truncate(time.Second)

			if !tokenTimeSeconds.Equal(userTimeSeconds) {
				return nil, errors.New("password has been changed, please sign in again")
			}
		} else {
			return nil, errors.New("invalid token claims: missing password creation time")
		}

		return user, nil
	}

	return nil, errors.New("invalid token")
}

func (s *UserService) GenerateAccessToken(user *users_models.User) (*users_dto.SignInResponseDTO, error) {
	secretKey, err := s.secretKeyRepository.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get secret key: %w", err)
	}

	tenYearsExpiration := time.Now().UTC().Add(time.Hour * 24 * 365 * 10)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                  user.ID.String(),
		"exp":                  tenYearsExpiration.Unix(),
		"iat":                  time.Now().UTC().Unix(),
		"role":                 string(user.Role),
		"passwordCreationTime": user.PasswordCreationTime.Unix(),
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &users_dto.SignInResponseDTO{
		UserID: user.ID,
		Token:  tokenString,
	}, nil
}

func (s *UserService) CreateInitialAdmin() error {
	return s.userRepository.CreateInitialAdmin()
}

func (s *UserService) IsRootAdminHasPassword() (bool, error) {
	admin, err := s.userRepository.GetUserByEmail("admin")

	if err != nil {
		return false, fmt.Errorf("failed to get admin user: %w", err)
	}

	if admin == nil {
		return false, errors.New("admin user does not exist")
	}

	return admin.HasPassword(), nil
}

func (s *UserService) SetRootAdminPassword(password string) error {
	admin, err := s.userRepository.GetUserByEmail("admin")
	if err != nil {
		return fmt.Errorf("failed to get admin user: %w", err)
	}

	if admin == nil {
		return errors.New("admin user does not exist")
	}

	if admin.HasPassword() {
		return errors.New("admin password is already set")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepository.UpdateUserPassword(admin.ID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to set admin password: %w", err)
	}

	if s.auditLogWriter != nil {
		s.auditLogWriter.WriteAuditLog(
			"Admin password set",
			&admin.ID,
			nil,
		)
	}

	return nil
}

func (s *UserService) ChangeUserPasswordByEmail(email string, newPassword string) error {
	user, err := s.userRepository.GetUserByEmail(email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	return s.ChangeUserPassword(user.ID, newPassword)
}

func (s *UserService) ChangeUserPassword(userID uuid.UUID, newPassword string) error {
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if !user.HasPassword() {
		return errors.New("user has no password set")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	if err := s.userRepository.UpdateUserPassword(userID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.auditLogWriter.WriteAuditLog(
		"Password changed",
		&userID,
		nil,
	)

	return nil
}

func (s *UserService) InviteUser(
	request *users_dto.InviteUserRequestDTO,
	invitedBy *users_models.User,
) (*users_dto.InviteUserResponseDTO, error) {
	// Get settings to check permissions
	settings, err := s.settingsService.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Check if user has permission to invite
	if !invitedBy.CanInviteUsers(settings) {
		return nil, errors.New("insufficient permissions to invite users")
	}

	// Check if user already exists
	existingUser, err := s.userRepository.GetUserByEmail(request.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	user := &users_models.User{
		ID:                   uuid.New(),
		Email:                request.Email,
		HashedPassword:       nil, // No password yet
		PasswordCreationTime: time.Now().UTC(),
		Role:                 users_enums.UserRoleMember,
		Status:               users_enums.UserStatusInvited,
		CreatedAt:            time.Now().UTC(),
	}

	if err := s.userRepository.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create invited user: %w", err)
	}

	message := fmt.Sprintf("User invited: %s", request.Email)
	if request.IntendedProjectID != nil {
		message += fmt.Sprintf(" for project %s", request.IntendedProjectID.String())
	}
	s.auditLogWriter.WriteAuditLog(
		message,
		&invitedBy.ID,
		request.IntendedProjectID,
	)

	return &users_dto.InviteUserResponseDTO{
		ID:                  user.ID,
		Email:               user.Email,
		IntendedProjectID:   request.IntendedProjectID,
		IntendedProjectRole: request.IntendedProjectRole,
		CreatedAt:           user.CreatedAt,
	}, nil
}

func (s *UserService) GetUserByID(userID uuid.UUID) (*users_models.User, error) {
	return s.userRepository.GetUserByID(userID)
}

func (s *UserService) GetUserByEmail(email string) (*users_models.User, error) {
	return s.userRepository.GetUserByEmail(email)
}

func (s *UserService) GetCurrentUserProfile(user *users_models.User) *users_dto.UserProfileResponseDTO {
	return &users_dto.UserProfileResponseDTO{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		IsActive:  user.IsActiveUser(),
		CreatedAt: user.CreatedAt,
	}
}
