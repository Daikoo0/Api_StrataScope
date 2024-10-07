package api

import (
	"github.com/ProyectoT/api/internal/repository"
	"github.com/ProyectoT/api/internal/service"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type API struct {
	serv          service.Service
	repo          repository.Repository
	dataValidator *validator.Validate
}

func New(serv service.Service, repo repository.Repository) *API {
	return &API{
		serv:          serv,
		repo:          repo,
		dataValidator: validator.New(),
	}
}

func (a *API) Start(e *echo.Echo, address string) error {
	a.RegisterRoutes(e)

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:*", "http://127.0.0.1:*", "https://stratascope.inf.uct.cl", "http://localhost*"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.DELETE},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	return e.Start(address)
}
