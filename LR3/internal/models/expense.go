package models
import "time"
type Expense struct {
	ID          int64
	Amount      float64
	Description string
	Date        time.Time
}
