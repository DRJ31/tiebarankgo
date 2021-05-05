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
	"strconv"
	"sync"
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

	for k, v := range oldMap {
		for i := range result {
			if result[i].Rank == k {
				result[i].Delta -= int(v)
			}
		}
	}

	return result
}

func getDist(level, rank uint, server string, ch chan model.DistRet, wg *sync.WaitGroup) {
	defer wg.Done()

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

	ch <- result
}
