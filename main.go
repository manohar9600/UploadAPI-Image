package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"uploadapi/cache"
	"uploadapi/metadata"

	"github.com/gorilla/mux"
)

func responseHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Response Recevied!")
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

func main() {
	port := ":8002"
	extension := "/uploadImage"
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc(extension, responseHandler)
	fmt.Printf("Listening on localhost%s%s\n", port, extension)
	http.ListenAndServe(port, router)
}
