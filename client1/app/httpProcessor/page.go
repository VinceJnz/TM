package httpProcessor

import (
	"fmt"
	"log"
	"net/url"
	"reflect"
	"strconv"
)

// Page belongs to the view, not the shared client, so callbacks do not retain
// destroyed views. Lookup callers may continue using the legacy list transport.
type Page struct {
	Offset, Limit, Next            int
	Path                           string
	Loading                        bool
	committedOffset, committedNext int
	committedPath                  string
	cursorSupported                bool
	nextCursor                     string
	cursors                        map[int]string
}

// Snapshot isolates mutable cursor history from an in-flight page request.
func (p Page) Snapshot() Page {
	if p.cursors != nil {
		p.cursors = cloneCursors(p.cursors)
	}
	return p
}
func cloneCursors(source map[int]string) map[int]string {
	copy := make(map[int]string, len(source))
	for offset, cursor := range source {
		copy[offset] = cursor
	}
	return copy
}

func (c *Client) PageRequest(page *Page, method, path string, rx, tx any, callbacks ...func(error)) (requestErr error) {
	if page.Loading {
		return
	}
	pending := page.Snapshot()
	fail := func(err error) {
		requestErr = err
		page.Offset, page.Next, page.Path = page.committedOffset, page.committedNext, page.committedPath
		if page.Path == "" {
			page.Next = -1
		}
		page.Loading = false
		if len(callbacks) > 1 && callbacks[1] != nil {
			callbacks[1](err)
		}
	}
	if pending.Path != path {
		pending.Offset = 0
		pending.Path = path
		pending.cursorSupported = false
		pending.cursors = nil
	}
	if pending.Limit <= 0 {
		pending.Limit = 100
	}
	parsed, err := url.Parse(path)
	if err != nil {
		fail(err)
		return
	}
	query := parsed.Query()
	query.Set("limit", strconv.Itoa(pending.Limit))
	query.Set("offset", strconv.Itoa(pending.Offset))
	if pending.Offset == 0 || pending.cursorSupported {
		query.Set("pagination", "cursor")
		after := "0"
		if pending.Offset > 0 {
			after = pending.cursors[pending.Offset]
		}
		if after != "" {
			query.Set("after", after)
		} else {
			query.Del("pagination")
		}
	}
	parsed.RawQuery = query.Encode()
	page.Loading = true
	defer func() { page.Loading = false }()
	// Decode into temporary storage: even malformed JSON must not partially
	// overwrite the currently displayed records.
	var temporary any
	if rx != nil {
		value := reflect.ValueOf(rx)
		if value.Kind() != reflect.Pointer || value.IsNil() {
			fail(fmt.Errorf("page destination must be a pointer"))
			return
		}
		temporary = reflect.New(value.Elem().Type()).Interface()
	}
	if err = c.newRequestPage(&pending, method, parsed.String(), temporary, tx, nil, func(error) {}); err != nil {
		fail(err)
		log.Print("page request failed: ", err)
		return
	}
	if rx != nil {
		reflect.ValueOf(rx).Elem().Set(reflect.ValueOf(temporary).Elem())
	}
	pending.committedOffset, pending.committedNext, pending.committedPath = pending.Offset, pending.Next, pending.Path
	if pending.cursorSupported {
		history := map[int]string{0: "0"}
		for offset, cursor := range pending.cursors {
			history[offset] = cursor
		}
		if pending.Next >= 0 {
			history[pending.Next] = pending.nextCursor
		}
		pending.cursors = history
	}
	pending.Loading = false
	*page = pending
	if len(callbacks) > 0 && callbacks[0] != nil {
		callbacks[0](nil)
	}
	return nil
}

// Keep the established callback API for non-paged callers.
func (c *Client) newRequest(method, path string, rx, tx any, callbacks ...func(error)) error {
	return c.newRequestPage(nil, method, path, rx, tx, callbacks...)
}
