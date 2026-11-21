package httpProcessor

import (
	"log"
	"net/url"
	"strconv"
)

// Page belongs to the view, not the shared client, so callbacks do not retain
// destroyed views. Lookup callers may continue using the legacy list transport.
type Page struct {
	Offset, Limit, Next int
	Path                string
	Loading             bool
}

func (c *Client) PageRequest(page *Page, method, path string, rx, tx any, callbacks ...func(error)) {
	if page.Loading {
		return
	}
	if page.Path != path {
		page.Offset = 0
		page.Path = path
	}
	if page.Limit <= 0 {
		page.Limit = 100
	}
	parsed, err := url.Parse(path)
	if err != nil {
		if len(callbacks) > 1 && callbacks[1] != nil {
			callbacks[1](err)
		}
		return
	}
	query := parsed.Query()
	query.Set("limit", strconv.Itoa(page.Limit))
	query.Set("offset", strconv.Itoa(page.Offset))
	parsed.RawQuery = query.Encode()
	page.Loading = true
	defer func() { page.Loading = false }()
	if err = c.newRequestPage(page, method, parsed.String(), rx, tx, callbacks...); err != nil {
		log.Print("page request failed: ", err)
	}
}

// Keep the established callback API for non-paged callers.
func (c *Client) newRequest(method, path string, rx, tx any, callbacks ...func(error)) error {
	return c.newRequestPage(nil, method, path, rx, tx, callbacks...)
}
