package entity

type Message struct {
	ID      string `bson:"_id,omitempty"`
	Room    string `bson:"room"`
	Message string `bson:"message"`
}