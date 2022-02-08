package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
	"uploadapi/app"
	"uploadapi/kafka"

	"github.com/gorilla/mux"
)

var name = "UploadAPI" // name used for ping test

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

// function that process image request, returns error when there is
// error in input data.
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

	var imageRequest app.Request
	err = json.Unmarshal([]byte(properties), &imageRequest)
	if err != nil {
		res = "Bad request"
		log.Println(res, "error", err)
		err = errors.New(res + ", err:" + err.Error())
		return res, err
	}

	res, imageRequest, err = app.UploadFile(buf, imageRequest)
	if err != nil {
		return res, nil
	}

	msgString, err := json.Marshal(imageRequest)
	err2 := kafka.ProduceToKafka(string(msgString))
	if err2 == nil {
		fmt.Println("sent kafka req, id:", imageRequest.PostId)
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
	res = app.SaveVideoData(id, hash, part, buf)
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

func pingHandler(w http.ResponseWriter, req *http.Request) {
	u, _ := url.Parse(req.URL.String())
	wait := u.Query().Get("wait")
	if len(wait) > 0 {
		duration, err := time.ParseDuration(wait)
		if err == nil {
			time.Sleep(duration)
		}
	}

	if name != "" {
		_, _ = fmt.Fprintln(w, "Name:", name)
	}

	hostname, _ := os.Hostname()
	_, _ = fmt.Fprintln(w, "Hostname:", hostname)

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			_, _ = fmt.Fprintln(w, "IP:", ip)
		}
	}

	_, _ = fmt.Fprintln(w, "RemoteAddr:", req.RemoteAddr)
	if err := req.Write(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	port := ":8002"
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", pingHandler)

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
