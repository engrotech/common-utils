package service

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// create a function for initFiber
func InitFiber() *fiber.App {
	app := fiber.New()
	app.Use(logger.New())
	// middleware for cors handling
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "POST, GET, PUT, DELETE",
		AllowHeaders:     "Access-Control-Allow-Origin, Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization ",
		AllowCredentials: false,
		ExposeHeaders:    "Content-Length",
		MaxAge:           86400,
	}))
	// compress middleware
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // 1
	}))

	return app
}
