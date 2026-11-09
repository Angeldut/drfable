package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"time"

	"lr2/internal/models"
	"lr2/internal/repository"
)

const CookieName = "session_id"
type ctxKey struct{}

func ContextWithUser(ctx context.Context, u models.User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

func UserFromContext(ctx context.Context) (models.User, bool) {
	u, ok := ctx.Value(ctxKey{}).(models.User)
	return u, ok
}

func NewSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func SetSessionCookie(w http.ResponseWriter, id string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    id,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,                 
		SameSite: http.SameSiteLaxMode,             
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func Require(sessions *repository.SessionRepository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")

		c, err := r.Cookie(CookieName)
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := sessions.GetUser(r.Context(), c.Value)
		if errors.Is(err, repository.ErrSessionNotFound) {
			ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Printf("auth: %v", err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r.WithContext(ContextWithUser(r.Context(), user)))
	})
}
