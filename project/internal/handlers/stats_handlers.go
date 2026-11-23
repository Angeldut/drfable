package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var monthLabels = []string{
	"Янв", "Фев", "Мар", "Апр", "Май", "Июн",
	"Июл", "Авг", "Сен", "Окт", "Ноя", "Дек",
}

type categoryStatsResponse struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
	Total  float64   `json:"total"`
}

type monthStatsResponse struct {
	Year   int       `json:"year"`
	Labels []string  `json:"labels"`
	Values []float64 `json:"values"`
	Total  float64   `json:"total"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		log.Printf("encode json: %v", err)
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (h *Handler) StatsPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, http.StatusOK, "stats.html", pageData{
		Title: "Статистика",
		Year:  time.Now().Year(),
	})
}

func (h *Handler) APIStatsByCategory(w http.ResponseWriter, r *http.Request) {
	filter, filterErr := parseFilter(r.URL.Query())
	if filterErr != "" {
		writeJSONError(w, http.StatusBadRequest, filterErr)
		return
	}

	stats, err := h.repo.StatsByCategory(r.Context(), currentUserID(r), filter.From, filter.To)
	if err != nil {
		log.Printf("stats by category: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Не удалось получить статистику")
		return
	}

	resp := categoryStatsResponse{Labels: []string{}, Values: []float64{}}
	if !filter.From.IsZero() {
		resp.From = filter.From.Format(dateLayout)
	}
	if !filter.To.IsZero() {
		resp.To = filter.To.Format(dateLayout)
	}
	for _, s := range stats {
		resp.Labels = append(resp.Labels, s.Category)
		resp.Values = append(resp.Values, s.Total)
		resp.Total += s.Total
	}
	resp.Total = round2(resp.Total)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) APIStatsByMonth(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()
	if s := strings.TrimSpace(r.URL.Query().Get("year")); s != "" {
		y, err := strconv.Atoi(s)
		if err != nil || y < 2000 || y > 2100 {
			writeJSONError(w, http.StatusBadRequest, "Некорректный год")
			return
		}
		year = y
	}

	stats, err := h.repo.StatsByMonth(r.Context(), currentUserID(r), year)
	if err != nil {
		log.Printf("stats by month: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Не удалось получить статистику")
		return
	}

	resp := monthStatsResponse{Year: year, Labels: monthLabels, Values: make([]float64, 0, len(stats))}
	for _, s := range stats {
		resp.Values = append(resp.Values, s.Total)
		resp.Total += s.Total
	}
	resp.Total = round2(resp.Total)
	writeJSON(w, http.StatusOK, resp)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
