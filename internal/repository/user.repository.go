package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/ProyectoT/api/internal/api/dtos"
	"github.com/ProyectoT/api/internal/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/ProyectoT/api/encryption"
)

// Guarda un nuevo usuario
func (r *repo) SaveUser(ctx context.Context, email, name, lastname, password string) error {
	user := entity.User{
		Email:    email,
		Name:     name,
		LastName: lastname,
		Password: password,
		Proyects: []string{},
	}

	users := r.db.Collection("users")
	_, err := users.InsertOne(ctx, user)
	if err != nil {
		log.Println("Error saving user:", err)
		return err
	}

	return nil
}

// Obtiene un usuario por su email
func (r *repo) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	users := r.db.Collection("users")
	filter := bson.M{"email": email}

	result := users.FindOne(ctx, filter)
	if err := result.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err // User not found
		}
		log.Println("Error finding user:", err)
		return nil, err
	}

	var user entity.User
	err := result.Decode(&user)
	if err != nil {
		log.Println("Error decoding user:", err)
		return nil, err
	}

	return &user, nil
}

// Elimina un usuario de una sala
func (r *repo) DeleteUserRoom(ctx context.Context, email string, roomID string) error {

	projectID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	users := r.db.Collection("projects")
	filter := bson.M{"_id": projectID}
	update := bson.M{
		"$pull": bson.M{
			"projectinfo.members.editors": email,
			"projectinfo.members.readers": email,
		},
	}
	opts := options.Update().SetUpsert(true)
	result, err := users.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Println("Error updating room:", err)
		return err
	}
	if result.MatchedCount == 0 {
		log.Printf("User with email %s not found in room %s", email, roomID)
	} else {
		log.Printf("Successfully deleted user %s from room %s", email, roomID)
	}

	return nil
}



// Actualiza el perfil de un usuario
func (r *repo) UpdateUserProfile(ctx context.Context, edit dtos.EditProfileRequest, email string) error {
	
	u, err := r.GetUserByEmail(ctx, email)
	if u == nil {
		return err
	}

	bb, err := encryption.FromBase64(u.Password) // Transforma la contraseña de "et3L3evT" a [122 221 203 221 235]
	if err != nil {
		return err
	}
	decryptedPassword, err := encryption.Decrypt(bb) // Contraseña desencriptada
	if err != nil {
		return err
	}
	if string(decryptedPassword) != edit.Password { // verifica que la contraseña sea la misma
		return errors.New("invalid credentials") // si no es la misma retorna error de credenciales invalidas
	}

	var finalPassword string
	if edit.NewPassword != "" && edit.NewPassword == edit.NewPwConfirm { // Verifica que `newPassword` exista y coincida con `newPwConfirm`
	pw, _ := encryption.FromBase64(edit.NewPassword)
		finalPassword = string(pw) // Usa la nueva contraseña
	} else {
		finalPassword = string(bb) // Si no, mantiene la contraseña antigua
	}

	users := r.db.Collection("users")
	filter := bson.M{"email": email}
	update := bson.M{
		"$set": bson.M{
			"email":      email,
			"name":       edit.Name,
			"lastName":   edit.LastName,
			"password" :  finalPassword,
		},
	}
	opts := options.Update().SetUpsert(true)
	_, err = users.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Println("Error updating user profile:", err)
		return err
	}

	return nil

}
