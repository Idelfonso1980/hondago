package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func execFile(db *sql.DB, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(b))
	return err
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run ./scripts/db_refactor_v1/sql_runner.go <db> <sql1> [sql2..]")
		os.Exit(1)
	}
	dbPath := os.Args[1]
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	for _, f := range os.Args[2:] {
		fmt.Println(">>", f)
		if err := execFile(db, f); err != nil {
			fmt.Printf("ERR %s: %v\n", f, err)
			os.Exit(2)
		}
	}
	fmt.Println("OK")
}

