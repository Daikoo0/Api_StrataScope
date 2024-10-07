package api

import (
	"net/http"

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

	if err := c.Bind(&params); err != nil {
		return a.handleError(c, http.StatusBadRequest, "Invalid request")
	}

	if err := a.dataValidator.Struct(params); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, fieldError := range validationErrors {
				switch fieldError.Field() {
				case "Password":
					return a.handleError(c, http.StatusBadRequest, "Password must be at least 8 characters long") // HTTP 400 Bad Request

				default:
					return a.handleError(c, http.StatusBadRequest, "Required field missing: "+fieldError.Field())
				}
			}
		}
		return a.handleError(c, http.StatusBadRequest, err.Error())
	}

	if params.Password != params.PasswordConfirm {
		return a.handleError(c, http.StatusBadRequest, "Password does not match")
	}

	if err := a.serv.RegisterUser(ctx, params.Email, params.Name, params.LastName, params.Password); err != nil {
		if err == service.ErrUserAlreadyExists {
			return a.handleError(c, http.StatusBadRequest, "Email already exists") // HTTP 400 Bad Request
		}
		return a.handleError(c, http.StatusInternalServerError, "Internal server error")
	}

	return c.JSON(http.StatusCreated, nil) // HTTP 201 Created
}

// LoginUser recibe un email y una contraseña, y devuelve un token de autenticación enviado en una cookie
func (a *API) LoginUser(c echo.Context) error {
	ctx := c.Request().Context()
	params := dtos.LoginUser{}

	if err := c.Bind(&params); err != nil {
		return a.handleError(c, http.StatusBadRequest, "Invalid request")
	}

	if err := a.dataValidator.Struct(params); err != nil {
		return a.handleError(c, http.StatusBadRequest, err.Error())
	}

	u, err := a.serv.LoginUser(ctx, params.Email, params.Password) // OBJID, email, name
	if err != nil {
		if err == service.ErrInvalidCredentials {
			return a.handleError(c, http.StatusUnauthorized, "User or password incorrect")
		}

		return a.handleError(c, http.StatusInternalServerError, "Internal server error")
	}

	token, err := encryption.SignedLoginToken(u) // Genera el token
	if err != nil {
		return a.handleError(c, http.StatusInternalServerError, "Internal server error")
	}

	return c.JSON(http.StatusOK, map[string]string{"token": token}) // HTTP 200 OK
}
