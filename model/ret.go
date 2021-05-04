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
