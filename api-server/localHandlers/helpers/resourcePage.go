package helpers

import (
	"api-server/v2/modelMethods/resourcequery"
	"net/http"
	"strconv"
)

func ReadResourcePage(w http.ResponseWriter, r *http.Request) (resourcequery.Page, bool) {
	page := resourcequery.Page{Limit: 100, Cursor: r.URL.Query().Get("pagination") == "cursor"}
	valid := true
	for name, target := range map[string]*int{"limit": &page.Limit, "offset": &page.Offset} {
		if raw := r.URL.Query().Get(name); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				valid = false
			}
			*target = n
		}
	}
	if page.Cursor {
		if raw := r.URL.Query().Get("after"); raw != "" {
			var err error
			page.After, err = strconv.ParseInt(raw, 10, 64)
			if err != nil {
				valid = false
			}
		}
	}
	if !valid || page.Limit < 1 || page.Limit > 500 || page.Offset < 0 || page.Offset > 1000000 || page.After < 0 {
		WriteError(w, 400, "invalid_pagination", "limit must be 1..500 and offset 0..1000000")
		return page, false
	}
	return page, true
}
func WriteResourcePage(w http.ResponseWriter, items any, page resourcequery.PageInfo) {
	w.Header().Set("X-Page-Limit", strconv.Itoa(page.Limit))
	if page.Cursor {
		w.Header().Set("X-Pagination-Mode", "cursor")
		if page.NextOffset >= 0 {
			w.Header().Set("X-Next-Cursor", strconv.FormatInt(page.NextCursor, 10))
		}
	}
	if page.NextOffset >= 0 {
		w.Header().Set("X-Next-Offset", strconv.Itoa(page.NextOffset))
	}
	WriteJSON(w, 200, items)
}
