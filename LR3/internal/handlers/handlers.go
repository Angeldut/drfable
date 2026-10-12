package handlers

import (
	"bytes"
	"errors"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lr2/internal/models"
	"lr2/internal/repository"
)

const dateLayout = "2006-01-02" 

type Handler struct {
	repo      *repository.ExpenseRepository
	templates map[string]*template.Template
}

type pageData struct {
	Title    string
	Expenses []models.Expense
	Total    float64
	Error    string
	ID          int64
	Amount      string
	Description string
	Date        string
}

func New(repo *repository.ExpenseRepository, templatesDir string) (*Handler, error) {
	layout := filepath.Join(templatesDir, "layout.html")
	h := &Handler{repo: repo, templates: make(map[string]*template.Template)}

	for _, page := range []string{"expenses.html", "add.html", "edit.html"} {
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
	ctx := r.Context()

	expenses, err := h.repo.GetAll(ctx)
	if err != nil {
		log.Printf("list expenses: %v", err)
		http.Error(w, "Не удалось загрузить траты", http.StatusInternalServerError)
		return
	}
	total, err := h.repo.Total(ctx)
	if err != nil {
		log.Printf("total expenses: %v", err)
		http.Error(w, "Не удалось посчитать итог", http.StatusInternalServerError)
		return
	}

	h.render(w, http.StatusOK, "expenses.html", pageData{
		Title:    "Список трат",
		Expenses: expenses,
		Total:    total,
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

	amount, date, ok := parseAmountAndDate(amountStr, dateStr, fail)
	if !ok {
		return
	}
	if description == "" {
		fail("Введите описание")
		return
	}

	if _, err := h.repo.Create(r.Context(), amount, description, date); err != nil {
		log.Printf("create expense: %v", err)
		fail("Не удалось сохранить трату")
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) EditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	expense, err := h.repo.GetByID(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("get expense %d: %v", id, err)
		http.Error(w, "Не удалось загрузить трату", http.StatusInternalServerError)
		return
	}

	h.render(w, http.StatusOK, "edit.html", pageData{
		Title:       "Редактировать трату",
		ID:          expense.ID,
		Amount:      strconv.FormatFloat(expense.Amount, 'f', 2, 64),
		Description: expense.Description,
		Date:        expense.Date.Format(dateLayout),
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	amountStr := strings.TrimSpace(r.FormValue("amount"))
	description := strings.TrimSpace(r.FormValue("description"))
	dateStr := r.FormValue("date")

	fail := func(msg string) {
		h.render(w, http.StatusUnprocessableEntity, "edit.html", pageData{
			Title: "Редактировать трату", Error: msg,
			ID: id, Amount: amountStr, Description: description, Date: dateStr,
		})
	}

	amount, date, ok := parseAmountAndDate(amountStr, dateStr, fail)
	if !ok {
		return
	}
	if description == "" {
		fail("Введите описание")
		return
	}

	err = h.repo.Update(r.Context(), id, amount, description, date)
	if errors.Is(err, repository.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("update expense %d: %v", id, err)
		fail("Не удалось сохранить изменения")
		return
	}

	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = h.repo.Delete(r.Context(), id)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		log.Printf("delete expense %d: %v", id, err)
		http.Error(w, "Не удалось удалить трату", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}
func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
func parseAmountAndDate(amountStr, dateStr string, fail func(string)) (amount float64, date time.Time, ok bool) {
	amount, err := strconv.ParseFloat(strings.ReplaceAll(amountStr, ",", "."), 64)
	if err != nil || amount <= 0 {
		fail("Сумма должна быть положительным числом")
		return 0, time.Time{}, false
	}
	date, err = time.Parse(dateLayout, dateStr)
	if err != nil {
		fail("Укажите корректную дату")
		return 0, time.Time{}, false
	}
	return amount, date, true
}
