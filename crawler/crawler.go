package crawler

import (
	"fmt"
	"github.com/DRJ31/tiebarankgo/model"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func GetUsers(tieba string, page int) []model.TiebaUser {
	url := fmt.Sprintf("http://tieba.baidu.com/f/like/furank?kw=%s&pn=%v", tieba, page)

	// Get content of webpage
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	// Ensure correct display of Chinese
	utf8Reader := transform.NewReader(res.Body, simplifiedchinese.GBK.NewDecoder())
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("Status code err: %d %s", res.StatusCode, res.Status)
	}

	// Create document from webpage
	doc, err := goquery.NewDocumentFromReader(utf8Reader)
	if err != nil {
		log.Fatal(err)
	}

	tiebaUsers := make([]model.TiebaUser, 0)

	doc.Find(".drl_list_item").Each(func(i int, s *goquery.Selection) {
		// Check if the user is VIP
		vip := s.Find(".drl_item_card").HasClass("drl_item_vip")

		// Get Rank of user
		rank, err := strconv.ParseUint(s.Find(".drl_item_index").Text(), 10, 32)
		if err != nil {
			log.Fatalf("Rank parse err: %v", err)
		}

		// Get experience value of user
		exp, err := strconv.ParseUint(s.Find(".drl_item_exp").Text(), 10, 32)
		if err != nil {
			log.Fatalf("Exp parse err: %v", err)
		}

		// Get link of user
		link, ok := s.Find(".drl_item_card").Find("a").Attr("href")
		if !ok {
			log.Fatal("Find link failed")
		}

		// Get level string of user
		level, ok := s.Find(".drl_item_title").Find("div").Attr("class")
		if !ok {
			log.Fatal("Find level failed")
		}
		level = strings.Split(level, "lv")[1]
		lv, err := strconv.ParseUint(level, 10, 32)
		if err != nil {
			log.Fatal("Parse Level failed")
		}

		// Construct final result
		tiebaUsers = append(tiebaUsers, model.TiebaUser{
			Rank:   rank,
			Member: vip,
			Name:   s.Find(".drl_item_card").Text(),
			Exp:    exp,
			Link:   link,
			Level:  lv,
		})
	})

	return tiebaUsers
}
