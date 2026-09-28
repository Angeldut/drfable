package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lr2/internal/models"
)

const dateLayout = "2006-01-02"

type Handler struct {
	store     *models.Store
	templates map[string]*template.Template
}

type pageData struct {
	Title    string
	Expenses []models.Expense
	Total    float64
	Error    string
	Amount      string
	Description string
	Date        string
}

func New(store *models.Store, templatesDir string) (*Handler, error) {
	layout := filepath.Join(templatesDir, "layout.html")
	h := &Handler{store: store, templates: make(map[string]*template.Template)}

	for _, page := range []string{"expenses.html", "add.html"} {
		t, err := template.ParseFiles(layout, filepath.Join(templatesDir, page))
		if err != nil {
			return nil, err
		}
		h.templates[page] = t
	}
	return h, nil
}

func (h *Handler) render(w http.ResponseWriter, status int, page string, data pageData) {
	var buf bytes.Buffer
	if err := h.templates[page].ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("render %s: %v", page, err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "expenses.html", pageData{
		Title:    "Список трат",
		Expenses: h.store.All(),
		Total:    h.store.Total(),
	})
}

func (h *Handler) NewForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "add.html", pageData{
		Title: "Добавить трату",
		Date:  time.Now().Format(dateLayout),
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	amountStr := strings.TrimSpace(r.FormValue("amount"))
	description := strings.TrimSpace(r.FormValue("description"))
	dateStr := r.FormValue("date")

	fail := func(msg string) {
		h.render(w, http.StatusUnprocessableEntity, "add.html", pageData{
			Title: "Добавить трату", Error: msg,
			Amount: amountStr, Description: description, Date: dateStr,
		})
	}

	amount, err := strconv.ParseFloat(strings.ReplaceAll(amountStr, ",", "."), 64)
	if err != nil || amount <= 0 {
		fail("Сумма должна быть положительным числом")
		return
	}
	if description == "" {
		fail("Введите описание")
		return
	}
	date, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		fail("Укажите корректную дату")
		return
	}

	h.store.Add(amount, description, date)

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}
