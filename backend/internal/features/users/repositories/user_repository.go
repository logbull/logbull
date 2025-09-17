package users_repositories

import (
	"fmt"
	users_enums "logbull/internal/features/users/enums"
	users_models "logbull/internal/features/users/models"
	"logbull/internal/storage"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct{}

func (r *UserRepository) CreateUser(user *users_models.User) error {
	return storage.GetDb().Create(user).Error
}

func (r *UserRepository) GetUserByEmail(email string) (*users_models.User, error) {
	var user users_models.User

	if err := storage.GetDb().Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(userID uuid.UUID) (*users_models.User, error) {
	var user users_models.User

	if err := storage.GetDb().Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateUserPassword(userID uuid.UUID, hashedPassword string) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"hashed_password":        hashedPassword,
			"password_creation_time": time.Now().UTC(),
		}).Error
}

func (r *UserRepository) CreateInitialAdmin() error {
	admin, err := r.GetUserByEmail("admin")
	if err != nil {
		return fmt.Errorf("failed to get admin user: %w", err)
	}

	if admin != nil {
		return nil
	}

	admin = &users_models.User{
		ID:                   uuid.New(),
		Email:                "admin",
		HashedPassword:       nil,
		PasswordCreationTime: time.Now().UTC(),
		Role:                 users_enums.UserRoleAdmin,
		Status:               users_enums.UserStatusActive,
		CreatedAt:            time.Now().UTC(),
	}

	return storage.GetDb().Create(admin).Error
}

func (r *UserRepository) GetUsers(limit, offset int, beforeCreatedAt *time.Time) ([]*users_models.User, int64, error) {
	var users []*users_models.User
	var total int64

	countQuery := storage.GetDb().Model(&users_models.User{})
	if beforeCreatedAt != nil {
		countQuery = countQuery.Where("created_at < ?", *beforeCreatedAt)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := storage.GetDb().
		Limit(limit).
		Offset(offset).
		Order("created_at DESC")

	if beforeCreatedAt != nil {
		query = query.Where("created_at < ?", *beforeCreatedAt)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) UpdateUserStatus(userID uuid.UUID, status users_enums.UserStatus) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"status": status,
		}).Error
}

func (r *UserRepository) UpdateUserRole(userID uuid.UUID, role users_enums.UserRole) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"role": role,
		}).Error
}

func (r *UserRepository) RenameUserEmailForTests(oldEmail, newEmail string) error {
	result := storage.GetDb().Model(&users_models.User{}).
		Where("email = ?", oldEmail).
		Update("email", newEmail)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return nil
	}

	return nil
}
