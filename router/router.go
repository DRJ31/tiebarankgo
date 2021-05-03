package router

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/DRJ31/tiebarankgo/crawler"
	"github.com/DRJ31/tiebarankgo/model"
	"github.com/DRJ31/tiebarankgo/secrets"
	C "github.com/DRJ31/tiebarankgo/secrets/constants"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"log"
	"strconv"
	"time"
)

var ctx = context.Background()

func GetUsers(c *fiber.Ctx) error {
	// Check token
	token := c.Query("token")
	pg := c.Query("page")
	if !secrets.TokenCheck(C.SALT, pg, token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	// Initialize Redis
	rdb := model.InitRedis()
	defer rdb.Close()

	// Get page information
	page, err := strconv.ParseUint(pg, 10, C.BITSIZE)
	if err != nil {
		log.Printf("Page parse err: %v", err)
		return err
	}
	pageSize, err := strconv.ParseUint(c.Query("pageSize"), 10, C.BITSIZE)
	if err != nil {
		log.Printf("PageSize parse err: %v", err)
		return err
	}

	// Check page number with different page size
	realPage := page
	if pageSize == 10 {
		realPage = page / 2
		if page%2 == 1 {
			realPage += 1
		}
	}

	// Get total number of genshin tieba member
	total, err := rdb.Get(ctx, "tieba_genshin_member_total").Result()
	if err != nil {
		total = C.MINUSER
	}
	totalMember, err := strconv.ParseUint(total, 10, C.BITSIZE)
	if err != nil {
		log.Println(err)
		return err
	}

	// Check if the users in the page are cached
	byteUsers, err := rdb.Get(ctx, "tieba_genshin_page_"+strconv.FormatUint(realPage, 10)).Bytes()
	var users []model.TiebaUser
	if err != nil {
		log.Println(err)
		users, err = crawler.GetUsers(C.TIEBA, uint(realPage))
		if err != nil {
			return err
		}
		byteUsers, _ = json.Marshal(users)
		rdb.Set(ctx, "tieba_genshin_page_"+strconv.FormatUint(realPage, 10), byteUsers, time.Minute)
	} else {
		err = json.Unmarshal(byteUsers, &users)
		if err != nil {
			log.Println("Unmarshal user failed")
			users, err = crawler.GetUsers(C.TIEBA, uint(realPage))
			if err != nil {
				return err
			}
		}
	}

	// Initialize database
	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	// Renew user information
	uss := make([]model.User, 0, 20)
	for _, user := range users {
		var oldUser model.User
		res := db.First(&oldUser, "name = ?", user.Name)
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			uss = append(uss, model.User{
				Rank:     user.Rank,
				Level:    user.Level,
				Exp:      user.Exp,
				Member:   user.Member,
				Link:     user.Link,
				Name:     user.Name,
				Nickname: user.Nickname,
			})
		} else {
			db.Model(&oldUser).Updates(model.User{
				Rank:     user.Rank,
				Level:    user.Level,
				Exp:      user.Exp,
				Member:   user.Member,
				Nickname: user.Nickname,
			})
		}
	}
	if len(uss) > 0 {
		db.Create(&uss)
	}

	var result []model.TiebaUser
	if pageSize == 10 {
		if page%2 != 0 {
			result = users[:10]
		} else {
			result = users[10:]
		}
	} else {
		result = users
	}

	return c.JSON(fiber.Map{
		"users": result,
		"total": totalMember,
	})
}

func GetUser(c *fiber.Ctx) error {
	var ul model.UserLink

	if err := c.BodyParser(&ul); err != nil {
		return err
	}

	if !secrets.TokenCheck(C.SALT, ul.Link, ul.Token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid request"})
	}

	// Get user information
	result, err := crawler.GetUser(ul.Link)
	if err != nil {
		return err
	}

	// Initialize database
	db, err := model.Init()
	if err != nil {
		return err
	}
	defer model.Close(db)

	var user model.User
	db.First(&user, "link = ?", ul.Link)
	if result.Nickname != user.Nickname {
		db.Model(&user).Updates(model.User{Nickname: result.Nickname})
	}

	return c.JSON(fiber.Map{
		"user": result,
	})
}
