package app

type Config struct {
	Redis struct {
		Address string `yaml:"address"`
	} `yaml:"redis"`
	Minio struct {
		Address     string `yaml:"address"`
		AccessKeyID string `yaml:"accessKeyId"`
		ImageBucket string `yaml:"imageBucket"`
		VideoBucket string `yaml:"videoBucket"`
	} `yaml:"minio"`
}

type VideoPart struct {
	PostId string `json:"postId"`
	Part   string `json:"part"`
	Hash   string `json:"hash"`
}

type Request struct {
	PostId         string   `json:"postId"`
	Type           string   `json:"type"`
	User           string   `json:"user"`
	Hash           string   `json:"hash"`
	Caption        string   `json:"caption"`
	PostedTime     string   `json:"postedTime"`
	FileNames      []string `json:"fileNames"`
	Parts          int      `json:"parts"`
	PartHashes     []string `json:"partHashes"`
	UploadedHashes []string `json:"uploadedHashes"`
}
