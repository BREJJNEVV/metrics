package main

import (
	"log"
	"net/http"

	"github.com/BREJJNEVV/metrics/internal/handler/update"
)

func main() {

	router := http.NewServeMux()
	upd := update.CreateUpdateHandler()
	router.HandleFunc("/update/", upd.Update)

	srv := http.Server{
		Handler: router,
		Addr:    ":8080",
	}
	log.Println("Server started at :8080")
	log.Fatal(srv.ListenAndServe())
}
