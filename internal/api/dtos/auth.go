package dtos

import "github.com/ProyectoT/api/internal/models"

type LoginUser struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RegisterUser struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8"`
	PasswordConfirm string `json:"passwordConfirm" validate:"required,min=8"`
	Name            string `json:"name" validate:"required"`
	LastName        string `json:"lastName" validate:"required"`
}

type InviteUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Participant struct {
	Email string
	Role  models.Role
}

type CreateProjectRequest struct {
	Participants []Participant `json:"participants"`
}
