package service

import (
	"context"

	"github.com/ProyectoT/api/internal/models"
)

func (s *serv) GetRoom(ctx context.Context, roomName string) (*models.Project, error) {
	room, err := s.repo.GetRoom(ctx, roomName)
	if err != nil {
		return nil, err
	}
	return room, nil
}

// func (s *serv) GetRoomInfo(ctx context.Context, roomName string) (*models.ProjectInfo, error) {
// 	room, err := s.repo.GetRoomInfo(ctx, roomName)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return room, nil
// }

func (s *serv) CreateRoom(ctx context.Context, roomName string, name string, correo string, desc string, location string, lat float64, long float64, visible bool) error {

	return s.repo.CreateRoom(ctx, roomName, name, correo, desc, location, lat, long, visible)
}
