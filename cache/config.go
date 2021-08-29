package cache

type Configuration struct {
	Elasticsearch struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"elasticsearch"`
}
