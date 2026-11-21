//go:build js && wasm

package httpProcessor

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
	"testing"
	"time"
)

func TestFetchHandlesBodyRejectionAndSuccess(t *testing.T) {
	for _, reject := range []bool{true, false} {
		oldFetch := js.Global().Get("fetch")
		text := js.FuncOf(func(js.Value, []js.Value) any {
			if reject {
				return js.Global().Get("Promise").Call("reject", js.Global().Get("Error").New("synthetic body failure"))
			}
			return js.Global().Get("Promise").Call("resolve", "complete body")
		})
		fetch := js.FuncOf(func(js.Value, []js.Value) any {
			return js.Global().Get("Promise").Call("resolve", js.ValueOf(map[string]any{"status": 200, "statusText": "OK", "text": text}))
		})
		js.Global().Set("fetch", fetch)
		client := New("https://example.test")
		request, _ := http.NewRequest("GET", "https://example.test/", nil)
		finished := make(chan error, 1)
		go func() {
			response, err := client.doRequest(request)
			if err == nil {
				data, readErr := io.ReadAll(response.Body)
				response.Body.Close()
				if string(data) != "complete body" {
					t.Error("response body was lost")
				}
				err = readErr
			}
			finished <- err
		}()
		select {
		case err := <-finished:
			if reject && (err == nil || !strings.Contains(err.Error(), "synthetic body failure")) {
				t.Fatalf("body rejection did not reach caller: %v", err)
			}
			if !reject && err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("body rejection left request blocked")
		}
		js.Global().Set("fetch", oldFetch)
		fetch.Release()
		text.Release()
	}
}

func TestListFollowsNextOffsetWithoutLosingRows(t *testing.T) {
	old := js.Global().Get("fetch")
	defer js.Global().Set("fetch", old)
	js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `
  const next = new URL(url).searchParams.get('offset');
  return Promise.resolve(new Response(JSON.stringify(next ? [{id:2}] : [{id:1}]),
    {status:200, headers:next ? {} : {'X-Next-Offset':'1'}}));
 `))
	client := New("https://example.test/custom/v9")
	var records []struct {
		ID int `json:"id"`
	}
	finished := make(chan error, 1)
	go func() { finished <- client.newRequest("GET", "/bookings", &records, nil) }()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("paginated fetch blocked")
	}
	if len(records) != 2 || records[0].ID != 1 || records[1].ID != 2 {
		t.Fatal(records)
	}
}

func TestExplicitPageRequestDoesNotDownloadFollowingPages(t *testing.T) {
	old := js.Global().Get("fetch")
	defer js.Global().Set("fetch", old)
	js.Global().Set("__tmPageFetches", 0)
	defer js.Global().Delete("__tmPageFetches")
	js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `
  globalThis.__tmPageFetches++;
  const offset=Number(new URL(url).searchParams.get('offset'));
  return Promise.resolve(new Response(JSON.stringify([{id:offset+1}]),{status:200,headers:{'X-Next-Offset':String(offset+1)}}));
 `))
	page := Page{Limit: 1}
	client := New("https://example.test/custom/v9")
	var records []struct {
		ID int `json:"id"`
	}
	done := make(chan struct{}, 1)
	go func() { client.PageRequest(&page, "GET", "/bookings", &records, nil); done <- struct{}{} }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("page request blocked")
	}
	if len(records) != 1 || records[0].ID != 1 || page.Next != 1 || js.Global().Get("__tmPageFetches").Int() != 1 {
		t.Fatal(records, page)
	}
	page.Offset = page.Next
	go func() { client.PageRequest(&page, "GET", "/bookings", &records, nil); done <- struct{}{} }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("next page blocked")
	}
	if records[0].ID != 2 || js.Global().Get("__tmPageFetches").Int() != 2 {
		t.Fatal(records, page)
	}
}

func TestFailedPageNavigationPreservesCommittedRecordsAndPosition(t *testing.T) {
	old := js.Global().Get("fetch")
	defer js.Global().Set("fetch", old)
	client := New("https://example.test")
	page := Page{Limit: 1}
	records := []struct {
		ID int `json:"id"`
	}{}
	fetch := func(body string, status int) {
		js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `return Promise.resolve(new Response(`+strconv.Quote(body)+`,{status:`+strconv.Itoa(status)+`,headers:{'X-Next-Offset':'1'}}));`))
	}
	request := func() {
		done := make(chan struct{}, 1)
		go func() {
			client.PageRequest(&page, "GET", "/bookings", &records, nil, nil, func(error) {
				if page.Offset != 0 || len(records) != 1 || records[0].ID != 1 {
					t.Error("failure callback observed uncommitted state")
				}
			})
			done <- struct{}{}
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("page blocked")
		}
	}
	fetch(`[{"id":1}]`, 200)
	request()
	for _, body := range []string{`[{"id":2},{"id":"invalid"}]`, `unavailable`} {
		page.Offset = page.Next
		fetch(body, 200)
		request()
		if page.Offset != 0 || page.Next != 1 || page.Loading || len(records) != 1 || records[0].ID != 1 {
			t.Fatal("failed navigation changed committed state", page, records)
		}
	}
	page.Offset = page.Next
	fetch(`unavailable`, 503)
	request()
	fetch(`[{"id":2}]`, 200)
	page.Offset = page.Next
	request()
	if page.Offset != 1 || records[0].ID != 2 {
		t.Fatal("retry did not advance")
	}
}

func TestPageCursorHistorySupportsNextAndPrevious(t *testing.T) {
	old := js.Global().Get("fetch")
	defer js.Global().Set("fetch", old)
	js.Global().Set("fetch", js.Global().Get("Function").New("url", "options", `
 const q=new URL(url).searchParams;if(q.get('pagination')!=='cursor')throw Error('missing cursor mode');
 const after=Number(q.get('after'));return Promise.resolve(new Response(JSON.stringify([{id:after+10}]),{status:200,headers:{'X-Pagination-Mode':'cursor','X-Next-Cursor':String(after+10),'X-Next-Offset':String(Number(q.get('offset'))+1)}}));`))
	client := New("https://example.test")
	page := Page{Limit: 1}
	var records []struct {
		ID int `json:"id"`
	}
	request := func() {
		done := make(chan struct{}, 1)
		go func() { client.PageRequest(&page, "GET", "/bookings", &records, nil); done <- struct{}{} }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("cursor request blocked")
		}
	}
	request()
	page.Offset = page.Next
	request()
	if records[0].ID != 20 {
		t.Fatal(records, page)
	}
	page.Offset = 0
	request()
	if records[0].ID != 10 {
		t.Fatal("previous lost cursor", records, page)
	}
}
