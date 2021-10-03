package cache

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strconv"
	"uploadapi/metadata"
	"uploadapi/validators"

	"github.com/go-redis/redis"
	elasticsearch "github.com/olivere/elastic/v7"
	"github.com/spf13/viper"
)

const videoIndex = "videocache_temp1"
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

// TODO: mappings for image elastic search

var es = getElasticConnection()
var redisClient = getRedisConnection()

func SaveImageData(data string) (string, string) {
	indexName := "imagecache"
	ctx := context.Background()
	var response metadata.Response
	// TODO: Add verification
	_, err := es.Index().Index(indexName).BodyJson(data).Do(ctx)
	location := ""
	if err != nil {
		var resError metadata.Errors
		resError.Side = "server"
		resError.Tag = "elastic"
		resError.Message = err.Error()
		response.Result = false
		response.Completed = false
		response.Errors = append(response.Errors, resError)
		log.Println("Error while data indexing in elastic")
	} else {
		// location = res.Header["Location"][0]
		response.Result = true
		response.Completed = true
		location = ""
	}
	res, _ := json.Marshal(&response)
	return location, string(res)
}

func SaveVideoData(id string, hash string, partStr string, data []byte) string {
	var response metadata.Response
	response.Result = false
	response.Completed = false

	var videoPart VideoPart
	videoPart.PostId = id
	videoPart.Hash = hash
	videoPart.Bytes = string(data)
	part, _ := strconv.Atoi(partStr)
	videoPart.Part = part

	// data verification
	err := verifyVideoPart(videoPart)
	if err != nil {
		var resError metadata.Errors
		resError.Side = "client"
		resError.Tag = "verification"
		resError.Message = err.Error()
		response.Errors = append(response.Errors, resError)
		res, _ := json.Marshal(&response)
		return string(res)
	}

	response = saveInElastic(videoPart, response)

	uploadBool, err := isPostFullyUploaded(videoPart)
	if err != nil {
		var resError metadata.Errors
		resError.Side = "server"
		resError.Tag = "elastic"
		resError.Message = err.Error()
		response.Errors = append(response.Errors, resError)
	}
	if uploadBool {
		response.Result = true
		response.Completed = true
	}
	res, _ := json.Marshal(&response)
	return string(res)
}

func saveInElastic(videoPart VideoPart, response metadata.Response) metadata.Response {
	ctx := context.Background()
	createIndex(ctx, videoIndex, mappingsVideo)
	if !checkPartExists(videoPart, ctx, videoIndex) {
		jsonString, _ := json.Marshal(videoPart)
		_, err := es.Index().Index(videoIndex).BodyJson(string(jsonString)).Do(ctx)
		if err != nil {
			var resError metadata.Errors
			resError.Side = "server"
			resError.Tag = "elastic"
			resError.Message = err.Error()
			response.Errors = append(response.Errors, resError)
		} else {
			response.Result = true
		}
	}
	return response
}

func checkPartExists(part VideoPart, ctx context.Context, indexName string) bool {
	termQuery := elasticsearch.NewMatchQuery("postId", part.PostId)
	termQuery2 := elasticsearch.NewMatchQuery("hash", part.Hash)
	query := elasticsearch.NewBoolQuery().Must(termQuery).Filter(termQuery2)
	result, err := es.Count().Index(indexName).Query(query).Pretty(true).Do(ctx)
	if err != nil || result == 0 {
		return false
	}
	log.Println("part already exists, hash:", part.Hash)
	return true
}

func verifyVideoPart(videoPart VideoPart) error {
	err := validators.ValidateData(videoPart.Bytes, videoPart.Hash)
	if err != nil {
		log.Println("Hash verification failed", videoPart.Part)
		return err
	}
	data, err := GetFromCache(videoPart.PostId)
	if err != nil {
		return err
	}
	var metadata MetadataVideo
	json.Unmarshal([]byte(data), &metadata)

	partBool := false
	for _, value := range metadata.PartHashes {
		if value == videoPart.Hash {
			partBool = true
			break
		}
	}
	if !partBool {
		err = errors.New("video broken, reupload")
		return err
	} else {
		return nil
	}
}

func isPostFullyUploaded(videoPart VideoPart) (bool, error) {
	data, err := GetFromCache(videoPart.PostId)
	if err != nil {
		return false, err
	}
	var metadata MetadataVideo
	json.Unmarshal([]byte(data), &metadata)
	ctx := context.Background()
	partsCount := getDocumentsCount(videoPart.PostId, ctx, videoIndex)
	if partsCount < metadata.Parts {
		return false, nil
	}

	storedHashes := getDocumentHashes(videoPart.PostId, ctx, videoIndex)
	completionBool := true
	for _, partHash := range metadata.PartHashes {
		partBool := false
		for _, storedHash := range storedHashes {
			if partHash == storedHash {
				partBool = true
				break
			}
		}
		if !partBool {
			completionBool = false
			break
		}
	}

	return completionBool, nil
}

func getDocumentsCount(postId string, ctx context.Context, indexName string) int {
	termQuery := elasticsearch.NewTermQuery("postId", postId)
	result, err := es.Count().Index(indexName).Query(termQuery).Pretty(true).Do(ctx)
	if err != nil {
		result = 0
	}
	resultInt := int(result)
	return resultInt
}

func getDocumentHashes(postId string, ctx context.Context, indexName string) []string {
	var hashes []string
	termQuery := elasticsearch.NewTermQuery("postId", postId)
	scroller := es.Scroll().Index(indexName).Query(termQuery).Size(1)
	for {
		res, err := scroller.Do(context.TODO())
		if err == io.EOF {
			// No remaining documents matching the search so break out of the 'forever' loop
			break
		}
		for _, hit := range res.Hits.Hits {
			var part VideoPart
			json.Unmarshal(hit.Source, &part)
			hashes = append(hashes, part.Hash)
		}
	}
	return hashes
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
		log.Println("Error communicating to elastic search, error: ", err)
	}
	if !exists {
		_, err := es.CreateIndex(indexName).BodyString(mappings).Do(ctx)
		if err != nil {
			log.Println("Error creating elastic index, error: ", err)
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

func GetFromCache(postId string) (string, error) {
	val, err := redisClient.Get(postId).Result()
	if err != nil {
		log.Println("Issue with redis service")
	}

	if len(val) == 0 {
		err = errors.New("empty result from redis")
	}
	return val, err
}

func PutIntoCache(reqBody []byte) string {
	var response metadata.Response
	var metaData MetadataVideo
	json.Unmarshal(reqBody, &metaData)
	err := redisClient.Set(metaData.ID, string(reqBody), 0).Err()
	if err != nil {
		var resError metadata.Errors
		resError.Side = "server"
		resError.Tag = "redis"
		resError.Message = err.Error()
		response.Completed = false
		response.Result = false
		response.Errors = append(response.Errors, resError)
		log.Println("Error while data indexing in redis. err:", err.Error())
	} else {
		response.Result = true
		log.Println("stored video input info, id:", metaData.ID)
	}
	res, _ := json.Marshal(response)
	return string(res)
}
