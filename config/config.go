package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type Config struct {
	Host      string `json:"host"`
	Port      uint   `json:"port"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Database  string `json:"database"`
	DBHost    string `json:"db_host"`
	DBPort    uint   `json:"db_port"`
	RedisHost string `json:"redis_host"`
	RedisPort string `json:"redis_port"`
	SessionId string `json:"session_id"`
	AsmToken  string `json:"asm_token"`
	Servers	[]ServerDistribution `json:"servers"`
}

type ServerDistribution struct {
	Level	uint `json:"level"`
	Server	string `json:"server"`
}

func GetConfig() Config {
	jsonFile, err := os.Open("etc/config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	data, _ := ioutil.ReadAll(jsonFile)

	var result Config
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Fatal(err)
	}

	return result
}
