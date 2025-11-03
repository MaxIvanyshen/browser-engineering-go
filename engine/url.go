package engine

import (
	"fmt"
	"slices"
	"strings"
)

type URL struct {
	scheme     string
	host       string
	path       string
	port       string
	ViewSource bool
}

func Parse(url string) (*URL, error) {
	if strings.HasPrefix(url, "view-source:") {
		parsed, err := Parse(strings.TrimPrefix(url, "view-source:"))
		if err != nil {
			return nil, err
		}
		parsed.ViewSource = true
		return parsed, nil
	}
	parts := strings.Split(url, "://")
	if strings.HasPrefix(url, "data:") {
		return &URL{
			scheme: "data",
			host:   "",
			path:   url[5:],
			port:   "",
		}, nil
	}
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
		scheme: scheme,
		host:   host,
		path:   "/" + url,
		port:   port,
	}, nil
}

func (u *URL) String() string {
	return fmt.Sprintf("%s://%s%s", u.scheme, u.host, u.path)
}
