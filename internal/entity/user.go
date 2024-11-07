package entity

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email       string             `bson:"email" json:"email"`
	Name        string             `bson:"name" json:"name"`
	LastName    string             `bson:"lastName" json:"lastName"`
	Age         int                `bson:"age" json:"age"`
	Gender      string             `bson:"gender" json:"gender"`
	Nationality string             `bson:"nationality" json:"nationality"`
	Password    string             `bson:"password" json:"password"`
}

type Password struct {
	Password string `bson:"password" json:"password"`
	NewPassword string `bson:"password" json:"newPassword"`
	NewPwConfirm string `bson:"password" json:"newPwConfirm"`
}
