# Collection navigation

Array response bodies and offset pagination remain supported (limit 1–500,
default 100). Growing list views negotiate `pagination=cursor&after=0` and read
`X-Pagination-Mode: cursor` plus `X-Next-Cursor`. Subsequent requests use the last
resource ID as `after`; ownership filtering remains inside the underlying SQL.
The shared client keeps cursor history for Previous and commits navigation only
after success. It falls back to offsets for endpoints without cursor support.

Cursors prevent earlier deletions from shifting the next page and avoid scanning
large offsets. They do not provide a database snapshot across browser requests;
newer IDs may appear and changed records may disappear from a filtered list.
Offset labels describe navigation position, not a live total count.

Person selectors use `/users/lookup?q=...` (two or more characters, at most 100
bytes, maximum 25 results), or `id` to restore an existing selection. The route
requires the existing users GET capability with any scope. It returns only ID,
name, and username. Small reference lookups retain the bounded legacy transport.
