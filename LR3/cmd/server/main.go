package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"lr2/internal/handlers"
	"lr2/internal/repository"

	_ "modernc.org/sqlite" 
)

const (
	dbFile        = "expenses.db"
	migrationFile = "migrations/0001_create_expenses.sql"
)
func runMigration(db *sql.DB, path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(sqlBytes))
	return err
}

func main() {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	if err := runMigration(db, migrationFile); err != nil {
		log.Fatalf("run migration: %v", err)
	}

	repo := repository.NewExpenseRepository(db)

	h, err := handlers.New(repo, "web/templates")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /expenses", h.List)
	mux.HandleFunc("GET /expenses/new", h.NewForm)
	mux.HandleFunc("POST /expenses", h.Create)
	mux.HandleFunc("GET /expenses/{id}/edit", h.EditForm)
	mux.HandleFunc("POST /expenses/{id}/edit", h.Update)
	mux.HandleFunc("POST /expenses/{id}/delete", h.Delete)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	addr := ":8080"
	log.Printf("Сервер запущен: http://localhost%s (база: %s)", addr, dbFile)
	log.Fatal(http.ListenAndServe(addr, mux))
}
