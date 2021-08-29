package metadata

type MetaData struct {
	ID             string     `json:"_id"`
	Type           string     `json:"type"`
	Path           string     `json:"path"`
	EsPath         []string   `json:"espath"`
	Length         int        `json:"length"`
	PiecesCount    int        `json:"piecesCount"`
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
