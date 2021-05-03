package model

type TiebaUser struct {
	Rank     uint64 `json:"rank"`
	Name     string `json:"name"`
	Link     string `json:"link"`
	Level    uint64 `json:"level"`
	Exp      uint64 `json:"exp"`
	Member   bool   `json:"member"`
	Nickname string `json:"nickname"`
}
