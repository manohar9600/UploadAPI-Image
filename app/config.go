package app

type Config struct {
	Elasticsearch struct {
		Username   string `yaml:"username"`
		Password   string `yaml:"password"`
		IndexImage string `yaml:"indexImage"`
		IndexMeta  string `yaml:"indexMeta"`
		IndexVideo string `yaml:"indexVideo"`
	} `yaml:"elasticsearch"`
	Redis struct {
		Address  string `yaml:"address"`
		Password string `yaml:"password"`
	} `yaml:"redis"`
}

type MetadataVideo struct {
	ID         string   `json:"_id"`
	Parts      int      `json:"parts"`
	PartHashes []string `json:"partHashes"`
}

type VideoPart struct {
	PostId string `json:"postId"`
	Part   int    `json:"part"`
	Hash   string `json:"hash"`
	Bytes  string `json:"bytes"`
}

type ImageRequest struct {
	PostId string `json:"postId"`
	Hash   string `json:"hash"`
	Bytes  string `json:"bytes"`
}
