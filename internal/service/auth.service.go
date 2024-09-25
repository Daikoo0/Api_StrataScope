package service

import (
	"context"
	"errors"

	"github.com/ProyectoT/api/encryption"
	"github.com/ProyectoT/api/internal/models"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRoleAlreadyAdded   = errors.New("role was already added for this user")
	ErrRoleNotFound       = errors.New("role not found")
)

func (s *serv) RegisterUser(ctx context.Context, email, name, lastname, password string) error {

	u, _ := s.repo.GetUserByEmail(ctx, email) // leer usuario de la bd completo OBJID, email, name, password, proyects
	if u != nil {
		return ErrUserAlreadyExists // Error de usuario ya existe
	}

	bb, err := encryption.Encrypt([]byte(password))
	if err != nil {
		return err
	}

	pass := encryption.ToBase64(bb)                //cifrar contraseña
	return s.repo.SaveUser(ctx, email, name, lastname, pass) // Guardar usuario en la base de datos
}

func (s *serv) LoginUser(ctx context.Context, email, password string) (*models.User, error) {
	u, err := s.repo.GetUserByEmail(ctx, email) // leer usuario de la bd completo OBJID, email, name, password, proyects
	if err != nil {
		return nil, err
	}

	bb, err := encryption.FromBase64(u.Password) // Transforma la contraseña de "et3L3evT" a [122 221 203 221 235]
	if err != nil {
		return nil, err
	}
	decryptedPassword, err := encryption.Decrypt(bb) // Contraseña desencriptada
	if err != nil {
		return nil, err
	}
	if string(decryptedPassword) != password { // verifica que la contraseña sea la misma
		return nil, ErrInvalidCredentials // si no es la misma retorna error de credenciales invalidas
	}

	return &models.User{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
	}, nil
}
