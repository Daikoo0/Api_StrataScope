package entity

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Email    string             `bson:"email"`
	Name     string             `bson:"name"`
	LastName string             `bson:"lastName"`
	Password string             `bson:"password"`
	Proyects []string           `bson:"proyects"`
}