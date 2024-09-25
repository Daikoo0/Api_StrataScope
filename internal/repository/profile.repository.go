package repository

import (
	"context"

	"github.com/ProyectoT/api/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Obtiene la info de todos los proyectos de un usuario
func (r *repo) GetProyects(ctx context.Context, correo string) ([]models.InfoProject, error) {
	users := r.db.Collection("projects")

	filter := bson.M{
		"$or": []bson.M{
			{"projectinfo.members.owner": correo},
			{"projectinfo.members.editors": correo},
			{"projectinfo.members.readers": correo},
		},
	}

	projection := bson.M{
		"projectinfo": 1,
	}

	cursor, err := users.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []models.InfoProject

	if err := cursor.All(ctx, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// Obtiene la info de todos los proyectos publicos
func (r *repo) HandleGetPublicProject(ctx context.Context) ([]models.InfoProject, error) {

	db := r.db.Collection("projects")

	filter := bson.M{
		"projectinfo.visible": true,
	}

	projection := bson.M{
		"projectinfo": 1,
	}

	cursor, err := db.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []models.InfoProject

	if err := cursor.All(ctx, &projects); err != nil {
		return nil, err
	}

	return projects, nil

}
