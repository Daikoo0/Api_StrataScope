package service

import (
	"context"

	"github.com/ProyectoT/api/internal/models"
	"github.com/ProyectoT/api/internal/repository"
)

// Service is the business logic of the application.
//
//go:generate mockery --name=Service --output=service --inpackage
type Service interface {

	// --- Todas se comunican con repository --- //

	// Room - room.service.go
	GetRoom(ctx context.Context, roomName string) (*models.Project, error)
	CreateRoom(ctx context.Context, roomName string, name string, correo string, desc string, location string, lat float64, long float64, visible bool) error

	// Profile - profile.service.go
	GetProyects(ctx context.Context, user string) ([]models.InfoProject, error)

	// Auth - auth.service.go
	RegisterUser(ctx context.Context, email, name, lastname, password string) error
	LoginUser(ctx context.Context, email, password string) (*models.User, error)
}

type serv struct {
	repo repository.Repository
}

func New(repo repository.Repository) Service {
	return &serv{
		repo: repo,
	}
}
