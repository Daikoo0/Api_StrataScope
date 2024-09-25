package api

import (
	"github.com/labstack/echo/v4"
)

func (a *API) RegisterRoutes(e *echo.Echo) {

	users := e.Group("/users")

	users.POST("/logout", a.LogoutUser)            // users/logout
	users.POST("/register", a.RegisterUser)        // users/register
	users.POST("/login", a.LoginUser)              // users/login
	users.GET("/auth", a.AuthUser)                 // users/Auth
	users.GET("/projects", a.projects)             // users
	users.DELETE("/projects/:id", a.DeleteProject) // users/projects/:id

	e.GET("/search/public", a.HandleGetPublicProject) // search/public
	e.GET("/ws/:room", a.HandleWebSocket)             //ws/sala
	e.POST("/validate-invitation", a.ValidateInvitation)
	e.POST("/rooms/create", a.HandleCreateProyect) //rooms/sala/usuario
	e.POST("/comment", a.AddComment)

	e.GET("/activeProject", a.HandleGetActiveProject)
}
