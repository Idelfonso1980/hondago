package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"honda/go-engine/internal/applog"
	"honda/go-engine/internal/config"
	"honda/go-engine/internal/db"
	"honda/go-engine/internal/engine"
)

func main() {
	configPath := flag.String("config", "config.ini", "caminho do config.ini")
	modeOverride := flag.String("mode", "", "override do modo de reserva (b1|b4)")
	updateTokens := flag.Bool("update-tokens", false, "atualiza tokens na tabela auth usando login da API")
	flag.Parse()

	logFile, logPath, err := applog.ConfigureFileLogging(*configPath)
	if err != nil {
		log.Fatalf("configure logging: %v", err)
	}
	defer logFile.Close()
	log.Printf("[go] Log file: %s", logPath)

	cfg, err := config.LoadFromINI(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if *modeOverride != "" {
		cfg.Booking.ReservarModo = *modeOverride
	}
	if cfg.Booking.ReservarModo == "" {
		cfg.Booking.ReservarModo = "b1"
	}

	store, err := db.OpenWithURL(cfg.DatabasePath, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	start := time.Now()
	e := engine.New(cfg, store)
	if *updateTokens {
		log.Printf("[go] starting token update db=%s", cfg.DatabasePath)
		if err := e.UpdateTokens(ctx); err != nil {
			log.Fatalf("update tokens: %v", err)
		}
	} else {
		log.Printf("[go] starting engine mode=%s db=%s", cfg.Booking.ReservarModo, cfg.DatabasePath)
		if err := e.Run(ctx); err != nil {
			log.Fatalf("run engine: %v", err)
		}
	}

	log.Printf("[go] finished in %s", time.Since(start))
}
