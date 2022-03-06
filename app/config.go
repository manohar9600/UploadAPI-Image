package app

type Config struct {
	Redis struct {
		Address string `yaml:"address"`
	} `yaml:"redis"`
	Minio struct {
		Address     string `yaml:"address"`
		AccessKeyID string `yaml:"accessKeyId"`
		ImageBucket string `yaml:"imageBucket"`
	} `yaml:"minio"`
}

type Request struct {
	PostId     string `json:"postId"`
	Type       string `json:"type"`
	User       string `json:"user"`
	Hash       string `json:"hash"`
	Caption    string `json:"caption"`
	PostedTime string `json:"postedTime"`
	FileName   string `json:"fileNames"`
}
