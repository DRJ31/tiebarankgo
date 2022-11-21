package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Host      string               `json:"host"`
	Port      uint                 `json:"port"`
	Username  string               `json:"username"`
	Password  string               `json:"password"`
	Database  string               `json:"database"`
	DBHost    string               `json:"db_host"`
	DBPort    uint                 `json:"db_port"`
	RedisHost string               `json:"redis_host"`
	RedisPort string               `json:"redis_port"`
	SessionId string               `json:"session_id"`
	AsmToken  string               `json:"asm_token"`
	Timeout   int                  `json:"timeout"`
	Servers   []ServerDistribution `json:"servers"`
}

type ServerDistribution struct {
	Level  uint   `json:"level"`
	Server string `json:"server"`
}

func GetConfig() Config {
	jsonFile, err := os.Open("etc/config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	var result Config
	err = json.NewDecoder(jsonFile).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	return result
}
