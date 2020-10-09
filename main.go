package main

import (
	"net/http"
	"os"
	"fmt"
	// "strings"
	"encoding/json"
	"log"
	"time"
	"github.com/gorilla/mux"
	// "gopkg.in/gographics/imagick.v2/imagick"
)

type TestStruct struct {
	returnType string
}

func buildJsonResponse(w http.ResponseWriter, code int, payload TestStruct){
	resp, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(resp)
}

func handleData(w http.ResponseWriter, r *http.Request){
	buildJsonResponse(w, http.StatusOK, TestStruct{returnType:"RAW"})
}

func handleImage(w http.ResponseWriter, r *http.Request){
	buildJsonResponse(w, http.StatusOK, TestStruct{returnType:"IMAGE"})
}

func main(){
	r := mux.NewRouter()
	r.HandleFunc("/aqi", handleData)
	r.HandleFunc("/image.png", handleImage)

	addr := fmt.Sprintf(":%q", os.Getenv("PORT"))
	srv := &http.Server{
		Handler: r,
		Addr: addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout: 15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}