package dtos

import "strconv"

// numericToFloat converts the text lib/pq returns for a NUMERIC column. The
// exactness lives in the database - prices are NUMERIC(10,2) and totals are
// multiplied in SQL - so nothing here is arithmetic, and a value Postgres
// printed always parses.
func numericToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
