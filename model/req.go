package model

import "time"

type UserLink struct {
	Token string `json:"token" xml:"token"`
	Link  string `json:"link" xml:"link"`
}

type RankInfo struct {
	Token string `json:"token" xml:"token"`
	Rank  uint   `json:"rank" xml:"rank"`
	Level uint   `json:"level" xml:"level"`
}

type PostInfo struct {
	Token     string `json:"token" xml:"token"`
	Followers uint   `json:"followers" xml:"followers"`
	Total     uint   `json:"total" xml:"total"`
	Signin    uint   `json:"signin" xml:"signin"`
}

type IncomeData struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Points []struct {
			Name      string   `json:"name"`
			Data      [][]uint `json:"data"`
			Isdefault int      `json:"isDefault"`
			Total     int      `json:"total"`
			Dashstyle string   `json:"dashStyle,omitempty"`
			Marker    struct {
				Enabled bool `json:"enabled"`
			} `json:"marker,omitempty"`
		} `json:"points"`
		Versions []struct {
			Releasedate    time.Time `json:"releaseDate"`
			Releasetime    int       `json:"releaseTime"`
			Versionstring  string    `json:"versionString"`
			Displaydate    string    `json:"displayDate"`
			Orgreleasetime int       `json:"orgReleaseTime"`
		} `json:"versions"`
	} `json:"data"`
}
