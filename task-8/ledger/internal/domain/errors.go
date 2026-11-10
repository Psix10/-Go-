package domain

import "errors"

var (
	ErrBudgetNotFound = errors.New("budget not found")
	ErrBudgetExceeded = errors.New("budget exceeded")
)
