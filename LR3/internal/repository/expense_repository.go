package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"lr2/internal/models"
)
var ErrNotFound = errors.New("expense not found")

const dateLayout = "2006-01-02"

type ExpenseRepository struct {
	db *sql.DB
}

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}
func (r *ExpenseRepository) Create(ctx context.Context, amount float64, description string, date time.Time) (models.Expense, error) {
	const query = `INSERT INTO expenses (amount, description, date) VALUES (?, ?, ?)`

	res, err := r.db.ExecContext(ctx, query, amount, description, date.Format(dateLayout))
	if err != nil {
		return models.Expense{}, fmt.Errorf("create expense: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.Expense{}, fmt.Errorf("create expense: get id: %w", err)
	}

	return models.Expense{ID: id, Amount: amount, Description: description, Date: date}, nil
}

func (r *ExpenseRepository) GetAll(ctx context.Context) ([]models.Expense, error) {
	const query = `SELECT id, amount, description, date FROM expenses ORDER BY date DESC, id DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all expenses: %w", err)
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, fmt.Errorf("get all expenses: scan: %w", err)
		}
		expenses = append(expenses, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get all expenses: rows: %w", err)
	}

	return expenses, nil
}

func (r *ExpenseRepository) GetByID(ctx context.Context, id int64) (models.Expense, error) {
	const query = `SELECT id, amount, description, date FROM expenses WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	e, err := scanExpense(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Expense{}, ErrNotFound
	}
	if err != nil {
		return models.Expense{}, fmt.Errorf("get expense %d: %w", id, err)
	}
	return e, nil
}

func (r *ExpenseRepository) Update(ctx context.Context, id int64, amount float64, description string, date time.Time) error {
	const query = `UPDATE expenses SET amount = ?, description = ?, date = ? WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query, amount, description, date.Format(dateLayout), id)
	if err != nil {
		return fmt.Errorf("update expense %d: %w", id, err)
	}
	return checkAffected(res, id)
}

func (r *ExpenseRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM expenses WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete expense %d: %w", id, err)
	}
	return checkAffected(res, id)
}

func (r *ExpenseRepository) Total(ctx context.Context) (float64, error) {
	const query = `SELECT COALESCE(SUM(amount), 0) FROM expenses`

	var total float64
	if err := r.db.QueryRowContext(ctx, query).Scan(&total); err != nil {
		return 0, fmt.Errorf("total expenses: %w", err)
	}
	return total, nil
}
type rowScanner interface {
	Scan(dest ...any) error
}

func scanExpense(s rowScanner) (models.Expense, error) {
	var (
		e       models.Expense
		dateStr string
	)
	if err := s.Scan(&e.ID, &e.Amount, &e.Description, &dateStr); err != nil {
		return models.Expense{}, err
	}
	date, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return models.Expense{}, fmt.Errorf("parse date %q: %w", dateStr, err)
	}
	e.Date = date
	return e, nil
}

func checkAffected(res sql.Result, id int64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
