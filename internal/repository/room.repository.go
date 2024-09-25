package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ProyectoT/api/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Obtiene todo el contenido de una sala
func (r *repo) GetRoom(ctx context.Context, roomID string) (*models.Project, error) {

	rooms := r.db.Collection("projects")
	var room models.Project
	objectID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		log.Fatal(err)
	}

	err = rooms.FindOne(ctx, bson.M{"_id": objectID}).Decode(&room)
	if err == mongo.ErrNoDocuments {
		return nil, err
	}

	return &room, nil
}

// Obtiene la info de una sala en específico
func (r *repo) GetRoomInfo(ctx context.Context, roomID string) (*models.ProjectInfo, error) {

	rooms := r.db.Collection("projects")
	var projectInfo models.ProjectInfo
	objectID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	projection := bson.M{
		"ProjectInfo": 1,
	}

	err = rooms.FindOne(ctx, bson.M{"_id": objectID}, options.FindOne().SetProjection(projection)).Decode(&projectInfo)
	if err != nil {
		return nil, err
	}

	return &projectInfo, nil
}

// Crea una nueva sala
func (r *repo) CreateRoom(ctx context.Context, roomName string, name string, correo string, desc string, location string, lat float64, long float64, visible bool) error {
	rooms := r.db.Collection("projects")
	anio, mes, dia := time.Now().Date()
	creationDate := fmt.Sprintf("%d-%02d-%02d", anio, mes, dia)

	room := &models.Project{
		ProjectInfo: models.ProjectInfo{
			Name:  roomName,
			Owner: name,
			Members: models.Members{
				Owner:   correo,
				Editors: []string{},
				Readers: []string{},
			},
			CreationDate: creationDate,
			Description:  desc,
			Location:     location,
			Lat:          lat,
			Long:         long,
			Visible:      visible,
		},
		Data:   []models.DataInfo{},
		Fosil:  map[string]models.Fosil{},
		Facies: map[string][]models.FaciesSection{},
		Config: models.Config{
			Columns: map[string]bool{
				"Edad":                   true,
				"Espesor":                true,
				"Facie":                  true,
				"Sistema":                true,
				"Formacion":              true,
				"Miembro":                true,
				"Litologia":              true,
				"Estructura fosil":       true,
				"AmbienteDepositacional": true,
				"Descripcion":            true,
			},
			IsInverted: false,
		},
	}

	count, err := rooms.CountDocuments(ctx, bson.M{"name": roomName, "members.0": correo})
	if err != nil {
		log.Println("Error checking project existence:", err)
		return err
	}

	if count > 0 {
		return errors.New("the user already has a project with this name")
	}

	_, err = rooms.InsertOne(ctx, room)
	if err != nil {
		log.Println("Error creating room:", err)
		return err
	}

	return nil
}

// Elimina una sala
func (r *repo) DeleteProject(ctx context.Context, roomID string) error {
	dbProject := r.db.Collection("projects")

	projectID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	res, err := dbProject.DeleteOne(ctx, bson.M{"_id": projectID})
	if err != nil {
		log.Println("Error deleting room:", err)
		return err
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("project not found in projects collection")
	}

	return nil
}

// Guarda una sala
func (r *repo) SaveRoom(ctx context.Context, data models.Project) error {
	rooms := r.db.Collection("projects")

	filter := bson.M{"_id": data.ID}
	update := bson.M{"$set": bson.M{
		"projectinfo": data.ProjectInfo,
		"data":        data.Data,
		"config":      data.Config,
		"fosil":       data.Fosil,
		"facies":      data.Facies,
		"shared":      data.Shared,
	}}

	opts := options.Update().SetUpsert(true)

	_, err := rooms.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Println("Error updating room:", err)
		return err
	}

	return nil
}

// NO SE USA
func (r *repo) AddUserToProject(ctx context.Context, email string, role string, roomID string) error {
	// Obtener la colección "projects" de la base de datos
	users := r.db.Collection("projects")
	projectID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}

	// Filtrar por id_project
	filter := bson.M{"_id": projectID}

	// Definir una actualización para agregar el correo electrónico a la lista correspondiente
	update := bson.M{
		"$push": bson.M{
			"projectinfo.members." + role: email,
		},
	}

	// Opciones de actualización
	opts := options.Update().SetUpsert(false)

	// Realizar la actualización en la base de datos
	result, err := users.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Println("Error updating room:", err)
		return err
	}

	if result.MatchedCount == 0 {
		log.Printf("Room with id %s not found", roomID)
	} else {
		log.Printf("Successfully added user %s to role %s in room %s", email, role, roomID)
	}

	return nil
}

func (r *repo) UpdateMembers(ctx context.Context, roomID string, members models.Members) error {
	rooms := r.db.Collection("projects")
	objectID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		log.Fatal(err)
	}

	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{
		"projectinfo.members": members,
	}}

	opts := options.Update().SetUpsert(true)

	_, err = rooms.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Println("Error updating room:", err)
		return err
	}

	return nil
}

// Obtiene los miembros de una sala
func (r *repo) GetMembers(ctx context.Context, roomID string) (*models.Members, error) {
	rooms := r.db.Collection("projects")
	var result struct {
		ProjectInfo struct {
			Members models.Members `bson:"members"`
		} `bson:"projectinfo"`
	}

	objectID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		log.Fatal(err)
	}

	projection := bson.M{
		"projectinfo.members": 1,
	}

	err = rooms.FindOne(ctx, bson.M{"_id": objectID}, options.FindOne().SetProjection(projection)).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result.ProjectInfo.Members, nil
}

func (r *repo) GetMembersAndPass(ctx context.Context, roomID string) (*models.Members, string, error) {

	rooms := r.db.Collection("projects")
	var result struct {
		ProjectInfo struct {
			Members models.Members `bson:"members"`
		} `bson:"projectinfo"`
		Shared struct {
			Pass string `bson:"pass"`
		} `bson:"shared"`
	}

	objectID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		log.Fatal(err)
	}

	projection := bson.M{
		"projectinfo.members": 1,
		"shared.pass":         1,
	}

	err = rooms.FindOne(ctx, bson.M{"_id": objectID}, options.FindOne().SetProjection(projection)).Decode(&result)
	if err != nil {
		log.Print("aaaaa")
		return nil, "", err
	}

	return &result.ProjectInfo.Members, result.Shared.Pass, nil
}
