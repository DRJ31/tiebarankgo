package router

import (
	"github.com/DRJ31/tiebarankgo/model"
	"github.com/gofiber/fiber/v2"
)

func GetUsers(c *fiber.Ctx) error {
	db, err := model.Init()
	if err != nil {
		panic(err)
	}
	err = db.AutoMigrate(&model.User{})
	if err != nil {
		panic(err)
	}
	defer model.Close(db)

	var user model.User
	db.First(&user, "name = ?", "辗转一生3")
	return c.JSON(user)
}
