package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ProyectoT/api/encryption"
	"github.com/ProyectoT/api/internal/api/dtos"
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

	existingRoom, exists := rooms[id]
	if exists {
		if existingRoom.ProjectInfo.Members.Owner != user {

			existingRoom.ProjectInfo.Members.Editors = removeUser(existingRoom.ProjectInfo.Members.Editors, user)
			existingRoom.ProjectInfo.Members.Readers = removeUser(existingRoom.ProjectInfo.Members.Readers, user)

			err = a.repo.UpdateMembers(ctx, id, existingRoom.ProjectInfo.Members)
			if err != nil {
				return a.handleError(c, http.StatusInternalServerError, "Failed to delete user from room")
			}

		} else {
			existingRoom.DisconnectUsers()

			err = a.repo.DeleteProject(ctx, id)
			if err != nil {
				return a.handleError(c, http.StatusInternalServerError, "Failed to delete room")
			}

			delete(rooms, id)
		}
	} else {
		proyect, err := a.repo.GetMembers(ctx, id)
		if err != nil {
			return a.handleError(c, http.StatusNotFound, "Room not found")
		}

		if proyect.Owner != user {
			if err = a.repo.DeleteUserRoom(ctx, user, id); err != nil {
				return a.handleError(c, http.StatusInternalServerError, "Failed to delete user from room")
			}
		} else {
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

	var req dtos.EditProfileRequest
	if err := c.Bind(&req); err != nil {
		return a.handleError(c, http.StatusBadRequest, "Invalid request")
	}

	if err := a.dataValidator.Struct(req); err != nil {
		return a.handleError(c, http.StatusBadRequest, err.Error())
	}

	if err = a.repo.UpdateUserProfile(ctx, req, email); err != nil {
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
