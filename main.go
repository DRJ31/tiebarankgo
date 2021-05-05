package main

import (
	"fmt"
	"github.com/DRJ31/tiebarankgo/config"
	"github.com/DRJ31/tiebarankgo/router"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func InitRouter(app *fiber.App) {
	app.Get("/api/v2/tieba/users", router.GetUsers)
	app.Get("/api/v2/tieba/event", router.GetEvent)
	app.Get("/api/v2/tieba/anniversary", router.GetAnniversaries)
	app.Get("/api/v2/tieba/events", router.GetEvents)
	app.Get("/api/v2/tieba/post", router.GetOnePost)
	app.Get("/api/v2/tieba/posts", router.GetMultiplePosts)
	app.Get("/api/v2/tieba/user", router.FindUsers)
	app.Post("/api/v2/tieba/user", router.GetUser)
	app.Post("/api/v2/tieba/rank", router.GetRank)
}

func main() {
	app := fiber.New()
	app.Use(cors.New())
	app.Use(compress.New())
	InitRouter(app)
	cf := config.GetConfig()
	_ = app.Listen(fmt.Sprintf("%v:%v", cf.Host, cf.Port))
}
