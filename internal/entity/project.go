package entity

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Name     string             `bson:"email"`
	Data     string             `bson:"name"`
}