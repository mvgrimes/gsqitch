package target

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Target represents a database deployment target
type Target struct {
	Name     string
	URI      *URI
	Engine   string
	Registry string
	Client   string
	TopDir   string
	PlanFile string
}

// URI represents a parsed database URI
type URI struct {
	Scheme   string
	User     string
	Password string
	Host     string
	Port     int
	Database string
	Params   map[string]string
}

// ParseURI parses a database URI string
// Supported formats:
//   - db:engine://user:pass@host:port/database?param=value
//   - db:engine:database
//   - engine://user:pass@host:port/database
func ParseURI(uriStr string) (*URI, error) {
	u := &URI{
		Params: make(map[string]string),
	}

	// Handle db: prefix
	if strings.HasPrefix(uriStr, "db:") {
		uriStr = strings.TrimPrefix(uriStr, "db:")
	}

	// Check for simple format: engine:database
	if !strings.Contains(uriStr, "://") {
		parts := strings.SplitN(uriStr, ":", 2)
		if len(parts) == 2 {
			u.Scheme = parts[0]
			u.Database = parts[1]
			return u, nil
		}
		// Just engine name
		u.Scheme = uriStr
		return u, nil
	}

	// Parse as URL
	parsed, err := url.Parse(uriStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URI: %w", err)
	}

	u.Scheme = parsed.Scheme
	u.Host = parsed.Hostname()

	if portStr := parsed.Port(); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}
		u.Port = port
	}

	if parsed.User != nil {
		u.User = parsed.User.Username()
		u.Password, _ = parsed.User.Password()
	}

	// Database is the path without leading slash
	u.Database = strings.TrimPrefix(parsed.Path, "/")

	// Parse query parameters
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			u.Params[key] = values[0]
		}
	}

	return u, nil
}

// String returns the URI as a string
func (u *URI) String() string {
	if u.Host == "" && u.User == "" {
		// Simple format
		if u.Database != "" {
			return fmt.Sprintf("db:%s:%s", u.Scheme, u.Database)
		}
		return fmt.Sprintf("db:%s", u.Scheme)
	}

	// Full URL format
	var sb strings.Builder
	sb.WriteString("db:")
	sb.WriteString(u.Scheme)
	sb.WriteString("://")

	if u.User != "" {
		sb.WriteString(url.QueryEscape(u.User))
		if u.Password != "" {
			sb.WriteString(":")
			sb.WriteString(url.QueryEscape(u.Password))
		}
		sb.WriteString("@")
	}

	sb.WriteString(u.Host)
	if u.Port != 0 {
		sb.WriteString(fmt.Sprintf(":%d", u.Port))
	}

	if u.Database != "" {
		sb.WriteString("/")
		sb.WriteString(u.Database)
	}

	if len(u.Params) > 0 {
		sb.WriteString("?")
		first := true
		for k, v := range u.Params {
			if !first {
				sb.WriteString("&")
			}
			sb.WriteString(url.QueryEscape(k))
			sb.WriteString("=")
			sb.WriteString(url.QueryEscape(v))
			first = false
		}
	}

	return sb.String()
}

// DSN returns a MySQL-compatible DSN string
func (u *URI) DSN() string {
	var sb strings.Builder

	if u.User != "" {
		sb.WriteString(u.User)
		if u.Password != "" {
			sb.WriteString(":")
			sb.WriteString(u.Password)
		}
		sb.WriteString("@")
	}

	if u.Host != "" {
		sb.WriteString("tcp(")
		sb.WriteString(u.Host)
		if u.Port != 0 {
			sb.WriteString(fmt.Sprintf(":%d", u.Port))
		}
		sb.WriteString(")")
	}

	sb.WriteString("/")
	sb.WriteString(u.Database)

	// Add default parameters for MySQL
	params := make(map[string]string)
	params["parseTime"] = "true"
	params["multiStatements"] = "true"
	for k, v := range u.Params {
		params[k] = v
	}

	if len(params) > 0 {
		sb.WriteString("?")
		first := true
		for k, v := range params {
			if !first {
				sb.WriteString("&")
			}
			sb.WriteString(k)
			sb.WriteString("=")
			sb.WriteString(v)
			first = false
		}
	}

	return sb.String()
}

// Engine returns the engine name from the scheme
func (u *URI) Engine() string {
	return u.Scheme
}

// New creates a new Target from a URI string
func New(name, uriStr string) (*Target, error) {
	uri, err := ParseURI(uriStr)
	if err != nil {
		return nil, err
	}

	return &Target{
		Name:   name,
		URI:    uri,
		Engine: uri.Engine(),
	}, nil
}

// DefaultPort returns the default port for an engine
func DefaultPort(engine string) int {
	switch engine {
	case "mysql", "mariadb":
		return 3306
	case "pg", "postgres", "postgresql":
		return 5432
	case "sqlite":
		return 0
	default:
		return 0
	}
}

// DefaultRegistry returns the default registry database name for an engine
func DefaultRegistry(engine string) string {
	return "sqitch"
}
