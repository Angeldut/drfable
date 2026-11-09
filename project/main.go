package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"lr2/internal/auth"
	"lr2/internal/handlers"
	"lr2/internal/repository"

	_ "modernc.org/sqlite"
)

const (
	dbFile        = "expenses.db"
	migrationsDir = "migrations"
)

func runMigrations(db *sql.DB, dir string) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		name       TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, f := range files {
		name := filepath.Base(f)

		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name = ?`, name).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			continue
		}

		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("%s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (name) VALUES (?)`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		log.Printf("применена миграция %s", name)
	}
	return nil
}

func main() {
	dsn := "file:" + dbFile + "?_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	if err := runMigrations(db, migrationsDir); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	repo := repository.NewExpenseRepository(db)
	categories := repository.NewCategoryRepository(db)
	users := repository.NewUserRepository(db)
	sessions := repository.NewSessionRepository(db)

	h, err := handlers.New(repo, categories, users, sessions, "web/templates")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			if err := sessions.DeleteExpired(context.Background()); err != nil {
				log.Printf("cleanup sessions: %v", err)
			}
			time.Sleep(time.Hour)
		}
	}()

	protect := func(f http.HandlerFunc) http.Handler { return auth.Require(sessions, f) }

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.Home)
	mux.HandleFunc("GET /register", h.RegisterForm)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("GET /login", h.LoginForm)
	mux.HandleFunc("POST /login", h.Login)
	mux.HandleFunc("POST /logout", h.Logout)
	mux.Handle("GET /expenses", protect(h.List))
	mux.Handle("GET /expenses/new", protect(h.NewForm))
	mux.Handle("POST /expenses", protect(h.Create))
	mux.Handle("GET /expenses/{id}/edit", protect(h.EditForm))
	mux.Handle("POST /expenses/{id}/edit", protect(h.Update))
	mux.Handle("POST /expenses/{id}/delete", protect(h.Delete))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	addr := ":8080"
	log.Printf("Сервер запущен: http://localhost%s (база: %s)", addr, dbFile)
	log.Fatal(http.ListenAndServe(addr, mux))
}
