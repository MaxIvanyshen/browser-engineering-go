package engine

import (
	"fmt"
	"slices"
	"strings"
)

type URL struct {
	Scheme string
	Host   string
	Path   string
	Port   string
}

func Parse(url string) (*URL, error) {
	parts := strings.Split(url, "://")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid URL format")
	}
	scheme := parts[0]
	if !slices.Contains(allowedSchemes, scheme) {
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}

	port := ""
	if scheme == "http" {
		port = "80"
	} else if scheme == "https" {
		port = "443"
	}

	if len(parts) > 1 {
		url = parts[1]
	} else {
		url = ""
	}
	parts = strings.SplitN(url, "/", 2)
	host := parts[0]

	if len(parts) > 1 {
		url = parts[1]
	} else {
		url = ""
	}

	return &URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/" + url,
		Port:   port,
	}, nil
}

func (u *URL) String() string {
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.Path)
}
