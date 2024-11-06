package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Role int

const (
	Owner Role = iota
	Editor
	Reader
)

type User struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Email string             `json:"email"`
	Name  string             `json:"name"`
}

type UserRole struct {
	Email string `json:"email"`
	Role  int    `json:"role"`
}

type InviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type EditProfileRequest struct {
	Name  string `json:"first_name" validate:"required"`
	LastName   string `json:"last_name" validate:"required"`
	Password string `bson:"password"`
	NewPassword string `bson:"newPassword"validate:"required,min=8"`
	NewPwConfirm string `bson:"newPwConfirm"validate:"required"`
}

