package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"uploadapi/kafka"
	"uploadapi/validators"

	"github.com/go-redis/redis"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

var config = loadConfig()
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

// Redis functions
func getRedisConnection() *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Address,
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	log.Println("Connected to redis service")
	return redisClient
}

// fetches video metadata from redis cache
func getVideoMetadata(postId string) (string, error) {
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
	var metaData Request
	json.Unmarshal(reqBody, &metaData)
	val, err := redisClient.Get(metaData.PostId).Result()
	if val == "" {
		err = redisClient.Set(metaData.PostId, string(reqBody), 0).Err()
	}
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
		log.Println("stored video input info, id:", metaData.PostId)
	}
	res, _ := json.Marshal(response)
	return string(res)
}

// minio functions
func getMinioConnection() *minio.Client {
	endpoint := config.Minio.Address
	accessKeyID := config.Minio.AccessKeyID
	secretAccessKey := os.Getenv("MINIO_SECRET_KEY")
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

func UploadFile(buf *bytes.Buffer, imgReq Request) (string, Request, error) {
	minioClient := getMinioConnection()
	ctx := context.Background()
	bucketName := config.Minio.ImageBucket
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
		imgReq.FileNames = append(imgReq.FileNames, objectName)
	}
	res, _ := json.Marshal(&response)
	return string(res), imgReq, err
}

// function to save a video part in minio and verifies full upload
// using redis cache
func SaveVideoData(id string, hash string, part string, buf *bytes.Buffer) string {
	data := buf.Bytes()
	videopart := VideoPart{id, part, hash}
	// data verification
	err := verifyVideoPart(data, videopart)
	if err != nil {
		return generateErrorJson("client", "verification", err)
	}
	if !isFileAlreadyUploded(hash, id) {
		filename, err := putInMinioVideo(buf, id, part)
		if err != nil {
			return generateErrorJson("client", "minio", err)
		}
		err = markUploadCompletion(id, hash, filename)
		if err != nil {
			return generateErrorJson("client", "redis", err)
		}
	}
	completionBool := checkVideoCompleted(id)
	if completionBool {
		sendToKafka(id)
	}
	return generateSuccessJson(completionBool)
}

// func to verify uploaded video part is valid, if not valid it returns error
// checks hash with uploaded video part
// checks video part hash with metadata which uploaded previously
func verifyVideoPart(data []byte, videoPart VideoPart) error {
	// hash verification
	err := validators.ValidateData(data, videoPart.Hash)
	if err != nil {
		log.Println("Hash verification failed", videoPart.Part)
		return err
	}
	err = checkHashExists(videoPart.Hash, videoPart.PostId)
	return err
}

// checks hash present or not in metadata
func checkHashExists(hash string, postId string) error {
	// fetches metadata from redis
	meta, err := getVideoMetadata(postId)
	if err != nil {
		return err
	}
	var metadata Request
	json.Unmarshal([]byte(meta), &metadata)
	partBool := false
	for _, value := range metadata.PartHashes {
		if value == hash {
			partBool = true
			break
		}
	}
	if !partBool {
		err = errors.New("video broken, reupload")
		return err
	}
	return err
}

// function to check current uploaded video part is new or already exists
// throws error if it exists
func isFileAlreadyUploded(hash string, id string) bool {
	// fetches metadata from redis
	meta, err := getVideoMetadata(id)
	if err != nil {
		return false
	}
	var metadata Request
	json.Unmarshal([]byte(meta), &metadata)

	uploadBool := false
	for _, value := range metadata.UploadedHashes {
		if value == hash {
			uploadBool = true
			break
		}
	}
	return uploadBool
}

// func to store video part in minio
// returns file name and error if any
func putInMinioVideo(buf *bytes.Buffer, id string, part string) (string, error) {
	ctx := context.Background()
	bucket := config.Minio.VideoBucket
	contentType := http.DetectContentType(buf.Bytes())
	objectName := id + "_" + part + "." + strings.Split(contentType, "/")[1]

	// TODO: write logic to avoid duplicates
	info, err := minioClient.PutObject(ctx, bucket, objectName,
		buf, int64(buf.Len()), minio.PutObjectOptions{ContentType: contentType})
	if err == nil {
		log.Printf("Successfully uploaded %s of size %d\n", objectName, info.Size)
	}
	return objectName, err
}

// this function marks current part of video as completed
// updates corresponding data in redis
func markUploadCompletion(id string, hash string, filename string) error {
	// fetches metadata from redis
	meta, err := getVideoMetadata(id)
	if err != nil {
		return err
	}
	var metadata Request
	json.Unmarshal([]byte(meta), &metadata)
	metadata.UploadedHashes = append(metadata.UploadedHashes, hash)
	metadata.FileNames = append(metadata.FileNames, filename)
	strJson, _ := json.Marshal(metadata)
	err = redisClient.Set(id, string(strJson), 0).Err()
	return err
}

// function to check all parts of video is uploaded to server or not
func checkVideoCompleted(id string) bool {
	meta, err := getVideoMetadata(id)
	if err != nil {
		return false
	}
	var metadata Request
	json.Unmarshal([]byte(meta), &metadata)
	if len(metadata.PartHashes) > len(metadata.UploadedHashes) {
		return false
	}
	completionBool := true
	for _, partHash := range metadata.PartHashes {
		partBool := false
		for _, storedHash := range metadata.UploadedHashes {
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
	return completionBool
}

func sendToKafka(id string) error {
	meta, err := getVideoMetadata(id)
	if err != nil {
		return err
	}
	var metadata Request
	json.Unmarshal([]byte(meta), &metadata)
	metadata.Type = "video"
	_meta, _ := json.Marshal(metadata)
	meta = string(_meta)
	err2 := kafka.ProduceToKafka(meta)
	if err2 == nil {
		fmt.Println("sent kafka req, id:", metadata.PostId)
	}
	return err2
}

// helper functions

// generate jsons that contains proper errors faced by server
func generateErrorJson(side string, tag string, err error) string {
	var response Response
	response.Result = false
	response.Completed = false
	reserr := Errors{"client", "verification", err.Error()}
	response.Errors = append(response.Errors, reserr)
	res, _ := json.Marshal(&response)
	return string(res)
}

// generates successfull upload json,
// completed is True when video or image is fully uploaded
func generateSuccessJson(completionBool bool) string {
	var response Response
	response.Result = true
	response.Completed = completionBool
	res, _ := json.Marshal(&response)
	return string(res)
}
