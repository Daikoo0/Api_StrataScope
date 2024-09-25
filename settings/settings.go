package settings

import (
	"os"
)

type Settings struct {
	Port string `yaml:"port"`
	DB   string `yaml:"database"`
	Name string `yaml:"dbname"`
	Key  string `yaml:"key"`
}

func New() (*Settings, error) {
	var s Settings

	port := os.Getenv("PORT")
	db := os.Getenv("DATABASE")
	dbname := os.Getenv("DBNAME")
	key := os.Getenv("KEYPWD")

	s = Settings{
		Port: port,
		DB:   db,
		Name: dbname,
		Key:  key,
	}

	return &s, nil
}
