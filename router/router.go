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
	"math"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"
)

var ctx = context.Background()

// GetUsers Get users of a page
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

// GetUser Get avatar of a user
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

// GetAnniversaries Get all anniversaries
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

// GetEvent Get event of today
func GetEvent(c *fiber.Ctx) error {
	token := c.Query("token")
	day := c.Query("date")
	if !secrets.TokenCheck(C.SALT, day, token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	d, err := time.Parse(C.DATEFMT, day)
	if err != nil {
		log.Println(err)
		return err
	}

	// Initialize database
	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	events := make([]string, 0)
	var data []model.Event
	result := db.Find(&data, "date = ?", d.Format(C.DATEFMT))
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		for _, e := range data {
			events = append(events, e.Event)
		}
	}

	var upIncome []model.UpIncome
	result = db.Find(&upIncome, "date = ?", d.Format(C.DATEFMT))
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		for _, e := range upIncome {
			events = append(events, e.Name+"池")
		}
	}

	return c.JSON(fiber.Map{"event": events})
}

// GetEvents Get all events
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
	var upIncome []model.UpIncome

	res := db.Find(&data, "date = ?", time.Now().Format(C.DATEFMT))
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

	db.Find(&upIncome)
	for _, d := range upIncome {
		dayStr := d.Date.Format(C.DATEFMT)
		results = append(results, model.EventRet{
			Event: d.Name + "池",
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

// GetOnePost Get post info of today
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

// GetMultiplePosts Get all posts info
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

// FindUsers Find users by keyword
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

// GetRank Get distribution of specific rank
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

	sp := math.Ceil(float64(info.Rank) / 20) // Float format of startPage
	startPage := uint(sp)

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

	return c.JSON(fiber.Map{
		"rank":  min,
		"level": info.Level,
	})
}

func GetDist(c *fiber.Ctx) error {
	token := c.Query("token")
	dateStr := c.Query("date")
	if !secrets.TokenCheck(C.SALT, dateStr, token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	day, err := time.Parse(C.DATEFMT, dateStr)
	if err != nil {
		log.Println(err)
		return err
	}

	// Initialize database
	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	currentDate := time.Now().Add(time.Hour * 8).Truncate(time.Hour * 24)
	var oldDivider map[uint]uint
	if day.Equal(currentDate) {
		rdb := model.InitRedis()
		defer rdb.Close()

		var firstDivider model.Divider
		var firstUser model.User
		var dividers []model.Divider
		var history model.History

		db.Order("level desc").First(&firstDivider)
		db.Order("level desc").First(&firstUser)

		if firstDivider.Level < firstUser.Level {
			db.Create(&model.Divider{
				Level: firstUser.Level,
				Rank:  1,
			})
		}

		db.Order("level desc").Find(&dividers)
		lastDay := day.Add(time.Duration(-24) * time.Hour)
		res := db.First(&history, "date = ?", lastDay.Format(C.DATEFMT))
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Println(res.Error)
			return res.Error
		}
		err = json.Unmarshal([]byte(history.Distribution), &oldDivider)
		if err != nil {
			panic(err)
			return err
		}

		var dist []model.DistRet
		if byteDivider, err := rdb.Get(ctx, "tieba_genshin_divider").Bytes(); err != nil {
			var wg sync.WaitGroup
			ch := make(chan model.DistRet)

			for _, div := range dividers {
				wg.Add(1)
				go getDist(div.Level, div.Rank, secrets.DistributeServer(div.Level), ch, &wg)
			}
			go func() {
				wg.Wait()
				close(ch)
			}()

			newDivider := make(map[uint]uint)
			for dr := range ch {
				newDivider[dr.Level] = dr.Rank
				db.Model(&model.Divider{}).Where("level = ?", dr.Level).Update("rank", dr.Rank)
			}
			dist = getDelta(convertDivider(newDivider), oldDivider)
		} else {
			err = json.Unmarshal(byteDivider, &dist)
			if err != nil {
				log.Println(err)
				return err
			}
		}

		posts, members, err := crawler.GetTotal()
		if err != nil {
			log.Println(err)
			return err
		}

		membership, err := rdb.Get(ctx, "tieba_genshin_member_total").Uint64()
		if err != nil {
			log.Println(err)
			return err
		}

		var users []model.User
		resp := db.Find(&users, "member = ?", 1)

		sort.Slice(dist, func(i, j int) bool {
			return dist[i].Level > dist[j].Level
		})

		byteDist, err := json.Marshal(dist)
		rdb.Set(ctx, "tieba_genshin_divider", byteDist, 10*time.Minute)

		return c.JSON(fiber.Map{
			"distribution": dist,
			"total":        members,
			"membership":   membership,
			"vip":          resp.RowsAffected,
			"posts":        posts,
			"signin":       0,
		})
	} else {
		lastDay := day.Add(time.Duration(-24) * time.Hour)
		var oldHistory, newHistory model.History
		var newDivider map[uint]uint
		var postInfo model.Post

		res := db.First(&newHistory, "date = ?", day.Format(C.DATEFMT))
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Println(res.Error)
			return res.Error
		}

		res = db.First(&oldHistory, "date = ?", lastDay.Format(C.DATEFMT))
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Println(res.Error)
			return res.Error
		}

		err = json.Unmarshal([]byte(newHistory.Distribution), &newDivider)
		if err != nil {
			panic(err)
			return err
		}
		err = json.Unmarshal([]byte(oldHistory.Distribution), &oldDivider)
		if err != nil {
			panic(err)
			return err
		}

		result := getDelta(newDivider, oldDivider)

		res = db.First(&postInfo, "date = ?", day.Format(C.DATEFMT))
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Println(res.Error)
			return res.Error
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].Level > result[j].Level
		})

		return c.JSON(fiber.Map{
			"distribution": result,
			"total":        postInfo.Followers,
			"membership":   postInfo.Members,
			"vip":          postInfo.Vip,
			"posts":        postInfo.Total,
			"signin":       postInfo.Signin,
		})
	}
}

func InsertPostInfo(c *fiber.Ctx) error {
	var postInfo model.PostInfo
	err := c.BodyParser(&postInfo)
	if err != nil {
		log.Println(err)
		return err
	}

	rdb := model.InitRedis()
	defer rdb.Close()

	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	members, err := rdb.Get(ctx, "tieba_genshin_member_total").Uint64()
	if err != nil {
		log.Println(err)
		return err
	}

	var users []model.User
	queryRes := db.Find(&users, "member = ?", 1)
	vip := queryRes.RowsAffected

	if !secrets.TokenCheck(C.SALT, strconv.FormatUint(uint64(postInfo.Total), 10), postInfo.Token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	var now time.Time
	if time.Now().Hour() < 8 {
		now = time.Now().AddDate(0, 0, -1)
	} else {
		now = time.Now()
	}

	post := model.Post{
		Total:     postInfo.Total,
		Date:      now,
		Followers: postInfo.Followers,
		Members:   uint(members),
		Vip:       uint(vip),
		Signin:    postInfo.Signin,
	}
	db.Create(&post)

	var distribute []model.Divider
	db.Find(&distribute)
	distMap := make(map[uint]uint)
	for _, v := range distribute {
		distMap[v.Level] = v.Rank
	}
	distByte, err := json.Marshal(convertDivider(distMap))
	if err != nil {
		log.Println(err)
		return err
	}
	db.Create(&model.History{
		Date:         time.Now(),
		Distribution: string(distByte),
	})

	return c.JSON(fiber.Map{"data": post})
}

func GetIncome(c *fiber.Ctx) error {
	token := c.Query("token")
	startDate := c.Query("start")
	endDate := c.Query("end")
	upIncome := make([]model.UpIncome, 0)
	var wg sync.WaitGroup

	if !secrets.TokenCheck(C.SALT, startDate+endDate, token) {
		c.Status(400)
		return c.JSON(fiber.Map{"message": "Invalid Request"})
	}

	startTime, _ := time.Parse(C.SHORT_DATE, startDate)
	endTime, _ := time.Parse(C.SHORT_DATE, endDate)
	incomeData, err := crawler.GetIncomeData(startTime, endTime)
	if err != nil {
		c.Status(500)
		return err
	}

	db, err := model.Init()
	if err != nil {
		log.Println(err)
		return err
	}
	defer model.Close(db)

	// Get data of income in a period of time
	incomes, average := parseIncomeData(incomeData)

	// Refresh data of UpIncome
	db.Find(&upIncome)
	for i := range upIncome {
		if upIncome[i].Date.Add(30*24*time.Hour).Unix() > time.Now().Unix() {
			wg.Add(1)
			refreshData(&upIncome[i], &wg)
		}
	}
	wg.Wait()
	db.Save(&upIncome)

	sort.Slice(upIncome, func(i, j int) bool {
		return upIncome[i].Date.Unix() > upIncome[j].Date.Unix()
	})

	monthIncome := getMonthIncome()

	return c.JSON(fiber.Map{
		"average": average,
		"income":  upIncome,
		"data":    incomes,
		"month":   monthIncome,
	})
}
