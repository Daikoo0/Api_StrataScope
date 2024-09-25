package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ProyectoT/api/database"
	"github.com/ProyectoT/api/internal/api"
	"github.com/ProyectoT/api/internal/repository"
	"github.com/ProyectoT/api/internal/service"
	"github.com/ProyectoT/api/settings"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"go.uber.org/fx"
)

func main() {
	log.Println("hola desde main")

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")

	}

	//patron de diseño inyección de dependencias
	app := fx.New(
		fx.Provide(
			context.Background,
			settings.New,
			database.New,
			repository.New,
			service.New,
			api.New,
			echo.New,
		),
		fx.Invoke(
			setLifeCycle,
		),
	)
	app.Run()
}
func setLifeCycle(lc fx.Lifecycle, a *api.API, s *settings.Settings, e *echo.Echo) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			address := fmt.Sprintf(":%s", s.Port)
			go a.Start(e, address)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return nil
		},
	})
}
