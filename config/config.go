package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type Config struct {
	Host     string `json:"host"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	DBHost   string `json:"db_host"`
	DBPort   uint   `json:"db_port"`
}

func GetConfig() Config {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	data, _ := ioutil.ReadAll(jsonFile)

	var result Config
	err = json.Unmarshal(data, &result)
	if err != nil {
		panic(err)
	}

	return result
}
