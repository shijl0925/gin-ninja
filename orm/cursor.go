package orm

import (
	"errors"
	"fmt"
	"net/http"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/pagination"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CursorPageOption customizes SelectCursorPage query construction.
type CursorPageOption func(*cursorPageConfig)

// CursorDecoder converts the opaque cursor string into the column value used
// by the keyset WHERE condition.
type CursorDecoder[C any] func(string) (C, error)

type cursorPageConfig struct {
	desc bool
}

// CursorPageDesc orders the cursor column in descending order and applies a
// "less than cursor" keyset condition.
func CursorPageDesc() CursorPageOption {
	return func(cfg *cursorPageConfig) {
		cfg.desc = true
	}
}

// SelectCursorPage applies single-column keyset pagination to db, fetches one
// extra row to detect the next page, and returns the trimmed items plus the next
// cursor value. The cursorColumn argument must be a trusted model column name.
func SelectCursorPage[T any, C any](
	db *gorm.DB,
	input pagination.CursorPagination,
	cursorColumn string,
	decodeCursor CursorDecoder[C],
	cursorFromItem func(T) string,
	opts ...CursorPageOption,
) ([]T, string, error) {
	if db == nil {
		return nil, "", errors.New("orm: nil database")
	}
	if cursorColumn == "" {
		return nil, "", errors.New("orm: cursor column is required")
	}
	if decodeCursor == nil {
		return nil, "", errors.New("orm: cursor decoder is required")
	}
	if cursorFromItem == nil {
		return nil, "", errors.New("orm: cursor extractor is required")
	}

	cfg := cursorPageConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	column := clause.Column{Name: cursorColumn}
	query := db.Order(clause.OrderByColumn{Column: column, Desc: cfg.desc})
	if cursor := input.GetCursor(); cursor != "" {
		cursorValue, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", ninja.NewError(http.StatusBadRequest, fmt.Sprintf("invalid cursor: %s", err))
		}
		if cfg.desc {
			query = query.Where(clause.Lt{Column: column, Value: cursorValue})
		} else {
			query = query.Where(clause.Gt{Column: column, Value: cursorValue})
		}
	}

	limit := input.Limit()
	var items []T
	if err := query.Limit(limit + 1).Find(&items).Error; err != nil {
		return nil, "", err
	}
	if len(items) <= limit {
		return items, "", nil
	}

	items = items[:limit]
	nextCursor := cursorFromItem(items[len(items)-1])
	if nextCursor == "" {
		return nil, "", errors.New("orm: cursor extractor returned empty string for non-terminal page")
	}
	return items, nextCursor, nil
}
