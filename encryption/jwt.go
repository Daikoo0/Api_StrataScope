package encryption

import (
	"fmt"
	"os"
	"time"

	"github.com/ProyectoT/api/internal/models"
	"github.com/golang-jwt/jwt/v4"
)

type InviteClaims struct {
	RoomID string `json:"roomID"`
	Role   string `json:"role"`
	Pass   string `json:"pass"`
	jwt.RegisteredClaims
}

func SignedLoginToken(u *models.User) (string, error) { //JWT
	key := os.Getenv("KEYPWD")
	// Se genera el token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{ //HS256 validacion unidireccional
		"email": u.Email,
		"name":  u.Name,
		"exp":   jwt.TimeFunc().AddDate(0, 0, 7).Unix(), // Expira en 7 dias
	})

	return token.SignedString([]byte(key)) // Se firma el token con la llave
}

func ParseLoginJWT(value string) (jwt.MapClaims, error) {
	key := os.Getenv("KEYPWD")
	token, err := jwt.Parse(value, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})

	if err != nil {
		return nil, err
	}
	//fmt.Print(token)
	return token.Claims.(jwt.MapClaims), nil
}

func InviteToken(roomID string, role string, pass string) (string, error) {
	key := os.Getenv("KEYPWD")
	if key == "" {
		return "", fmt.Errorf("clave secreta no configurada")
	}
	if pass == "" {
		return "", fmt.Errorf("el campo pass es obligatorio")
	}

	claims := InviteClaims{
		RoomID: roomID,
		Role:   role,
		Pass:   pass,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)), // Expiración en 7 días
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(key))
	if err != nil {
		return "", fmt.Errorf("error al firmar el token: %w", err)
	}

	return tokenString, nil
}

func ParseInviteToken(tokenString string) (*InviteClaims, error) {
	key := os.Getenv("KEYPWD")
	if key == "" {
		return nil, fmt.Errorf("clave secreta no configurada")
	}

	token, err := jwt.ParseWithClaims(tokenString, &InviteClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		return nil, fmt.Errorf("error al parsear el token: %w", err)
	}

	claims, ok := token.Claims.(*InviteClaims)
	if !ok {
		return nil, fmt.Errorf("no se pudieron obtener los claims")
	}

	return claims, nil
}
