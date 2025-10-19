package models

// Define a struct to unmarshal the password change request
type PasswordChangeRequest struct {
	AccountId          string `json:"account-id"`
	Email              string `json:"email"`
	CurrentPassword    string `json:"current-password"`
	NewPassword        string `json:"new-password"`
	ConfirmNewPassword string `json:"confirm-new-password"`
}
