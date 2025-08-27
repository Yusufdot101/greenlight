package data

import (
	"math"
	"slices"
	"strings"

	"github.com/Yusufdot101/greenlight/internal/validator"
)

type Filter struct {
	Page         int    `json:"page"`
	PageSize     int    `json:"page_size"`
	Sort         string `json:"sort"`
	SafeSortList []string
}

func ValidateFilters(v *validator.Validator, f *Filter) {
	v.CheckAdd(f.Page > 0, "page", "must be postive integer")
	v.CheckAdd(f.Page <= 10_000_000, "page", "cannot exceed 10 million")

	v.CheckAdd(f.PageSize > 0, "page", "must be postive integer")
	v.CheckAdd(f.PageSize <= 100, "page", "cannot exceed 100")

	v.CheckAdd(validator.ValueInList(f.Sort, f.SafeSortList...), "sort", "invalid")
}

func (f Filter) Limit() int {
	return f.PageSize
}

func (f *Filter) Offset() int {
	return (f.Page - 1) * f.PageSize
}

func (f Filter) SortColumn() string {
	if slices.Contains(f.SafeSortList, f.Sort) {
		return strings.TrimPrefix(f.Sort, "-")
	}
	panic("invalid sort: " + f.Sort)
}

func (f Filter) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

type Metadata struct {
	CurrentPage  int `json:"current_page"`
	PageSize     int `json:"page_size"`
	TotalRecords int `json:"total_records"`
	LastPage     int `json:"last_page"`
}

func NewMetadata(currentPage, pageSize, totalRecords int) *Metadata {
	lastPage := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	return &Metadata{
		CurrentPage:  currentPage,
		PageSize:     pageSize,
		TotalRecords: totalRecords,
		LastPage:     lastPage,
	}
}
