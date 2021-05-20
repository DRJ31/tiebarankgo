package model

import (
	"fmt"
	"github.com/DRJ31/tiebarankgo/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

type User struct {
	Id       uint   `json:"id"`
	Rank     uint   `json:"rank"`
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Link     string `json:"link"`
	Level    uint   `json:"level"`
	Exp      uint   `json:"exp"`
	Member   bool   `json:"member"`
}

type Anniversary struct {
	Id          uint   `json:"id"`
	Date        string `json:"date"`
	Event       string `json:"event"`
	Adj         string `json:"adj"`
	Description string `json:"description"`
}

type Event struct {
	Id    uint      `json:"id"`
	Date  time.Time `json:"date"`
	Event string    `json:"event"`
}

type Post struct {
	Id        uint      `json:"id"`
	Date      time.Time `json:"date"`
	Total     uint      `json:"total"`
	Followers uint      `json:"followers"`
	Members   uint      `json:"members"`
	Vip       uint      `json:"vip"`
	Signin    uint      `json:"signin"`
}

type History struct {
	Id           uint      `json:"id"`
	Date         time.Time `json:"date"`
	Distribution string    `json:"distribution"`
}

type Divider struct {
	Id    uint `json:"id"`
	Level uint `json:"level"`
	Rank  uint `json:"rank"`
}

type UpIncome struct {
	Id     uint      `json:"id"`
	Name   string    `json:"name"`
	Date   time.Time `json:"date"`
	Income uint      `json:"income"`
	Max    uint      `json:"max"`
}

func (User) TableName() string {
	return "user"
}

func (Anniversary) TableName() string {
	return "anniversary"
}

func (Event) TableName() string {
	return "event"
}

func (Post) TableName() string {
	return "post"
}

func (History) TableName() string {
	return "history"
}

func (Divider) TableName() string {
	return "divider"
}

func (UpIncome) TableName() string {
	return "income"
}

func Init() (*gorm.DB, error) {
	formatStr := "%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=True&loc=Local"
	cf := config.GetConfig()
	dsn := fmt.Sprintf(formatStr, cf.Username, cf.Password, cf.DBHost, cf.DBPort, cf.Database)
	Conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return Conn, nil
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
