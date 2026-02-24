package models

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Telephone string `json:"telephone"`
	Email     string `json:"email"`
	Password  string `json:"password"` // Store hash only
	Role      string `json:"role"`
}
