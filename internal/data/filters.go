package data

import (
	"math"
	"strings"

	"github.com/Yusufdot101/greenlight/internal/validator"
)

type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafeList []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(f.Page > 0, "pages", "must be greater than 0")
	v.Check(f.Page <= 10_000_000, "pages", "must be a maximum of 10 million")

	v.Check(f.PageSize > 0, "page_size", "must be greater than 0")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")

	v.Check(
		validator.ValueInList(f.Sort, f.SortSafeList...),
		"sort", "invalid sort",
	)
}

func (f Filters) sortColumn() string {
	for _, safeValue := range f.SortSafeList {
		if f.Sort == safeValue {
			return strings.TrimPrefix(safeValue, "-")
		}
	}
	panic("unsafe sort parameter: " + f.Sort)
}

func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DECS"
	}
	return "ASC"
}

func (f Filters) limit() int {
	return f.PageSize
}

func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

type Metadata struct {
	CurrentPage   int `json:"current_page,omitempty"`
	PageSize      int `json:"page_size,omitempty"`
	FirstPage     int `json:"first_page,omitempty"`
	LastPage      int `json:"last_page,omitempty"`
	TotatlRecorld int `json:"total_records,omitempty"`
}

func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}

	lastPage := int(math.Ceil(float64(totalRecords) / float64(pageSize)))

	return Metadata{
		CurrentPage:   page,
		PageSize:      pageSize,
		FirstPage:     1,
		LastPage:      lastPage,
		TotatlRecorld: totalRecords,
	}
}
