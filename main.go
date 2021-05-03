package main

import (
	"fmt"
	"github.com/DRJ31/tiebarankgo/config"
	"github.com/DRJ31/tiebarankgo/router"
	"github.com/gofiber/fiber/v2"
)

func initRouter(app *fiber.App) {
	app.Get("/api/tieba/users", router.GetUsers)
}

func main() {
	app := fiber.New()
	initRouter(app)
	cf := config.GetConfig()
	app.Listen(fmt.Sprintf("%v:%v", cf.Host, cf.Port))
}
