package router

import (
	"bytes"
	"encoding/json"
	"fmt"
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
