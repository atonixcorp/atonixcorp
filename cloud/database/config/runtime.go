package config

import (
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
)

func DatabaseDSN() (string, error) {
	if dsn := strings.TrimSpace(os.Getenv("ATONIXCORP_DB_DSN")); dsn != "" {
		return dsn, nil
	}

	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	username := strings.TrimSpace(os.Getenv("DB_USER"))
	password := os.Getenv("DB_PASSWORD")
	database := strings.TrimSpace(os.Getenv("DB_NAME"))
	port := strings.TrimSpace(os.Getenv("DB_PORT"))
	if host == "" || username == "" || password == "" || database == "" {
		return "", errors.New("ATONIXCORP_DB_DSN or DB_HOST, DB_USER, DB_PASSWORD, and DB_NAME are required")
	}
	if port == "" {
		port = "5432"
	}

	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + database,
	}
	query := url.Values{}
	if sslMode := strings.TrimSpace(os.Getenv("DB_SSLMODE")); sslMode != "" {
		query.Set("sslmode", sslMode)
	} else {
		query.Set("sslmode", "disable")
	}
	dsn.RawQuery = query.Encode()
	return dsn.String(), nil
}
