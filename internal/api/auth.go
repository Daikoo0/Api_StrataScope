package api

import (
	"log"
	"net/http"
	"time"

	"github.com/ProyectoT/api/encryption"
	"github.com/ProyectoT/api/internal/api/dtos"
	"github.com/ProyectoT/api/internal/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// registerUser recibe un email, un nombre y una contraseña, y registra un usuario en la base de datos
func (a *API) RegisterUser(c echo.Context) error {
	ctx := c.Request().Context()
	params := dtos.RegisterUser{}

	err := c.Bind(&params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: "Invalid request"}) // HTTP 400 Bad Request
	}

	err = a.dataValidator.Struct(params)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, fieldError := range validationErrors {
				switch fieldError.Field() {
				case "Password":
					return c.JSON(http.StatusBadRequest, responseMessage{Message: "Password must be at least 8 characters long"}) // HTTP 400 Bad Request

				default:
					return c.JSON(http.StatusBadRequest, responseMessage{Message: "required field missing: " + fieldError.Field()}) // HTTP 400 Bad Request
				}
			}
		}
		return c.JSON(http.StatusBadRequest, responseMessage{Message: err.Error()})
	}

	if params.Password != params.PasswordConfirm {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: "Password does not match"}) // HTTP 400 Bad Request
	}

	err = a.serv.RegisterUser(ctx, params.Email, params.Name, params.LastName, params.Password)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			return c.JSON(http.StatusBadRequest, responseMessage{Message: "Email already exists"}) // HTTP 409 Conflict
		}

		return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Internal server error"}) // HTTP 500 Internal Server Error
	}

	return c.JSON(http.StatusCreated, nil) // HTTP 201 Created
}

// LoginUser recibe un email y una contraseña, y devuelve un token de autenticación enviado en una cookie
func (a *API) LoginUser(c echo.Context) error {
	ctx := c.Request().Context()
	params := dtos.LoginUser{}

	err := c.Bind(&params) // llena a params con los datos de la solicitud

	if err != nil {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: "Invalid request"})
	}

	err = a.dataValidator.Struct(params) // valida los datos de la solicitud

	if err != nil {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: err.Error()}) // HTTP 400 Bad Request
	}

	u, err := a.serv.LoginUser(ctx, params.Email, params.Password) // OBJID, email, name
	if err != nil {
		if err == service.ErrInvalidCredentials {
			return c.JSON(http.StatusUnauthorized, responseMessage{Message: "user or password incorrect"}) // HTTP 401
		}

		return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Internal Server Error"}) // HTTP 500

	}

	token, err := encryption.SignedLoginToken(u) // Genera el token
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Internal Server Error"}) // HTTP 500
	}

	return c.JSON(http.StatusOK, map[string]string{"token": token}) // HTTP 200 OK
}

// Verifica si existe una cookie de autenticación
func (a *API) AuthUser(c echo.Context) error {

	_, err := c.Cookie("Authorization")
	if err != nil {
		log.Print("No cookie")
		return c.NoContent(http.StatusUnauthorized) // HTTP 401 Unauthorized
	}
	log.Print("Cookie found")
	return c.NoContent(http.StatusOK) // HTTP 200 OK
}

// LogoutUser elimina la cookie de autenticación
func (a *API) LogoutUser(c echo.Context) error {

	expiredCookie := &http.Cookie{
		Name:     "Authorization",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
		HttpOnly: true,
		Path:     "/",
	}

	c.SetCookie(expiredCookie) // Setea la cookie en el navegador

	return c.JSON(http.StatusOK, map[string]string{"success": "true"}) // HTTP 200 OK
}
