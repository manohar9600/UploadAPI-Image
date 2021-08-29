package cache

import (
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/spf13/viper"
)

var es = getElasticConnection()

func getElasticConnection() *elasticsearch.Client {
	viper.SetConfigFile("config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("Error reading config file, ", err)
	}
	var configuration Configuration
	err2 := viper.Unmarshal(&configuration)
	if err2 != nil {
		log.Println("Unable to decode into struct, ", err)
	}

	cfg := elasticsearch.Config{
		Username: configuration.Elasticsearch.Username,
		Password: configuration.Elasticsearch.Password,
	}
	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}
	return es
}

func SaveImageData(data string) string {
	indexName := "imagecache"
	res, _ := es.Index(indexName, strings.NewReader(data))
	location := ""
	if res.StatusCode != 201 {
		log.Fatalln("Error while data indexing")
	} else {
		location = res.Header["Location"][0]
	}
	return location
}
