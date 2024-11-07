package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ProyectoT/api/encryption"
	"github.com/ProyectoT/api/internal/api/dtos"
	"github.com/ProyectoT/api/internal/entity"
	"github.com/ProyectoT/api/internal/models"
	"github.com/labstack/echo/v4"
)

func (a *API) AddComment(c echo.Context) error {
	ctx := c.Request().Context()
	params := models.Comment{}

	if err := c.Bind(&params); err != nil {
		return a.handleError(c, http.StatusBadRequest, "Invalid request")
	}

	if err := a.repo.HandleAddComment(ctx, params); err != nil {
		log.Println(err)
		return a.handleError(c, http.StatusInternalServerError, "Invalid Credentials")
	}

	return c.JSON(http.StatusOK, map[string]string{"success": "true"})
}

func (a *API) projects(c echo.Context) error {
	ctx, claims, err := a.getContextAndClaims(c)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, err.Error())
	}

	user := claims["email"].(string)

	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil || limit < 1 {
		limit = 5
	}

	proyects, currentPage, totalPages, err := a.repo.GetProyects(ctx, user, page, limit)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, "Error getting projects")
	}

	response := ProjectResponse{
		Projects:    proyects,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
	}
	return c.JSON(http.StatusOK, response)
}

func (a *API) HandleCreateProyect(c echo.Context) error {

	ctx, claims, err := a.getContextAndClaims(c)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, err.Error())
	}

	correo := claims["email"].(string)
	name := claims["name"].(string)

	var params dtos.Project

	if err := c.Bind(&params); err != nil {
		return a.handleError(c, http.StatusBadRequest, "Invalid request")
	}

	if err := a.dataValidator.Struct(params); err != nil {
		return a.handleError(c, http.StatusBadRequest, err.Error())
	}

	if err = a.serv.CreateRoom(ctx, params.RoomName, name, correo, params.Desc, params.Location, params.Lat, params.Long, params.Visible); err != nil {
		return a.handleError(c, http.StatusInternalServerError, "Failed to create a room")
	}

	return c.JSON(http.StatusOK, responseMessage{Message: "Room created successfully"})
}

func (a *API) DeleteProject(c echo.Context) error {
	ctx, claims, err := a.getContextAndClaims(c)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, err.Error())
	}

	user := claims["email"].(string)
	id := c.Param("id")

	roomInterface, exists := rooms.Load(id)
	if exists {
		existingRoom := roomInterface.(*RoomData)

		// Verificación de permisos del usuario
		if existingRoom.ProjectInfo.Members.Owner != user {
			// El usuario no es el dueño, solo se eliminará su acceso
			existingRoom.ProjectInfo.Members.Editors = removeUser(existingRoom.ProjectInfo.Members.Editors, user)
			existingRoom.ProjectInfo.Members.Readers = removeUser(existingRoom.ProjectInfo.Members.Readers, user)

			err = a.repo.UpdateMembers(ctx, id, existingRoom.ProjectInfo.Members)
			if err != nil {
				return a.handleError(c, http.StatusInternalServerError, "Failed to delete user from room")
			}
		} else {
			// El usuario es el dueño, se desconectan todos los usuarios y se elimina la sala
			existingRoom.DisconnectUsers()

			rooms.Delete(id)

			err = a.repo.DeleteProject(ctx, id)
			if err != nil {
				return a.handleError(c, http.StatusInternalServerError, "Failed to delete room")
			}

		}
	} else {
		// La sala no está en memoria, verifica directamente en la base de datos
		project, err := a.repo.GetMembers(ctx, id)
		if err != nil {
			return a.handleError(c, http.StatusNotFound, "Room not found")
		}

		if project.Owner != user {
			// Si el usuario no es el dueño, solo elimina su acceso en la base de datos
			if err = a.repo.DeleteUserRoom(ctx, user, id); err != nil {
				return a.handleError(c, http.StatusInternalServerError, "Failed to delete user from room")
			}
		} else {
			// Si el usuario es el dueño, elimina el proyecto de la base de datos
			if err = a.repo.DeleteProject(ctx, id); err != nil {
				return a.handleError(c, http.StatusInternalServerError, "Failed to delete room")
			}
		}
	}

	return c.JSON(http.StatusOK, responseMessage{Message: "Room deleted successfully"})
}

func (a *API) HandleGetPublicProject(c echo.Context) error {

	ctx := c.Request().Context()
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return a.handleError(c, http.StatusUnauthorized, "invalid or expired token")
	}

	proyects, err := a.repo.HandleGetPublicProject(ctx)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, "Error getting proyects")
	}

	response := ProjectResponse{Projects: proyects}

	return c.JSON(http.StatusOK, response)
}

func (a *API) HandleEditProfile(c echo.Context) error {
	ctx, claims, err := a.getContextAndClaims(c)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, err.Error())
	}

	email := claims["email"].(string)

	var req entity.User
	if err := c.Bind(&req); err != nil {
		return a.handleError(c, http.StatusBadRequest, "Invalid request")
	}

	if req.Name == "" || req.LastName == "" || req.Age == 0 || req.Gender == "" || req.Nationality == "" {
		return a.handleError(c, http.StatusBadRequest, "All fields are required")
	}

	if err = a.repo.UpdateUserProfile(ctx, req, email); err != nil {
		return a.handleError(c, http.StatusInternalServerError, "Failed to update profile")
	}

	return c.JSON(http.StatusOK, responseMessage{Message: "Profile updated successfully"})
}

func (a *API) HandleEditPassword(c echo.Context) error {
	ctx, claims, err := a.getContextAndClaims(c)
	if err != nil {
		return a.handleError(c, http.StatusUnauthorized, err.Error())
	}

	email := claims["email"].(string)

	var req entity.Password
	if err := c.Bind(&req); err != nil {
		return a.handleError(c, http.StatusBadRequest, "Invalid request")
	}

	if strings.TrimSpace(req.Password) == ""|| strings.TrimSpace(req.NewPassword) == "" || strings.TrimSpace(req.NewPwConfirm) == ""  {
		return a.handleError(c, http.StatusBadRequest, "All fields are required")
	}

	if err = a.repo.UpdatePassword(ctx, req, email); err != nil {
		return a.handleError(c, http.StatusInternalServerError, "Failed to update profile")
	}

	return c.JSON(http.StatusOK, responseMessage{Message: "Profile updated successfully"})
}



func removeUser(users []string, userToRemove string) []string {
	for i, u := range users {
		if u == userToRemove {
			return append(users[:i], users[i+1:]...)
		}
	}
	return users
}

func (a *API) getContextAndClaims(c echo.Context) (ctx context.Context, claims map[string]interface{}, err error) {
	ctx = c.Request().Context()
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		err = fmt.Errorf("invalid or expired token")
		return
	}

	claims, parseErr := encryption.ParseLoginJWT(auth)
	if parseErr != nil {
		err = fmt.Errorf("invalid or expired token")
		return
	}

	return
}

func (a *API) handleError(c echo.Context, statusCode int, message string) error {
	return c.JSON(statusCode, responseMessage{Message: message})
}
