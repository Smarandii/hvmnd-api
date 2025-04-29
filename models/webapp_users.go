package models

type RegistrationInput struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type PasswordResetInput struct {
	ResetToken      string `json:"reset_token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WebAppUser struct {
	ID                     int     `json:"id"`
	Email                  string  `json:"email"`
	Balance                float64 `json:"balance"`
	TotalSpent             float64 `json:"total_spent"`
	Banned                 bool    `json:"banned"`
	EmailConfirmed         bool    `json:"is_email_confirmed"`
	EmailConfirmationToken string  `json:"email_confirmation_token"`
}
