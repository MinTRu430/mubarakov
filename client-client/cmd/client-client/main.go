package main

import (
	"log"
	"net/http"

	"client-client/internal/authclient"
)

func main() {
	service := authclient.NewService("http://localhost:8080")
	handler := authclient.NewHandler(service)

	http.HandleFunc("/start-auth", handler.StartAuth)
	http.HandleFunc("/finish-auth", handler.FinishAuth)

	go func() {
		if err := authclient.RunConsole(service); err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("Client started on :8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
}
