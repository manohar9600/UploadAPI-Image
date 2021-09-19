package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"uploadapi/cache"
	"uploadapi/metadata"

	"github.com/gorilla/mux"
)

func responseHandlerImage(w http.ResponseWriter, r *http.Request) {
	log.Println("Response Recevied! - Image")
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Data is not proper")
	}
	location := cache.SaveImageData(string(reqBody))
	metadata, err := metadata.GetMetadataJson(reqBody, location)
	resultJson := ""
	if err != nil {
		resultJson = err.Error()
	} else {
		resultJson = "success"
		fmt.Println(metadata.EsPath, metadata.PostedTime)
	}
	fmt.Fprint(w, resultJson)
}

func responseHandlerVideo(w http.ResponseWriter, r *http.Request) {
	log.Println("Response Recevied! - Video part")
	err := r.ParseMultipartForm(100 * 1024)
	if err != nil {
		log.Fatalln("Bad request", "error", err)
	}

	id := r.FormValue("_id")
	hash := r.FormValue("hash")
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Fatalln("Bad request", "error", err)
	}
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		log.Fatalln("Failed to convert to bytes", "error", err)
	}
	cache.SaveVideoData(id, hash, buf.Bytes())
}

func responseHandlerMetadata(w http.ResponseWriter, r *http.Request) {
	log.Println("Response Recevied! - Video metadata")
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Data is not proper")
	}
	err = cache.PutIntoCache(reqBody)
	resultJson := ""
	if err != nil {
		resultJson = err.Error()
	} else {
		resultJson = "success"
	}
	fmt.Fprint(w, resultJson)
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
