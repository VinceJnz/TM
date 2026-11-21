# Browser request lifecycle

The Fetch bridge requires AbortController. RequestContext and PageRequestContext
propagate caller cancellation; HTTPClient.Timeout bounds each operation.
Client.Destroy cancels all active requests and rejects reuse. Copies used for
per-request headers share the same lifecycle, so they cannot escape destruction.
Fetch and body-promise callbacks use the shared owned callback helpers and settle
after success, failure or browser abort. Cancellation also releases Go waiters.

Booking page loads have a per-load context. Invalidation on hide, reset or mutation
cancels it; generation checks prevent late data or cursors replacing newer state.
Use the same context APIs as other views migrate. Existing callback APIs remain
compatible. Cancellation does not undo a mutation already committed by the server;
refresh authoritative state before retrying a possibly successful write.
