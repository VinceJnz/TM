package httpProcessor

import "strings"

// Endpoint resolves API-relative paths for browser primitives that need a URL.
func (c *Client) Endpoint(path string) string {
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		return path
	}
	return strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
}
