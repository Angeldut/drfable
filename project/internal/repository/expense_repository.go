package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"lr2/internal/models"
)

var ErrNotFound = errors.New("expense not found")
const dateLayout = "2006-01-02"
const selectExpenses = `
SELECT e.id, e.amount, e.description, e.date, e.category_id, c.name
FROM expenses e
LEFT JOIN categories c ON c.id = e.category_id`

type ExpenseRepository struct {
	db *sql.DB
}

func NewExpenseRepository(db *sql.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func nullableID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func buildWhere(f models.ExpenseFilter) (string, []any) {
	var conds []string
	var args []any

	if f.CategoryID != 0 {
		conds = append(conds, "e.category_id = ?")
		args = append(args, f.CategoryID)
	}
	if !f.From.IsZero() {
		conds = append(conds, "e.date >= ?")
		args = append(args, f.From.Format(dateLayout))
	}
	if !f.To.IsZero() {
		conds = append(conds, "e.date <= ?")
		args = append(args, f.To.Format(dateLayout))
	}

	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

func (r *ExpenseRepository) Create(ctx context.Context, amount float64, description string, date time.Time, categoryID int64) (models.Expense, error) {
	const query = `INSERT INTO expenses (amount, description, date, category_id) VALUES (?, ?, ?, ?)`

	res, err := r.db.ExecContext(ctx, query, amount, description, date.Format(dateLayout), nullableID(categoryID))
	if err != nil {
		return models.Expense{}, fmt.Errorf("create expense: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.Expense{}, fmt.Errorf("create expense: get id: %w", err)
	}

	return models.Expense{ID: id, Amount: amount, Description: description, Date: date, CategoryID: categoryID}, nil
}

func (r *ExpenseRepository) GetAll(ctx context.Context, f models.ExpenseFilter) ([]models.Expense, error) {
	where, args := buildWhere(f)
	query := selectExpenses + where + " ORDER BY e.date DESC, e.id DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
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
	row := r.db.QueryRowContext(ctx, selectExpenses+" WHERE e.id = ?", id)
	e, err := scanExpense(row)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Expense{}, ErrNotFound
	}
	if err != nil {
		return models.Expense{}, fmt.Errorf("get expense %d: %w", id, err)
	}
	return e, nil
}

func (r *ExpenseRepository) Update(ctx context.Context, id int64, amount float64, description string, date time.Time, categoryID int64) error {
	const query = `UPDATE expenses SET amount = ?, description = ?, date = ?, category_id = ? WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query, amount, description, date.Format(dateLayout), nullableID(categoryID), id)
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
func (r *ExpenseRepository) Total(ctx context.Context, f models.ExpenseFilter) (float64, error) {
	where, args := buildWhere(f)
	query := `SELECT COALESCE(SUM(e.amount), 0) FROM expenses e` + where

	var total float64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
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
		catID   sql.NullInt64
		catName sql.NullString
	)
	if err := s.Scan(&e.ID, &e.Amount, &e.Description, &dateStr, &catID, &catName); err != nil {
		return models.Expense{}, err
	}
	date, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return models.Expense{}, fmt.Errorf("parse date %q: %w", dateStr, err)
	}
	e.Date = date
	e.CategoryID = catID.Int64 
	e.CategoryName = catName.String
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
