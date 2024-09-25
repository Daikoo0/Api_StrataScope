package api

import (
	"log"
	"net/http"

	"github.com/ProyectoT/api/encryption"
	"github.com/ProyectoT/api/internal/api/dtos"
	"github.com/ProyectoT/api/internal/models"
	"github.com/labstack/echo/v4"
)

func (a *API) AddComment(c echo.Context) error {
	ctx := c.Request().Context() // Context.Context es una interfaz que permite el paso de valores entre funciones
	params := models.Comment{}

	err := c.Bind(&params) // llena a params con los datos de la solicitud

	// Sin error  == nil - Con error != nil
	if err != nil {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: "Invalid request"})
	}

	err = a.repo.HandleAddComment(ctx, params)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Invalid Credentials"})
	}

	return c.JSON(http.StatusOK, map[string]string{"success": "true"}) // HTTP 200 OK
}

func (a *API) projects(c echo.Context) error {

	ctx := c.Request().Context()
	auth := c.Request().Header.Get("Authorization")

	//validar datos
	if auth == "" {
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	claims, err := encryption.ParseLoginJWT(auth)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: err.Error()})
	}

	user := claims["email"].(string)

	proyects, err := a.serv.GetProyects(ctx, user)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Error getting proyects"})
	}

	response := ProjectResponse{
		Projects: proyects,
	}

	// Devolver la respuesta JSON con los proyectos
	return c.JSON(http.StatusOK, response)
}

func (a *API) HandleCreateProyect(c echo.Context) error {

	ctx := c.Request().Context()

	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	// Si existe, revisa si es valido
	claims, err := encryption.ParseLoginJWT(auth)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	correo := claims["email"].(string)
	name := claims["name"].(string)
	log.Println(correo)
	log.Println(name)

	var params dtos.Project

	err = c.Bind(&params) // llena a params con los datos de la solicitud

	log.Print(params)

	if err != nil {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: "Invalid request"})
	}

	err = a.dataValidator.Struct(params) // valida los datos de la solicitud
	if err != nil {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: err.Error()}) // HTTP 400 Bad Request
	}

	err = a.serv.CreateRoom(ctx, params.RoomName, name, correo, params.Desc, params.Location, params.Lat, params.Long, params.Visible)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Failed to create a room"})
	}

	return c.JSON(http.StatusOK, responseMessage{Message: "Room created successfully"})
}

func (a *API) DeleteProject(c echo.Context) error {

	ctx := c.Request().Context()

	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	// Parsear el token de autenticación
	claims, err := encryption.ParseLoginJWT(auth)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	user := claims["email"].(string)
	id := c.Param("id")

	// Revisar si la sala ya está en memoria
	existingRoom, exists := rooms[id]
	if exists {
		// Si la sala está en memoria, acceder a los miembros directamente
		if existingRoom.ProjectInfo.Members.Owner != user {

			existingRoom.ProjectInfo.Members.Editors = removeUser(existingRoom.ProjectInfo.Members.Editors, user)
			existingRoom.ProjectInfo.Members.Readers = removeUser(existingRoom.ProjectInfo.Members.Readers, user)

			// Actualizar la sala en base de datos
			err = a.repo.UpdateMembers(ctx, id, existingRoom.ProjectInfo.Members)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Failed to delete user from room"})
			}

		} else {
			// Si el usuario es el dueño de la sala, eliminar el proyecto
			err = a.repo.DeleteProject(ctx, id)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Failed to delete room"})
			}
			// También puedes eliminar la sala de la memoria
			delete(rooms, id)
		}
	} else {
		// Si la sala no está en memoria, consultarla desde la base de datos
		proyect, err := a.repo.GetMembers(ctx, id)
		if err != nil {
			return c.JSON(http.StatusNotFound, responseMessage{Message: "Room not found"})
		}

		// Verificar si el usuario es dueño o no
		if proyect.Owner != user {
			// Eliminar al usuario de la sala en la base de datos
			err = a.repo.DeleteUserRoom(ctx, user, id)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Failed to delete user from room"})
			}
		} else {
			// Eliminar el proyecto completo si el usuario es el dueño
			err = a.repo.DeleteProject(ctx, id)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Failed to delete room"})
			}
		}
	}

	return c.JSON(http.StatusOK, responseMessage{Message: "Room deleted successfully"})
}

func removeUser(users []string, userToRemove string) []string {
	for i, u := range users {
		if u == userToRemove {
			return append(users[:i], users[i+1:]...)
		}
	}
	return users
}

func (a *API) HandleGetPublicProject(c echo.Context) error {

	ctx := c.Request().Context()
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	proyects, err := a.repo.HandleGetPublicProject(ctx)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Error getting proyects"})
	}

	response := ProjectResponse{
		Projects: proyects,
	}

	return c.JSON(http.StatusOK, response)
}

func (a *API) HandleEditProfile(c echo.Context) error {
	ctx := c.Request().Context()
	auth := c.Request().Header.Get("Authorization")

	if auth == "" {
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	claims, err := encryption.ParseLoginJWT(auth)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
	}

	email := claims["email"].(string)

	var req dtos.EditProfileRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: "Invalid request"})
	}

	if err := a.dataValidator.Struct(req); err != nil {
		return c.JSON(http.StatusBadRequest, responseMessage{Message: err.Error()})
	}

	err = a.repo.UpdateUserProfile(ctx, req, email)
	if err != nil {
		log.Println(err)
		return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Failed to update profile"})
	}

	return c.JSON(http.StatusOK, responseMessage{Message: "Profile updated successfully"})
}

// func (a *API) HandleInviteUser(c echo.Context) error {

// 	ctx := c.Request().Context()
// 	auth := c.Request().Header.Get("Authorization")

// 	//validar datos
// 	if auth == "" {
// 		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
// 	}

// 	claims, err := encryption.ParseLoginJWT(auth)
// 	if err != nil {
// 		log.Println(err)
// 		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
// 	}

// 	user := claims["email"].(string)
// 	id := c.Param("id")

// 	var newUser dtos.InviteUserRequest

// 	err = c.Bind(&newUser) // llena a params con los datos de la solicitud

// 	if err != nil {
// 		return c.JSON(http.StatusBadRequest, responseMessage{Message: "Invalid request"})
// 		//return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"}) //HTTP 400 Bad Request
// 	}

// 	log.Print(newUser)

// 	if newUser.Email == "" || newUser.Role == "0" {
// 		return c.JSON(http.StatusBadRequest, responseMessage{Message: "Invalid request"})
// 	}

// 	//obtener el proyecto
// 	proyect, err := a.serv.GetRoomInfo(ctx, id)
// 	if err != nil {
// 		return c.JSON(http.StatusNotFound, responseMessage{Message: "Room not found"})
// 	}

// 	//validar si el usuario tiene permisos
// 	if proyect.Members.Owner != user {
// 		return c.JSON(http.StatusUnauthorized, responseMessage{Message: "Unauthorized"})
// 	}

// 	//validar si el usuario ya esta en el proyecto dentro de Members[0], Members[1][array], Members[2][array]
// 	if proyect.Members["0"] == newUser.Email {
// 		return c.JSON(http.StatusBadRequest, responseMessage{Message: "User already in the project"})
// 	}

// 	members1 := proyect.Members["1"].(primitive.A)
// 	members2 := proyect.Members["2"].(primitive.A)

// 	for _, member := range members1 {
// 		if member == newUser.Email {
// 			return c.JSON(http.StatusBadRequest, responseMessage{Message: "User already in the project"})
// 		}
// 	}

// 	for _, member := range members2 {
// 		if member == newUser.Email {
// 			return c.JSON(http.StatusBadRequest, responseMessage{Message: "User already in the project"})
// 		}
// 	}

// 	err = a.repo.AddUserToProject(ctx, newUser.Email, newUser.Role, id)
// 	if err != nil {
// 		return c.JSON(http.StatusInternalServerError, responseMessage{Message: "Failed to invite user"})
// 	}

// 	return c.JSON(http.StatusOK, responseMessage{Message: "User invited successfully"})
// }
