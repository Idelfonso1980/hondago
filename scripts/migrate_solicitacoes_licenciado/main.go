package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"honda/go-engine/internal/db"
)

func main() {
	dbPath := flag.String("db", "honda.sqlite", "caminho do banco sqlite")
	flag.Parse()

	store, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("abrir store: %v", err)
	}
	defer store.Close()

	if err := store.EnsureLegacySchema(context.Background()); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	rows, err := store.DB.QueryContext(context.Background(), "PRAGMA table_info(solicitacoes)")
	if err != nil {
		log.Fatalf("pragma table_info: %v", err)
	}
	defer rows.Close()

	fmt.Println("Colunas solicitacoes:")
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt any
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			log.Fatalf("scan: %v", err)
		}
		fmt.Printf("- %s\n", name)
	}
}

