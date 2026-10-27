package models

import "time"

type Expense struct {
	ID           int64
	Amount       float64
	Description  string
	Date         time.Time
	CategoryID   int64  
	CategoryName string
}

type ExpenseFilter struct {
	CategoryID int64
	From       time.Time
	To         time.Time
}

func (f ExpenseFilter) Active() bool {
	return f.CategoryID != 0 || !f.From.IsZero() || !f.To.IsZero()
}
