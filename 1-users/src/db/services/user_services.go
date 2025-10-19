package services

import (
	"errors"
	"time"

	"omni/src/db"
	"omni/src/models"
	"omni/src/utils"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) CreateUser(user *models.User) error {
	if _, exists := db.Users[user.Email]; exists {
		return errors.New("user already exists")
	}

	accountId, err := utils.GenerateAccountId()
	if err != nil {
		return err
	}
	user.AccountId = accountId

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.HashedPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.HashedPassword = string(hashedPassword)

	user.KYCStatus = models.KYCStatusPending

	db.Users[user.Email] = *user
	return nil
}

func (s *UserService) GetUser(email string) (*models.User, error) {
	user, exists := db.Users[email]
	if !exists {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (s *UserService) UpdateUser(email string, updates map[string]interface{}) error {
	user, exists := db.Users[email]
	if !exists {
		return errors.New("user not found")
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "FirstName":
			user.FirstName = value.(string)
		case "LastName":
			user.LastName = value.(string)
		case "OmniTag":
			user.OmniTag = value.(string)
		case "KYCStatus":
			user.KYCStatus = value.(models.KYCStatus)
			// Add other fields as needed
		}
	}

	db.Users[email] = user
	return nil
}

func (s *UserService) DeleteUser(email string) error {
	if _, exists := db.Users[email]; !exists {
		return errors.New("user not found")
	}
	delete(db.Users, email)
	return nil
}

func (s *UserService) CreateRefreshToken(email, deviceInfo string) (string, error) {
	refreshToken, err := utils.GenerateRefreshToken(email)
	if err != nil {
		return "", err
	}

	db.RefreshTokens[refreshToken] = db.RefreshTokenInfo{
		UserEmail:  email,
		DeviceInfo: deviceInfo,
		CreatedAt:  time.Now(),
	}

	return refreshToken, nil
}
