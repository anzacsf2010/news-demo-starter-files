package main

import (
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
)

var temp = template.Must(template.ParseFiles("index.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	temp.Execute(w, nil)
}

func main() {
	_err := godotenv.Load()
	if _err != nil {
		log.Println("Error loading .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		return
	}
}
