package app

type MetaData struct {
	ID             string     `json:"_id"`
	Type           string     `json:"type"`
	Path           string     `json:"path"`
	Owner          string     `json:"owner"`
	Caption        string     `json:"caption"`
	GeoID          string     `json:"geoId"`
	Likes          int        `json:"likes"`
	LikedBy        []LikedBy  `json:"likedBy"`
	Comments       []Comments `json:"comments"`
	PostedTime     string     `json:"postedTime"`
	LastEditedTime string     `json:"lastEditedTime"`
	UserTags       string     `json:"userTags"`
	ModelTags      string     `json:"modelTags"`
	Active         bool       `json:"active"`
}

type LikedBy struct {
	User      string `json:"user"`
	LikedTime string `json:"likedTime"`
}

type Comments struct {
	User           string `json:"user"`
	Comment        string `json:"comment"`
	Edited         bool   `json:"edited"`
	PostedTime     string `json:"postedTime"`
	LastEditedTime string `json:"lastEditedTime"`
}

type Response struct {
	ID        string   `json:"id"`
	Result    bool     `json:"result"`
	Completed bool     `json:"completed"`
	Errors    []Errors `json:"errors"`
}

type Errors struct {
	Side    string `json:"side"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}
