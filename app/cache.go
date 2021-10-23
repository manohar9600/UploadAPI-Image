package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"uploadapi/validators"

	"github.com/go-redis/redis"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

// TODO: mappings for image elastic search
var config = loadConfig()
var es = getElasticConnection()
var redisClient = getRedisConnection()
var minioClient = getMinioConnection()

func loadConfig() Config {
	viper.SetConfigFile("config.yaml")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalln("Error reading config file, ", err)
	}
	var config Config
	err2 := viper.Unmarshal(&config)
	if err2 != nil {
		log.Fatalln("Unable to decode into struct, ", err)
	}
	return config
}

func SaveVideoData(id string, hash string, partStr string, data []byte) string {
	var response Response
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
		var resError Errors
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
		var resError Errors
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

func saveInElastic(videoPart VideoPart, response Response) Response {
	ctx := context.Background()
	createIndex(ctx, config.Elasticsearch.IndexVideo, mappingsVideo)
	if !checkPartExists(videoPart, ctx) {
		jsonString, _ := json.Marshal(videoPart)
		_, err := es.Index().Index(
			config.Elasticsearch.IndexVideo).BodyJson(string(jsonString)).Do(ctx)
		if err != nil {
			var resError Errors
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

func checkPartExists(part VideoPart, ctx context.Context) bool {
	termQuery := elasticsearch.NewMatchQuery("postId", part.PostId)
	termQuery2 := elasticsearch.NewMatchQuery("hash", part.Hash)
	query := elasticsearch.NewBoolQuery().Must(termQuery).Filter(termQuery2)
	result, err := es.Count().Index(config.Elasticsearch.IndexVideo).Query(
		query).Pretty(true).Do(ctx)
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
	partsCount := getDocumentsCount(videoPart.PostId, ctx)
	if partsCount < metadata.Parts {
		return false, nil
	}

	storedHashes := getDocumentHashes(videoPart.PostId, ctx)
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

func getDocumentsCount(postId string, ctx context.Context) int {
	termQuery := elasticsearch.NewTermQuery("postId", postId)
	result, err := es.Count().Index(config.Elasticsearch.IndexVideo).Query(
		termQuery).Pretty(true).Do(ctx)
	if err != nil {
		result = 0
	}
	resultInt := int(result)
	return resultInt
}

func getDocumentHashes(postId string, ctx context.Context) []string {
	var hashes []string
	termQuery := elasticsearch.NewTermQuery("postId", postId)
	scroller := es.Scroll().Index(
		config.Elasticsearch.IndexVideo).Query(termQuery).Size(1)
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
	es, err := elasticsearch.NewClient(
		elasticsearch.SetBasicAuth(config.Elasticsearch.Username,
			config.Elasticsearch.Password))
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
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Address,
		Password: config.Redis.Password,
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
	var response Response
	var metaData MetadataVideo
	json.Unmarshal(reqBody, &metaData)
	err := redisClient.Set(metaData.ID, string(reqBody), 0).Err()
	if err != nil {
		var resError Errors
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

// minio functions
func getMinioConnection() *minio.Client {
	endpoint := config.Minio.Address
	accessKeyID := config.Minio.AccessKeyID
	secretAccessKey := config.Minio.SecretKey
	// useSSL := true

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		// Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}
	return minioClient
}

func UploadFile(buf *bytes.Buffer, imgReq ImageRequest) string {
	ctx := context.Background()
	bucketName := "imagecache"
	contentType := http.DetectContentType(buf.Bytes())
	objectName := imgReq.PostId + "." + strings.Split(contentType, "/")[1]

	info, err := minioClient.PutObject(ctx, bucketName, objectName,
		buf, int64(buf.Len()), minio.PutObjectOptions{ContentType: contentType})
	var response Response
	if err != nil {
		var resError Errors
		resError.Side = "server"
		resError.Tag = "minio"
		resError.Message = err.Error()
		response.Result = false
		response.Completed = false
		response.Errors = append(response.Errors, resError)
		log.Println("Error while stroring data in minio")
	} else {
		log.Printf("Successfully uploaded %s of size %d\n", objectName, info.Size)
		response.Result = true
		response.Completed = true
	}
	res, _ := json.Marshal(&response)
	return string(res)
}
