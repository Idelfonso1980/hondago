package applog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func utcMinus3Location() *time.Location {
	loc, err := time.LoadLocation("America/Fortaleza")
	if err == nil {
		return loc
	}
	return time.FixedZone("UTC-3", -3*60*60)
}

// ConfigureFileLogging creates a timestamped log file under <configDir>/log
// and configures the default logger to write to both stdout and this file.
func ConfigureFileLogging(configPath string) (*os.File, string, error) {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("resolve config path: %w", err)
	}
	projectRoot := filepath.Dir(absConfigPath)
	logDir := filepath.Join(projectRoot, "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("create log dir: %w", err)
	}

	timestamp := time.Now().In(utcMinus3Location()).Format("20060102_150405_-0700")
	logPath := filepath.Join(logDir, fmt.Sprintf("honda_go_%s.log", timestamp))
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("open log file: %w", err)
	}

	log.SetFlags(log.Ldate | log.Ltime)
	log.SetOutput(io.MultiWriter(os.Stdout, file))

	return file, logPath, nil
}
