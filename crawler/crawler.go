package crawler

import (
	"context"
	"errors"
	"fmt"
	"github.com/DRJ31/tiebarankgo/model"
	C "github.com/DRJ31/tiebarankgo/secrets/constants"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type MyError struct {
	Message string
}

func (e *MyError) Error() string {
	return fmt.Sprintf("Error: %v", e.Message)
}

var ErrUserNotFound = errors.New("user not found")

// Get multiple users in a page
func GetUsers(tieba string, page uint) ([]model.TiebaUser, error) {
	url := fmt.Sprintf("http://tieba.baidu.com/f/like/furank?kw=%s&pn=%v", tieba, page)

	// Get content of webpage
	res, err := http.Get(url)
	if err != nil {
		log.Printf("Crawl err: %v", err)
		return nil, err
	}

	// Ensure correct display of Chinese
	utf8Reader := transform.NewReader(res.Body, simplifiedchinese.GBK.NewDecoder())
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("Status code err: %d %s", res.StatusCode, res.Status)
		return nil, &MyError{
			fmt.Sprintf("%d %s", res.StatusCode, res.Status),
		}
	}

	// Create document from webpage
	doc, err := goquery.NewDocumentFromReader(utf8Reader)
	if err != nil {
		log.Printf("New document err: %v", err)
		return nil, err
	}

	tiebaUsers := make([]model.TiebaUser, 0)

	db, err := model.Init()
	defer model.Close(db)

	// Get total users
	total, err := strconv.ParseUint(doc.Find(".drl_info_txt_gray").Text(), 10, 32)
	if err != nil {
		log.Printf("Total parse err: %v", err)
	}
	var ctx = context.Background()
	rdb := model.InitRedis()
	defer rdb.Close()
	rdb.Set(ctx, "tieba_genshin_member_total", total, 0)

	doc.Find(".drl_list_item").Each(func(i int, s *goquery.Selection) {
		// Check if the user is VIP
		vip := s.Find(".drl_item_card").HasClass("drl_item_vip")

		// Get Rank of user
		rank, e := strconv.ParseUint(s.Find(".drl_item_index").Text(), 10, C.BITSIZE)
		if e != nil {
			log.Printf("Rank parse err: %v", e)
			err = e
			return
		}

		// Get experience value of user
		exp, e := strconv.ParseUint(s.Find(".drl_item_exp").Text(), 10, C.BITSIZE)
		if e != nil {
			log.Printf("Exp parse err: %v", e)
			err = e
			return
		}

		// Get link of user
		link, ok := s.Find(".drl_item_card").Find("a").Attr("href")
		if !ok {
			log.Println("Failed to find link")
			err = &MyError{"Failed to find link"}
			return
		}

		// Get level string of user
		level, ok := s.Find(".drl_item_title").Find("div").Attr("class")
		if !ok {
			log.Println("Failed to find level")
			err = &MyError{"Failed to find level"}
			return
		}
		level = strings.Split(level, "lv")[1]
		lv, e := strconv.ParseUint(level, 10, C.BITSIZE)
		if e != nil {
			log.Printf("Level parse err: %v", e)
			err = e
			return
		}

		name := s.Find(".drl_item_card").Text()
		var user model.User
		var nickname string
		result := db.First(&user, "name = ?", name)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			userAvatar, e := GetUser(link)
			if e != nil {
				if !errors.Is(e, ErrUserNotFound) {
					err = e
					return
				}
			}
			nickname = userAvatar.Nickname
		} else {
			nickname = user.Nickname
		}

		// Construct final result
		tiebaUsers = append(tiebaUsers, model.TiebaUser{
			Rank:     uint(rank),
			Member:   vip,
			Name:     name,
			Exp:      uint(exp),
			Link:     link,
			Level:    uint(lv),
			Nickname: nickname,
		})
	})

	if err != nil {
		return nil, err
	}
	return tiebaUsers, nil
}

// Get single user information
func GetUser(url string) (model.UserAvatar, error) {
	res, err := http.Get("http://tieba.baidu.com" + url)
	if err != nil {
		log.Printf("Crawl err: %v", err)
		return model.UserAvatar{}, err
	}

	// Ensure correct display of Chinese
	//utf8Reader := transform.NewReader(res.Body, simplifiedchinese.GBK.NewDecoder())
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("Status code err: %d %s", res.StatusCode, res.Status)
		return model.UserAvatar{}, &MyError{
			fmt.Sprintf("%d %s", res.StatusCode, res.Status),
		}
	}

	// Create document from webpage
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("New document err: %v", err)
		return model.UserAvatar{}, err
	}

	avatar, ok := doc.Find(".userinfo_left_head").Find("img").Attr("src")
	if !ok {
		return model.UserAvatar{}, ErrUserNotFound
	}

	nicknameArr := strings.Split(doc.Find("title").Text(), "的贴吧")

	return model.UserAvatar{Avatar: avatar, Nickname: nicknameArr[0]}, nil
}
