package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/DRJ31/tiebarankgo/crawler"
	"github.com/DRJ31/tiebarankgo/model"
	"github.com/DRJ31/tiebarankgo/secrets"
	C "github.com/DRJ31/tiebarankgo/secrets/constants"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
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
	}

	return nil
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

	arr := make([]model.TiebaUser, 0)
	for i := 1; i <= length; i++ {
		users, e := crawler.GetUsers(C.TIEBA, uint(i))
		if e != nil {
			panic(e)
		}

		ch := make(chan model.TiebaUser)
		for _, user := range users {
			wg.Add(1)
			go func(u model.TiebaUser, ch chan model.TiebaUser) {
				info, e := crawler.GetUser(u.Link)
				if e != nil {
					panic(e)
				}
				u.Nickname = info.Nickname
				ch <- u
			}(user, ch)
		}
		go func() {
			wg.Wait()
			close(ch)
		}()

		for u := range ch {
			arr = append(arr, u)
		}

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
}
