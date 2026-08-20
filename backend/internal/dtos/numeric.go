package dtos

import (
	"log/slog"
	"strconv"
)

// numericToFloat converts the text lib/pq returns for a NUMERIC column. The
// exactness lives in the database - prices are NUMERIC(10,2) and totals are
// multiplied in SQL - so nothing here is arithmetic, and a value Postgres
// printed always parses.
func numericToFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		slog.Error("could not parse a NUMERIC value", "value", s, "error", err)
	}
	return f
}
