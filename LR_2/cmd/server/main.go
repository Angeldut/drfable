package main

import (
	"log"
	"net/http"

	"lr2/internal/handlers"
	"lr2/internal/models"
)

func main() {
	store := models.NewStore()

	h, err := handlers.New(store, "web/templates")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /expenses", h.List)
	mux.HandleFunc("GET /expenses/new", h.NewForm)
	mux.HandleFunc("POST /expenses", h.Create)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	addr := ":8080"
	log.Printf("Сервер запущен: http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}