package model

type UserLink struct {
	Token string `json:"token" xml:"token"`
	Link  string `json:"link" xml:"link"`
}

type RankInfo struct {
	Token string `json:"token" xml:"token"`
	Rank  uint   `json:"rank" xml:"rank"`
	Level uint   `json:"level" xml:"level"`
}
