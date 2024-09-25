package models

type Comment struct {
	Content   string   `bson:"content"`
	CreatedAt string   `bson:"createdAt"`
	Labels    []string `bson:"labels"`
}
