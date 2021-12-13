package crawler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DRJ31/tiebarankgo/config"
	"github.com/DRJ31/tiebarankgo/model"
	C "github.com/DRJ31/tiebarankgo/secrets/constants"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-redis/redis/v8"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MyError struct {
	Message string
}

func (e *MyError) Error() string {
	return fmt.Sprintf("Error: %v", e.Message)
}

var ErrUserNotFound = errors.New("user not found")

// GetUsers Get multiple users in a page
func GetUsers(tieba string, page uint) ([]model.TiebaUser, error) {
	site := fmt.Sprintf("http://tieba.baidu.com/f/like/furank?kw=%s&pn=%v", tieba, page)

	// Get content of webpage
	res, err := http.Get(site)
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
	total, err := strconv.ParseUint(doc.Find(".drl_info_txt_gray").Text(), C.BASE, C.BITSIZE)
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
		rank, e := strconv.ParseUint(s.Find(".drl_item_index").Text(), C.BASE, C.BITSIZE)
		if e != nil {
			log.Printf("Rank parse err: %v", e)
			err = e
			return
		}

		// Get experience value of user
		exp, e := strconv.ParseUint(s.Find(".drl_item_exp").Text(), C.BASE, C.BITSIZE)
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
		lv, e := strconv.ParseUint(level, C.BASE, C.BITSIZE)
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

// GetUser Get single user information
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

// GetDistribution Get multiple users in a page
func GetDistribution(tieba string, page int, level uint, ch chan uint, wg *sync.WaitGroup) {
	defer wg.Done()
	site := fmt.Sprintf("http://tieba.baidu.com/f/like/furank?kw=%s&pn=%v", tieba, page)

	// Get content of webpage
	res, err := http.Get(site)
	if err != nil {
		log.Printf("Crawl err: %v", err)
		return
	}

	// Ensure correct display of Chinese
	utf8Reader := transform.NewReader(res.Body, simplifiedchinese.GBK.NewDecoder())
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("Status code err: %d %s", res.StatusCode, res.Status)
		return
	}

	// Create document from webpage
	doc, err := goquery.NewDocumentFromReader(utf8Reader)
	if err != nil {
		log.Printf("New document err: %v", err)
		return
	}

	doc.Find(".drl_list_item").Each(func(i int, s *goquery.Selection) {
		// Get Rank of user
		rank, e := strconv.ParseUint(s.Find(".drl_item_index").Text(), C.BASE, C.BITSIZE)
		if e != nil {
			log.Printf("Rank parse err: %v", e)
			err = e
			return
		}

		// Get level string of user
		levelStr, ok := s.Find(".drl_item_title").Find("div").Attr("class")
		if !ok {
			log.Println("Failed to find level")
			err = &MyError{"Failed to find level"}
			return
		}
		levelStr = strings.Split(levelStr, "lv")[1]
		lv, e := strconv.ParseUint(levelStr, C.BASE, C.BITSIZE)
		if e != nil {
			log.Printf("Level parse err: %v", e)
			err = e
			return
		}

		if uint(lv) < level {
			ch <- uint(rank - 1)
		}
	})
}

// GetTotal Get total number of posts and members
func GetTotal() (uint, uint, error) {
	rdb := model.InitRedis()
	defer rdb.Close()
	var ctx = context.Background()

	members, err := rdb.Get(ctx, "tieba_genshin_total").Uint64()
	if err != nil {
		return getTotal(rdb, ctx)
	}

	posts, err := rdb.Get(ctx, "tieba_genshin_post_total").Uint64()
	if err != nil {
		return getTotal(rdb, ctx)
	}

	return uint(posts), uint(members), nil
}

func getTotal(rdb *redis.Client, ctx context.Context) (uint, uint, error) {
	res, err := http.Get(fmt.Sprintf("http://tieba.baidu.com/f?ie=utf-8&kw=%s", C.TIEBA))
	if err != nil {
		log.Printf("Crawl err: %v", err)
		return 0, 0, err
	}

	// Ensure correct display of Chinese
	//utf8Reader := transform.NewReader(res.Body, simplifiedchinese.GBK.NewDecoder())
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("Status code err: %d %s", res.StatusCode, res.Status)
		return 0, 0, &MyError{
			fmt.Sprintf("%d %s", res.StatusCode, res.Status),
		}
	}

	// Create document from webpage
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Printf("New document err: %v", err)
		return 0, 0, err
	}

	memberStr := doc.Find(".card_menNum").Text()
	memberStr = strings.Join(strings.Split(memberStr, ","), "")
	postStr := doc.Find(".card_infoNum").Text()
	postStr = strings.Join(strings.Split(postStr, ","), "")

	members, err := strconv.ParseUint(memberStr, C.BASE, C.BITSIZE)
	if err != nil {
		log.Println(err)
		return 0, 0, err
	}
	posts, err := strconv.ParseUint(postStr, C.BASE, C.BITSIZE)
	if err != nil {
		log.Println(err)
		return 0, 0, err
	}

	rdb.Set(ctx, "tieba_genshin_total", members, time.Minute)
	rdb.Set(ctx, "tieba_genshin_post_total", posts, time.Minute)

	return uint(posts), uint(members), nil
}

func GetIncomeData(start, end time.Time) (model.IncomeData, error) {
	endTime := end
	if endTime.Unix() > time.Now().Add(-24*time.Hour).Unix() {
		endTime = time.Now().Add(-24 * time.Hour)
	}

	startDate := start.Format(C.SHORT_DATE)
	endDate := endTime.Format(C.SHORT_DATE)

	location := fmt.Sprintf("https://www.chandashi.com/interf/v1/apps/incomeEstimateLine?country=cn&appId=1467190251&startDate=%v&endDate=%v", startDate, endDate)

	// Construct request
	cf := config.GetConfig()
	jar, _ := cookiejar.New(nil)
	cookies := make([]*http.Cookie, 0)
	cookies = append(cookies, &http.Cookie{
		Name:  "cds_session_id",
		Value: cf.SessionId,
	})
	cookies = append(cookies, &http.Cookie{
		Name:  "cds_asm_token",
		Value: cf.AsmToken,
	})
	loc, _ := url.Parse(location)
	jar.SetCookies(loc, cookies)

	client := &http.Client{Jar: jar}

	req, err := http.NewRequest("GET", location, nil)
	if err != nil {
		return model.IncomeData{}, err
	}

	// Get content of webpage
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Crawl err: %v", err)
		return model.IncomeData{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return model.IncomeData{}, err
	}

	var income model.IncomeData

	err = json.Unmarshal(body, &income)
	if err != nil {
		fmt.Println(startDate, endDate)
		fmt.Printf("%s", body)
		log.Println(err)
		return model.IncomeData{}, err
	}

	return income, nil
}
