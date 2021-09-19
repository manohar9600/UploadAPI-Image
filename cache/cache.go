package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/go-redis/redis"
	elasticsearch "github.com/olivere/elastic/v7"
	"github.com/spf13/viper"
)

const mappingsVideo = `
{
    "settings": {
        "number_of_shards": 1,
        "number_of_replicas": 0
    },
    "mappings": {
		"properties": {
			"postid": {
				"type": "keyword"
			},
			"part": {
				"type": "integer"
			},
			"hash": {
				"type": "keyword"
			},
			"bytes": {
				"enabled": false
			}
        }
    }
}`

var es = getElasticConnection()
var redisClient = getRedisConnection()

func SaveImageData(data string) string {
	indexName := "imagecache"
	ctx := context.Background()
	_, err := es.Index().Index(indexName).BodyJson(data).Do(ctx)
	location := ""
	if err != nil {
		log.Fatalln("Error while data indexing")
	} else {
		// location = res.Header["Location"][0]
		location = ""
	}
	return location
}

func SaveVideoData(id string, hash string, data []byte) error {
	indexName := "videocache_temp1"
	var videoPart VideoPart
	videoPart.PostId = id
	videoPart.Hash = hash
	videoPart.Bytes = string(data)
	ctx := context.Background()
	createIndex(ctx, indexName, mappingsVideo)
	// TODO: Add verification function
	if !checkPartExists(videoPart, ctx, indexName) {
		jsonString, _ := json.Marshal(videoPart)
		_, err := es.Index().Index(indexName).BodyJson(string(jsonString)).Do(ctx)
		if err != nil {
			log.Fatalln("Error while saving to elastic, error", err)
			return errors.New("esError")
		}
	}
	return nil
}

func checkPartExists(part VideoPart, ctx context.Context, indexName string) bool {
	termQuery := elasticsearch.NewTermQuery("hash", part.Hash)
	result, err := es.Count().Index(indexName).Query(termQuery).Pretty(true).Do(ctx)
	if err != nil || result == 0 {
		return false
	}
	return true
}

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

	es, err := elasticsearch.NewClient(
		elasticsearch.SetBasicAuth(configuration.Elasticsearch.Username,
			configuration.Elasticsearch.Password))
	if err != nil {
		log.Fatalf("Error creating the client: %s", err)
	}
	return es
}

func createIndex(ctx context.Context, indexName string, mappings string) {
	exists, err := es.IndexExists(indexName).Do(ctx)
	if err != nil {
		log.Fatalln("Error communicating to elastic search, error: ", err)
	}
	if !exists {
		_, err := es.CreateIndex(indexName).BodyString(mappings).Do(ctx)
		if err != nil {
			log.Fatalln("Error creating elastic index, error: ", err)
		}
	}
}

// Redis functions
func getRedisConnection() *redis.Client {
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

	redisClient := redis.NewClient(&redis.Options{
		Password: configuration.Redis.Password,
	})
	log.Println("Connected to redis service")
	return redisClient
}

func GetFromCache(username string) (string, error) {
	val, err := redisClient.Get(username).Result()
	if err != nil {
		log.Println("Issue with redis service")
	}

	if len(val) == 0 {
		err = errors.New("empty result from redis")
	}
	return val, err
}

func PutIntoCache(reqBody []byte) error {
	var metadata MetadataVideo
	json.Unmarshal(reqBody, &metadata)
	err := redisClient.Set(metadata.ID, reqBody, 0).Err()
	if err != nil {
		log.Println("Failed to put data into cache", err)
	} else {
		log.Println("stored video input info, id:", metadata.ID)
	}
	return err
}
