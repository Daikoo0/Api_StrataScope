package entity

import "github.com/ProyectoT/api/internal/models"

type Room struct {
	Name    string
	Clients map[string]models.Role
	Data    string
}