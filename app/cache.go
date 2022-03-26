package app

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

var config = loadConfig()

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

// minio functions
func getMinioConnection() *minio.Client {
	endpoint := config.Minio.Address
	accessKeyID := os.Getenv("MINIO_ACCESS_KEY")
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
		log.Println("Error while storing data in minio")
	} else {
		log.Printf("Successfully uploaded %s of size %d\n", objectName, info.Size)
		response.Result = true
		response.Completed = true
		imgReq.FileName = objectName
	}
	res, _ := json.Marshal(&response)
	return string(res), imgReq, err
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
