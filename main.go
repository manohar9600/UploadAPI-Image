package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"uploadapi/app"
	"uploadapi/kafka"

	"github.com/gorilla/mux"
)

type KafkaRequest struct {
	ID string `json:"id"`
}

func responseHandlerImage(w http.ResponseWriter, r *http.Request) {
	log.Println("Response Recevied! - Image")
	res, err := processImageRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())

	} else {
		fmt.Fprint(w, res)
	}
}

func processImageRequest(r *http.Request) (string, error) {
	err := r.ParseMultipartForm(100 * 1024)
	res := ""
	if err != nil {
		res = "Bad request"
		log.Println(res, "error", err)
		err = errors.New(res + ", err:" + err.Error())
		return res, err
	}

	file, _, err := r.FormFile("file")
	properties := r.FormValue("properties")
	if err != nil {
		res = "Bad request"
		log.Println(res, "error", err)
		err = errors.New(res + ", err:" + err.Error())
		return res, err
	}
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		res = "Failed to convert to bytes"
		log.Println(res, "error", err)
		err = errors.New(res + ", err:" + err.Error())
		return res, err
	}

	var imageRequest app.ImageRequest
	err = json.Unmarshal([]byte(properties), &imageRequest)
	if err != nil {
		res = "Bad request"
		log.Println(res, "error", err)
		err = errors.New(res + ", err:" + err.Error())
		return res, err
	}

	res = app.UploadFile(buf, imageRequest)
	var kafkaReq KafkaRequest
	kafkaReq.ID = imageRequest.PostId
	msgString, err := json.Marshal(kafkaReq)
	err2 := kafka.ProduceToKafka(string(msgString))
	if err2 == nil {
		fmt.Println("sent kafka req, id:", kafkaReq.ID)
	}
	return res, err
}

func responseHandlerVideo(w http.ResponseWriter, r *http.Request) {
	log.Println("Response Recevied! - video part")
	res, err := processVideoRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())

	} else {
		fmt.Fprint(w, res)
	}
}

func processVideoRequest(r *http.Request) (string, error) {
	err := r.ParseMultipartForm(100 * 1024)
	res := ""
	if err != nil {
		res = "Bad request"
		log.Println(res, "error", err)
		err = errors.New(res + ", err:" + err.Error())
		return res, err
	}

	id := r.FormValue("postId")
	part := r.FormValue("part")
	log.Println("video part postid:", id, "part:", part)
	hash := r.FormValue("hash")
	file, _, err := r.FormFile("file")
	if err != nil {
		res = "Bad request"
		log.Println(res, "error", err)
		err = errors.New(res + ", err:" + err.Error())
		return res, err
	}
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		res = "Failed to convert to bytes"
		log.Println(res, "error", err)
		err = errors.New(res + ", err:" + err.Error())
		return res, err
	}
	res = app.SaveVideoData(id, hash, part, buf.Bytes())
	return res, err
}

func responseHandlerMetadata(w http.ResponseWriter, r *http.Request) {
	log.Println("Response Recevied! - Video metadata")
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Data is not proper")
	}
	response := app.PutIntoCache(reqBody)
	fmt.Fprint(w, response)
}

func main() {
	port := ":8002"
	router := mux.NewRouter().StrictSlash(true)

	extension := "/uploadImage"
	router.HandleFunc(extension, responseHandlerImage)
	fmt.Printf("Listening on localhost%s%s\n", port, extension)

	videoMetaExtension := "/uploadMetaVideo"
	router.HandleFunc(videoMetaExtension, responseHandlerMetadata)
	fmt.Printf("Listening on localhost%s%s\n", port, videoMetaExtension)

	videoExtension := "/uploadVideo"
	router.HandleFunc(videoExtension, responseHandlerVideo)
	fmt.Printf("Listening on localhost%s%s\n", port, videoExtension)

	http.ListenAndServe(port, router)
}
