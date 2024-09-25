package repository

import (
	"context"

	"github.com/ProyectoT/api/internal/api/dtos"
	"github.com/ProyectoT/api/internal/entity"
	"github.com/ProyectoT/api/internal/models"
	"go.mongodb.org/mongo-driver/mongo"
)

// mockery se configura con comentarios, mockery sirve para debugear el code
//
//go:generate mockery --name=Repository --output=repository --inpackage
type Repository interface { //comunicaciones con la base de datos

	// Room - room.repository.go
	GetRoom(ctx context.Context, roomName string) (*models.Project, error)
	GetRoomInfo(ctx context.Context, roomID string) (*models.ProjectInfo, error)                                                                              // Devuelve la entidad sala
	GetMembers(ctx context.Context, roomID string) (*models.Members, error)                                                                                   // Devuelve los miembros de una sala
	GetMembersAndPass(ctx context.Context, roomID string) (*models.Members, string, error)                                                                    // Devuelve los miembros y la contraseña de una sala
	CreateRoom(ctx context.Context, roomName string, name string, correo string, desc string, location string, lat float64, long float64, visible bool) error // Crea una sala                                                                                                       // Guarda un nuevo proyecto en la base de datos
	SaveRoom(ctx context.Context, data models.Project) error                                                                                                  // Actualiza un proyecto en MongoDB
	AddUserToProject(ctx context.Context, email string, role string, roomID string) error                                                                     // Guarda los usuarios en una sala en la base de datos
	UpdateMembers(ctx context.Context, roomID string, members models.Members) error                                                                           // Actualiza los miembros de una sala en la base de datos
	DeleteProject(ctx context.Context, roomID string) error                                                                                                   // Elimina un proyecto                                                                                                // Elimina un proyecto                                                                                                    // Elimina una sala de la base de datos

	// Profile - profile.repository.go
	GetProyects(ctx context.Context, email string) ([]models.InfoProject, error) // Devuelve los proyectos de un usuario
	//GetPermission(ctx context.Context, correo string, proyectID string) (int, error)
	HandleGetPublicProject(ctx context.Context) ([]models.InfoProject, error) // Devuelve los proyectos publicos

	// Users - user.repository.go
	SaveUser(ctx context.Context, email, name, lastname, password string) error // Guarda un usuario en la base de datos
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)     // Devuelve la entidad usuario (OBJID, email, name, password, proyects)
	DeleteUserRoom(ctx context.Context, email string, roomName string) error    // Elimina un usuario de una sala
	UpdateUserProfile(ctx context.Context, edit dtos.EditProfileRequest, email string) error

	// Comments - comments.repository.go
	HandleAddComment(ctx context.Context, comment models.Comment) error
}
type repo struct {
	db *mongo.Database
}

func New(db *mongo.Database) Repository {
	return &repo{
		db: db,
	}
}
