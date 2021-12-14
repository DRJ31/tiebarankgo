package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/DRJ31/tiebarankgo/crawler"
	"github.com/DRJ31/tiebarankgo/model"
	"github.com/DRJ31/tiebarankgo/secrets"
	C "github.com/DRJ31/tiebarankgo/secrets/constants"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type UsersRet struct {
	Total int               `json:"total"`
	Users []model.TiebaUser `json:"users"`
}

type UsersSent struct {
	Token string            `json:"token"`
	Users []model.TiebaUser `json:"users"`
}

func randInt(min, max int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return min + r.Intn(max-min)
}

func sendNotification(total int, key string) {
	content := fmt.Sprintf("### 用户信息\n总人数: <font color=\"comment\">%d</font>", total)
	loc := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=%v", key)

	var msg secrets.WxMsgMarkdown
	msg.Markdown.Content = content
	msg.Msgtype = "markdown"

	jsonStr, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", loc, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(body))
}

func sendUsers(users []model.TiebaUser, page int) error {
	token := secrets.Encrypt(C.SALT, users[0].Name)
	loc := "https://api.drjchn.com/api/v2/tieba/users"
	usersSent := UsersSent{
		Token: token,
		Users: users,
	}
	jsonStr, e := json.Marshal(usersSent)
	if e != nil {
		return e
	}
	req, e := http.NewRequest("POST", loc, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, e := client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Printf("Submission of %d succeeded.", page)
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
	}

	return nil
}

func getUsers(tieba string, page uint) ([]model.TiebaUser, error) {
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
		return nil, &crawler.MyError{
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
			err = &crawler.MyError{"Failed to find link"}
			return
		}

		// Get level string of user
		level, ok := s.Find(".drl_item_title").Find("div").Attr("class")
		if !ok {
			log.Println("Failed to find level")
			err = &crawler.MyError{"Failed to find level"}
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
		var nickname string
		userAvatar, e := crawler.GetUser(link)
		if e != nil {
			log.Println(e)
			return
		}
		nickname = userAvatar.Nickname

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

func main() {
	token := secrets.Encrypt(C.SALT, "1")
	loc := fmt.Sprintf("https://api.drjchn.com/api/v2/tieba/users?page=1&token=%v&pageSize=10", token)
	var wg sync.WaitGroup

	res, err := http.Get(loc)
	if err != nil {
		log.Printf("Crawl err: %v", err)
		panic(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var usersRet UsersRet
	err = json.Unmarshal(body, &usersRet)
	if err != nil {
		panic(err)
	}

	length := usersRet.Total / 20
	fmt.Println(length)

	arr := make([]model.TiebaUser, 0)
	for i := 1; i <= length; i++ {
		users, e := getUsers(C.TIEBA, uint(i))
		if e != nil {
			panic(e)
		}

		ch := make(chan model.TiebaUser)
		for _, user := range users {
			wg.Add(1)
			go func(u model.TiebaUser, ch chan model.TiebaUser, wg *sync.WaitGroup) {
				defer wg.Done()
				info, e := crawler.GetUser(u.Link)
				if e != nil {
					panic(e)
				}
				u.Nickname = info.Nickname
				ch <- u
			}(user, ch, &wg)
		}
		go func() {
			wg.Wait()
			close(ch)
		}()

		for u := range ch {
			arr = append(arr, u)
		}

		fmt.Printf("Page %d done.\n", i)

		if i%100 == 0 {
			err = sendUsers(arr, i)
			if err != nil {
				panic(err)
			}
			arr = make([]model.TiebaUser, 0)
			time.Sleep(time.Duration(randInt(30, 60)) * time.Second)
			fmt.Printf("%s Sleeping: %d", time.Now().Format(C.TIMEFMT), i)
		}

		if i%10 == 0 {
			time.Sleep(time.Duration(randInt(10, 30)) * time.Second)
			fmt.Printf("%s Sleeping: %d", time.Now().Format(C.TIMEFMT), i)
		}
	}

	key := os.Getenv("NOTIFY_KEY")
	if len(key) > 0 {
		sendNotification(usersRet.Total, key)
	}
}
