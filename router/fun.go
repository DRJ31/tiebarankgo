package router

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
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

func inArr(arr []string, str string) bool {
	for _, s := range arr {
		if s == str {
			return true
		}
	}
	return false
}

func getDelta(newMap, oldMap map[uint]uint) []model.DistRet {
	var result []model.DistRet

	for k, v := range newMap {
		result = append(result, model.DistRet{
			Level: k,
			Rank:  v,
			Delta: int(v),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Level > result[j].Level
	})

	for k, v := range oldMap {
		for i := range result {
			if result[i].Level == k {
				result[i].Delta -= int(v)
			}
		}
	}

	var sum uint = 0
	for i := range result {
		sum += result[i].Rank
		result[i].Rank = sum
	}

	return result
}

func getDist(level, rank uint, server string, ch chan model.DistRet, wg *sync.WaitGroup) {
	defer wg.Done()

	start := time.Now()

	token := secrets.Encrypt(C.SALT, strconv.FormatUint(uint64(rank), 10))
	url := fmt.Sprintf("%v/api/v2/tieba/rank", server)
	var response []byte
	var info model.DistInfo

	rankInfo := model.RankInfo{
		Token: token,
		Rank:  rank,
		Level: level,
	}
	jsonByte, err := json.Marshal(rankInfo)
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	res, err := client.Post(url, "application/json", bytes.NewBuffer(jsonByte))
	if err != nil {
		log.Printf("Crawl err: %v", err)
		return
	}

	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("Status code err: %d %s", res.StatusCode, res.Status)
		return
	}

	response, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return
	}

	err = json.Unmarshal(response, &info)
	if err != nil {
		panic(err)
		return
	}

	result := model.DistRet{
		Rank:  info.Rank,
		Level: info.Level,
		Delta: int(info.Rank),
	}

	elapsed := time.Since(start)
	log.Println(server, level, elapsed)
	ch <- result
}

func convertDivider(old map[uint]uint) map[uint]uint {
	var tmp []model.DistInfo
	mp := make(map[uint]uint)

	for k, v := range old {
		tmp = append(tmp, model.DistInfo{
			Level: k,
			Rank:  v,
		})
	}

	sort.Slice(tmp, func(i, j int) bool {
		return tmp[i].Level > tmp[j].Level
	})

	var sum uint = 0
	for i := range tmp {
		if i > 0 {
			tmp[i].Rank -= sum
		}
		sum += tmp[i].Rank
	}

	for _, elem := range tmp {
		mp[elem.Level] = elem.Rank
	}
	return mp
}

func parseIncomeData(incomeData model.IncomeData) ([]model.Income, uint) {
	incomes := make([]model.Income, 0)

	for _, data := range incomeData.Data.Points[0].Data {
		incomes = append(incomes, model.Income{Date: data[0], Income: data[1]})
	}

	return incomes, incomeData.Data.Points[1].Data[0][1]
}

func refreshData(income *model.UpIncome, wg *sync.WaitGroup) {
	defer wg.Done()

	var max uint = 0
	var sum uint = 0

	incomeData, err := crawler.GetIncomeData(income.Date, income.Date.Add(4*24*time.Hour))
	if err != nil {
		log.Println(err)
		return
	}

	for _, data := range incomeData.Data.Points[0].Data {
		sum += data[1]
		if data[1] > max {
			max = data[1]
		}
	}

	income.Income = sum
	income.Max = max
}

func getMonthIncome() []model.MonthIncome {
	startDate, _ := time.Parse(C.SHORT_DATE, "20200928")
	current := "202009"
	monthIncome := model.MonthIncome{Date: current, Income: 0}
	incomes := make([]model.MonthIncome, 0)

	incomeData, err := crawler.GetIncomeData(startDate, time.Now())
	if err != nil {
		panic(err)
	}

	for _, data := range incomeData.Data.Points[0].Data {
		currentMonth := time.Unix(int64(data[0])/1000, 0).Format(C.MONTHFMT)
		if currentMonth != current {
			incomes = append(incomes, monthIncome)
			nextMonth, _ := time.Parse(C.MONTHFMT, current)
			current = nextMonth.AddDate(0, 1, 0).Format(C.MONTHFMT)
			monthIncome = model.MonthIncome{Date: current, Income: 0}
		}
		monthIncome.Income += data[1]
	}

	incomes = append(incomes, monthIncome)

	return incomes
}
