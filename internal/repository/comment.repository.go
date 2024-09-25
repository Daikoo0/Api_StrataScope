package repository

import (
	"context"

	"github.com/ProyectoT/api/internal/models"
)

func (r *repo) HandleAddComment(ctx context.Context, comment models.Comment) error {
	dbComments := r.db.Collection("comments")

	// Inserta el nuevo comentario en la base de datos
	_, err := dbComments.InsertOne(ctx, comment)
	if err != nil {
		// Manejar el error, por ejemplo, imprimirlo o devolverlo
		return err
	}

	return nil

}
