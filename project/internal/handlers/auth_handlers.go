package handlers

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"lr2/internal/auth"
	"lr2/internal/repository"
)

const (
	sessionTTL = 24 * time.Hour
	minPasswordLen = 8
	maxPasswordLen = 72
)

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func validEmail(s string) bool {
	if s == "" || len(s) > 254 {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}
func (h *Handler) RegisterForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, "register.html", pageData{Title: "Регистрация"})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	fail := func(status int, msg string) {
		h.render(w, r, status, "register.html", pageData{
			Title: "Регистрация", Error: msg, FormEmail: email,
		})
	}

	switch {
	case !validEmail(email):
		fail(http.StatusUnprocessableEntity, "Введите корректный email")
		return
	case len(password) < minPasswordLen:
		fail(http.StatusUnprocessableEntity, "Пароль должен быть не короче 8 символов")
		return
	case len(password) > maxPasswordLen:
		fail(http.StatusUnprocessableEntity, "Пароль слишком длинный (максимум 72 байта)")
		return
	case password != confirm:
		fail(http.StatusUnprocessableEntity, "Пароли не совпадают")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("bcrypt: %v", err)
		fail(http.StatusInternalServerError, "Не удалось зарегистрировать пользователя")
		return
	}

	if _, err := h.users.Create(r.Context(), email, string(hash)); err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			fail(http.StatusConflict, "Пользователь с таким email уже существует")
			return
		}
		log.Printf("create user: %v", err)
		fail(http.StatusInternalServerError, "Не удалось зарегистрировать пользователя")
		return
	}

	http.Redirect(w, r, "/login?registered=1", http.StatusSeeOther)
}
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	data := pageData{Title: "Вход"}
	if r.URL.Query().Get("registered") == "1" {
		data.Info = "Регистрация прошла успешно. Теперь войдите."
	}
	h.render(w, r, http.StatusOK, "login.html", data)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Некорректная форма", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")

	fail := func(status int, msg string) {
		h.render(w, r, status, "login.html", pageData{Title: "Вход", Error: msg, FormEmail: email})
	}

	user, userErr := h.users.GetByEmail(r.Context(), email)
	notFound := errors.Is(userErr, repository.ErrUserNotFound)
	if userErr != nil && !notFound {
		log.Printf("login: %v", userErr)
		fail(http.StatusInternalServerError, "Не удалось выполнить вход")
		return
	}

	hash := h.dummyHash
	if !notFound {
		hash = []byte(user.PasswordHash)
	}
	cmpErr := bcrypt.CompareHashAndPassword(hash, []byte(password))

	if notFound || cmpErr != nil {
		fail(http.StatusUnauthorized, "Неверный email или пароль")
		return
	}

	if old, err := r.Cookie(auth.CookieName); err == nil {
		_ = h.sessions.Delete(r.Context(), old.Value)
	}

	sessionID, err := auth.NewSessionID()
	if err != nil {
		log.Printf("session id: %v", err)
		fail(http.StatusInternalServerError, "Не удалось выполнить вход")
		return
	}
	expires := time.Now().Add(sessionTTL)
	if err := h.sessions.Create(r.Context(), sessionID, user.ID, expires); err != nil {
		log.Printf("create session: %v", err)
		fail(http.StatusInternalServerError, "Не удалось выполнить вход")
		return
	}

	auth.SetSessionCookie(w, sessionID, expires)
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := h.sessions.Delete(r.Context(), c.Value); err != nil {
			log.Printf("logout: %v", err)
		}
	}
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
