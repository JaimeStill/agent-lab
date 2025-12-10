package images

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ParsePageRange parses a page range expression into a sorted slice of page numbers.
// Supports formats: "1", "1-5", "1,3,5", "1-5,10,15-20", "-3" (start at 1), "5-" (end at maxPage).
// Results are deduplicated and sorted.
func ParsePageRange(expr string, maxPage int) ([]int, error) {
	if expr == "" {
		return nil, fmt.Errorf("%w: empty page range", ErrInvalidPageRange)
	}

	seen := make(map[int]bool)
	parts := strings.SplitSeq(expr, ",")

	for part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			start, end, err := parseRange(part, maxPage)
			if err != nil {
				return nil, err
			}

			for i := start; i <= end; i++ {
				seen[i] = true
			}
		} else {
			page, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid page %q", ErrInvalidPageRange, part)
			}
			if page < 1 || page > maxPage {
				return nil, fmt.Errorf("%w: page %d out of range [1-%d]", ErrPageOutOfRange, page, maxPage)
			}
			seen[page] = true
		}
	}

	if len(seen) == 0 {
		return nil, fmt.Errorf("%w: no valid pages", ErrInvalidPageRange)
	}

	pages := make([]int, 0, len(seen))
	for page := range seen {
		pages = append(pages, page)
	}

	sort.Ints(pages)

	return pages, nil
}

func parseRange(part string, maxPage int) (int, int, error) {
	idx := strings.Index(part, "-")
	if idx == -1 {
		return 0, 0, fmt.Errorf("%w: invalid range %q", ErrInvalidPageRange, part)
	}

	startStr := strings.TrimSpace(part[:idx])
	endStr := strings.TrimSpace(part[idx+1:])

	var start, end int
	var err error

	if startStr == "" {
		start = 1
	} else {
		start, err = strconv.Atoi(startStr)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: invalid start %q", ErrInvalidPageRange, startStr)
		}
	}

	if endStr == "" {
		end = maxPage
	} else {
		end, err = strconv.Atoi(endStr)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: invalid end %q", ErrInvalidPageRange, endStr)
		}
	}

	if start < 1 {
		return 0, 0, fmt.Errorf("%w: start page must be >= 1", ErrInvalidPageRange)
	}
	if end > maxPage {
		return 0, 0, fmt.Errorf("%w: end page %d exceends document pages (%d)", ErrPageOutOfRange, end, maxPage)
	}
	if start > end {
		return 0, 0, fmt.Errorf("%w: start > end in %q", ErrInvalidPageRange, part)
	}

	return start, end, nil
}
