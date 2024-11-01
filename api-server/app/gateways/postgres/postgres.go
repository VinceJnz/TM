package postgres

import (
	"fmt"
	"net/url"

	_ "github.com/lib/pq"
)

type Config struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	ApplicationName string
	SearchPath      string
}

// ConnectionString returns the database connection string.
func ConnectionString(config Config) string {
	password := config.Password
	if password != "" {
		password = ":" + url.QueryEscape(password)
	}

	sslMode := config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	connString := fmt.Sprintf(
		"postgres://%s%s@%s:%s/%s?sslmode=%s&application_name=%s&binary_parameters=yes",
		url.QueryEscape(config.User),
		password,
		url.QueryEscape(config.Host),
		url.QueryEscape(config.Port),
		url.QueryEscape(config.Name),
		url.QueryEscape(config.SSLMode),
		url.QueryEscape(config.ApplicationName),
	)

	if len(config.SearchPath) > 0 {
		connString += fmt.Sprintf("&search_path=%s", url.QueryEscape(config.SearchPath))
	}
	return connString
}
