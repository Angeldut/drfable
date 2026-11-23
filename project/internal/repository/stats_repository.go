package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"lr2/internal/models"
)

const uncategorized = "Без категории"

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func (r *ExpenseRepository) StatsByCategory(ctx context.Context, userID int64, from, to time.Time) ([]models.CategoryStat, error) {
	where, args := buildWhere(userID, models.ExpenseFilter{From: from, To: to})
	query := `
SELECT COALESCE(c.name, '` + uncategorized + `'), SUM(e.amount)
FROM expenses e
LEFT JOIN categories c ON c.id = e.category_id` + where + `
GROUP BY e.category_id, c.name
ORDER BY SUM(e.amount) DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("stats by category: %w", err)
	}
	defer rows.Close()

	stats := []models.CategoryStat{}
	for rows.Next() {
		var s models.CategoryStat
		if err := rows.Scan(&s.Category, &s.Total); err != nil {
			return nil, fmt.Errorf("stats by category: scan: %w", err)
		}
		s.Total = round2(s.Total)
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stats by category: rows: %w", err)
	}
	return stats, nil
}

func (r *ExpenseRepository) StatsByMonth(ctx context.Context, userID int64, year int) ([]models.MonthStat, error) {
	const query = `
SELECT CAST(strftime('%m', e.date) AS INTEGER), SUM(e.amount)
FROM expenses e
WHERE e.user_id = ? AND e.date >= ? AND e.date <= ?
GROUP BY 1
ORDER BY 1`

	from := fmt.Sprintf("%04d-01-01", year)
	to := fmt.Sprintf("%04d-12-31", year)

	rows, err := r.db.QueryContext(ctx, query, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("stats by month: %w", err)
	}
	defer rows.Close()

	stats := make([]models.MonthStat, 12)
	for i := range stats {
		stats[i].Month = i + 1
	}
	for rows.Next() {
		var month int
		var total float64
		if err := rows.Scan(&month, &total); err != nil {
			return nil, fmt.Errorf("stats by month: scan: %w", err)
		}
		if month >= 1 && month <= 12 {
			stats[month-1].Total = round2(total)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stats by month: rows: %w", err)
	}
	return stats, nil
}
