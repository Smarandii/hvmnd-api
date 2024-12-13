package models

type RegistrationInput struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type WebAppUser struct {
	ID         int     `json:"id"`
	Email      string  `json:"email"`
	Balance    float64 `json:"balance"`
	TotalSpent float64 `json:"total_spent"`
	Banned     bool    `json:"banned"`
}
