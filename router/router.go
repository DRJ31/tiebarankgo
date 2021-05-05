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
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var ctx = context.Background()

// Get users of a page
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
	page, err := strconv.ParseUint(pg, C.BASE, C.BITSIZE)
	if err != nil {
		log.Printf("Page parse err: %v", err)
		return err
	}
	pageSize, err := strconv.ParseUint(c.Query("pageSize"), C.BASE, C.BITSIZE)
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
	total, err := rdb.Get(ctx, "tieba_genshin_member_total").Uint64()
	if err != nil {
		total = C.MINUSER
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

	// Decide how many data to display according to page size
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
		"total": total,
	})
}

// Get avatar of a user
func GetUser(c *fiber.Ctx) error {
	var ul model.UserLink

	if err := c.BodyParser(&ul); err != nil {
		return err
	}

	if !secrets.TokenCheck(C.SALT, ul.Link, ul.Token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	// Get user information
	result, err := crawler.GetUser(ul.Link)
	if err != nil {
		return err
	}

	// Initialize database
	db, err := model.Init()
	if err != nil {
		log.Println(err)
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

// Get all anniversaries
func GetAnniversaries(c *fiber.Ctx) error {
	var anniversaries []model.Anniversary

	// Initialize database
	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	db.Find(&anniversaries)

	return c.JSON(fiber.Map{"anniversaries": anniversaries})
}

// Get event of today
func GetEvent(c *fiber.Ctx) error {
	token := c.Query("token")
	day := c.Query("date")
	if !secrets.TokenCheck(C.SALT, day, token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	dayStr := strings.Split(day, "-")
	dayInt := make([]int, 0, 3)
	for _, ds := range dayStr {
		di, err := strconv.ParseInt(ds, C.BASE, C.BITSIZE)
		if err != nil {
			log.Println(err)
			return err
		}
		dayInt = append(dayInt, int(di))
	}
	d := time.Date(dayInt[0], time.Month(dayInt[1]), dayInt[2], 0, 0, 0, 0, time.Local)

	// Initialize database
	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	var events []string
	var data []model.Event
	result := db.Find(&data, "date = ?", d)
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		for _, e := range data {
			events = append(events, e.Event)
		}
	}

	return c.JSON(fiber.Map{"event": events})
}

// Get all events
func GetEvents(c *fiber.Ctx) error {
	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	var days, event []string
	var results []model.EventRet
	var data []model.Event
	res := db.Find(&data, "date = ?", time.Now())
	if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
		for _, e := range data {
			event = append(event, e.Event)
		}
	}

	db.Find(&data)
	for _, e := range data {
		dayStr := e.Date.Format(C.DATEFMT)
		results = append(results, model.EventRet{
			Event: e.Event,
			Date:  dayStr,
		})
		if !inArr(days, dayStr) {
			days = append(days, dayStr)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Date > results[j].Date
	})

	return c.JSON(fiber.Map{
		"event":  event,
		"days":   days,
		"events": results,
	})
}

// Get post info of today
func GetOnePost(c *fiber.Ctx) error {
	token := c.Query("token")
	date := c.Query("date")
	if !secrets.TokenCheck(C.SALT, date, token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	posts, _, err := crawler.GetTotal()
	if err != nil {
		log.Println(err)
		return err
	}

	return c.JSON(fiber.Map{"total": posts})
}

// Get all posts info
func GetMultiplePosts(c *fiber.Ctx) error {
	token := c.Query("token")
	page := c.Query("page")
	if !secrets.TokenCheck(C.SALT, page, token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	var data []model.Post
	var results []model.PostRet
	db.Order("date desc").Find(&data)

	for _, d := range data {
		results = append(results, model.PostRet{
			Total: d.Total,
			Date:  d.Date.Format(C.DATEFMT),
		})
	}

	return c.JSON(fiber.Map{"results": results})
}

// Find users by keyword
func FindUsers(c *fiber.Ctx) error {
	token := c.Query("token")
	keyword := c.Query("keyword")
	if !secrets.TokenCheck(C.SALT, keyword, token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	keyword = "%" + keyword + "%"
	var users []model.User

	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	db.Where("name LIKE ?", keyword).Or("nickname LIKE ?", keyword).Find(&users)

	return c.JSON(fiber.Map{"users": users})
}

// Get distribution of specific rank
func GetRank(c *fiber.Ctx) error {
	var info model.RankInfo
	err := c.BodyParser(&info)
	if err != nil {
		log.Println(err)
		return err
	}

	var MAXTHREAD = runtime.NumCPU() * C.THREADS
	var wg sync.WaitGroup
	var min uint = 0

	if !secrets.TokenCheck(C.SALT, strconv.FormatUint(uint64(info.Rank), 10), info.Token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	startPage := info.Rank / 20

	for {
		ch := make(chan uint)
		for i := 0; i < MAXTHREAD; i++ {
			wg.Add(1)
			go crawler.GetDistribution(C.TIEBA, int(startPage)+i, info.Level, ch, &wg)
		}
		go func() {
			wg.Wait()
			close(ch)
		}()
		for rank := range ch {
			if min == 0 || rank < min {
				min = rank
			}
		}
		if min > 0 {
			break
		}
		startPage += uint(MAXTHREAD)
	}

	return c.JSON(fiber.Map{"rank": min})
}
