package model

type TiebaUser struct {
	Rank     uint   `json:"rank"`
	Name     string `json:"name"`
	Link     string `json:"link"`
	Level    uint   `json:"level"`
	Exp      uint   `json:"exp"`
	Member   bool   `json:"member"`
	Nickname string `json:"nickname"`
}

type UserAvatar struct {
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}

type EventRet struct {
	Event string `json:"event"`
	Date  string `json:"date"`
}

type PostRet struct {
	Date  string `json:"date"`
	Total uint   `json:"total"`
}

type DistRet struct {
	Level uint `json:"level"`
	Rank  uint `json:"rank"`
	Delta int  `json:"delta"`
}

type DistInfo struct {
	Level uint `json:"level"`
	Rank  uint `json:"rank"`
}

type Income struct {
	Date    uint `json:"date"`
	Income  uint `json:"income"`
	Average uint `json:"average"`
}

type MonthIncome struct {
	Date   string `json:"date"`
	Income uint   `json:"income"`
}

type WallpaperImage struct {
	Startdate     string        `json:"startdate"`
	Fullstartdate string        `json:"fullstartdate"`
	Enddate       string        `json:"enddate"`
	Url           string        `json:"url"`
	Urlbase       string        `json:"urlbase"`
	Copyright     string        `json:"copyright"`
	Copyrightlink string        `json:"copyrightlink"`
	Title         string        `json:"title"`
	Quiz          string        `json:"quiz"`
	Wp            bool          `json:"wp"`
	Hsh           string        `json:"hsh"`
	Drk           int           `json:"drk"`
	Top           int           `json:"top"`
	Bot           int           `json:"bot"`
	Hs            []interface{} `json:"hs"`
}

type WallpaperRet struct {
	Images []WallpaperImage `json:"images"`
}
