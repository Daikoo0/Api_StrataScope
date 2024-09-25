package entity

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserRole struct {
	ID          primitive.ObjectID 	   `bson:"_id,omitempty"`
	UserEmail   string			       `db:"user_email"`
	RoleID 		int64                  `db:"role_id"`
}