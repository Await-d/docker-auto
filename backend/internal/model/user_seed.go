package model

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// CreateDefaultUsers creates default demo users for testing and development
func CreateDefaultUsers(db *gorm.DB) error {
	// Define demo users
	demoUsers := []struct {
		Username string
		Email    string
		Password string
		Role     UserRole
	}{
		{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "admin123",
			Role:     UserRoleAdmin,
		},
		{
			Username: "operator",
			Email:    "operator@example.com",
			Password: "operator123",
			Role:     UserRoleOperator,
		},
		{
			Username: "viewer",
			Email:    "viewer@example.com",
			Password: "viewer123",
			Role:     UserRoleViewer,
		},
	}

	// Create each user if they don't exist
	for _, userData := range demoUsers {
		var existing User
		if err := db.Where("username = ? OR email = ?", userData.Username, userData.Email).First(&existing).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Hash the password
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
				if err != nil {
					return err
				}

				// Create the user
				user := User{
					Username:           userData.Username,
					Email:              userData.Email,
					PasswordHash:       string(hashedPassword),
					Role:               userData.Role,
					IsActive:           true,
					EmailNotifications: true,
				}

				if err := db.Create(&user).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}