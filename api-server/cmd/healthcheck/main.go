package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

func probe(ctx context.Context, client *http.Client, url string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("not ready")
	}
	return nil
}
func main() {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8085"
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if probe(ctx, &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, "http://127.0.0.1:"+port+"/readyz") != nil {
		os.Exit(1)
	}
}
