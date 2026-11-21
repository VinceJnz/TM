# Logout transition

The browser now sends POST to `<API-prefix>/auth/logout/`. Logout belongs to the
authenticated session and does not require an administration capability. It
revokes that session only and expires the Secure, HttpOnly, SameSite=Lax cookie
with the same path/domain as session creation. Provider account linkage remains.

Legacy GET requests return 405 with `Allow: POST` and never revoke a session.
Update external clients to POST alongside the frontend deployment. Requests with
an Origin header must use the application's HTTPS origin. Requests without Origin
remain supported for non-browser clients. Other application permission grants are
unchanged; refresh authoritative login status after an interrupted logout.
