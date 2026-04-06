package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type registryDump struct {
	Table   string                   `json:"table"`
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

func main() {
	host := getenv("MYSQL_HOST", "127.0.0.1")
	port := getenv("MYSQL_PORT", "3307")
	user := getenv("MYSQL_USER", "sqitch")
	pass := getenv("MYSQL_PASSWORD", "sqitch")
	db := getenv("MYSQL_DB", "sqitch")
	outDir := getenv("OUT_DIR", "")
	if outDir == "" {
		fmt.Fprintln(os.Stderr, "OUT_DIR is required")
		os.Exit(1)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true", user, pass, host, port, db)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	tables := []string{"changes", "dependencies", "events", "projects", "releases", "tags"}
	for _, table := range tables {
		dump, err := dumpTable(conn, db, table)
		if err != nil {
			panic(err)
		}
		data, err := json.MarshalIndent(dump, "", "  ")
		if err != nil {
			panic(err)
		}
		if err := os.MkdirAll(outDir, 0755); err != nil {
			panic(err)
		}
		path := filepath.Join(outDir, table+".json")
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			panic(err)
		}
	}
}

func dumpTable(db *sql.DB, schema, table string) (*registryDump, error) {
	cols, err := queryColumnNames(db, schema, table)
	if err != nil {
		return nil, err
	}
	orderBy := make([]string, 0, len(cols))
	for _, col := range cols {
		orderBy = append(orderBy, "`"+col+"`")
	}

	query := fmt.Sprintf("SELECT * FROM `%s` ORDER BY %s", table, strings.Join(orderBy, ","))
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	values := make([]sql.RawBytes, len(cols))
	scans := make([]interface{}, len(cols))
	for i := range values {
		scans[i] = &values[i]
	}

	out := make([]map[string]interface{}, 0)
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			if values[i] == nil {
				row[col] = nil
			} else {
				row[col] = string(values[i])
			}
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &registryDump{Table: table, Columns: cols, Rows: out}, nil
}

func queryColumnNames(db *sql.DB, schema, table string) ([]string, error) {
	rows, err := db.Query("SELECT COLUMN_NAME FROM information_schema.columns WHERE table_schema=? AND table_name=? ORDER BY ORDINAL_POSITION", schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := []string{}
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
