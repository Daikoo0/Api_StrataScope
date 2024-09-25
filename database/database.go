package database

import (
    "context"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "github.com/ProyectoT/api/settings"
)

func New(ctx context.Context, s *settings.Settings) (*mongo.Database, error) {
    clientOptions := options.Client().ApplyURI(s.DB)
    client, err := mongo.Connect(ctx, clientOptions)
    if err != nil {
        return nil, err
    }
    
    err = client.Ping(ctx, nil)
    if err != nil {
        return nil, err
    }
    DB := client.Database(s.Name)
    
    
    return DB, nil
}
