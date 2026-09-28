package models

import (
	"sync"
	"time"
)

type Expense struct {
	ID          int
	Amount      float64
	Description string
	Date        time.Time
}
type Store struct {
	mu       sync.RWMutex
	expenses []Expense
	nextID   int
}

func NewStore() *Store {
	return &Store{nextID: 1}
}

func (s *Store) Add(amount float64, description string, date time.Time) Expense {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := Expense{ID: s.nextID, Amount: amount, Description: description, Date: date}
	s.nextID++
	s.expenses = append(s.expenses, e)
	return e
}

func (s *Store) All() []Expense {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Expense, len(s.expenses))
	copy(out, s.expenses)
	return out
}

func (s *Store) Total() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sum float64
	for _, e := range s.expenses {
		sum += e.Amount
	}
	return sum
}
