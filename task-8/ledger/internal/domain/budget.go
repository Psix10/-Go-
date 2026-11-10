package domain

import (
	"errors"
	"math"
	"strings"
)

type Budget struct {
	Category string
	Limit    float64
	Period   string
}

func (b Budget) Validate() error {
	if strings.TrimSpace(b.Category) == "" {
		return errors.New("budget category cannot be empty")
	}

	if math.IsNaN(b.Limit) || math.IsInf(b.Limit, 0) || b.Limit <= 0 {
		return errors.New("budget limit must be a finite positive number")
	}

	return nil
}
