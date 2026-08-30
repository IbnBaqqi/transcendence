package service

import (
	"math"
	"strconv"
)

type paging struct {
	page       int
	limit      int
	pageLimit  int32
	pageOffset int32
}

func parsePaging(rawPage, rawLimit string) (paging, error) {
	page := defaultPage
	if rawPage != "" {
		p, convErr := strconv.Atoi(rawPage)
		if convErr != nil || p < 1 || p > math.MaxInt32 {
			return paging{}, &ValidationError{Message: "Page must be a positive integer"}
		}
		page = p
	}

	limit := defaultLimit
	if rawLimit != "" {
		l, convErr := strconv.Atoi(rawLimit)
		if convErr != nil || l < 1 {
			return paging{}, &ValidationError{Message: "Limit must be a positive integer"}
		}
		limit = l
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := (page - 1) * limit
	if offset < 0 || offset > math.MaxInt32 {
		return paging{}, &ValidationError{Message: "Page is too large"}
	}

	return paging{
		page:       page,
		limit:      limit,
		pageLimit:  int32(limit),
		pageOffset: int32(offset),
	}, nil
}
