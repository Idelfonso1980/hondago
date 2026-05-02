package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type BookingConfig struct {
	Modelo              string
	Produto             []int
	ReservarModo        string
	Grupo               []int
	CPF                 []string
	CodEmpre            []string
	Vencimento          []int
	IDCota              []int
	LoteriaFederal      string
	AcrescimoDecrescimo string
	TipoGrupo           string
	Limit               int
	DryRun              bool
	CooldownUserMS      int
	WorkerCount         int
	RequestTimeoutMS    int
	TokenPrincipal      string
}

type Config struct {
	DatabasePath string
	DatabaseURL  string
	APIBaseURL   string
	CrawlerAPIURL string
	Booking      BookingConfig
}

func LoadFromINI(path string) (*Config, error) {
	loadDotEnvIfPresent(path)
	raw, err := parseINI(path)
	if err != nil {
		return nil, err
	}
	configDir := filepath.Dir(path)
	defaultDBPath := filepath.Join(configDir, "honda.sqlite")
	booking := raw["BOOKING"]
	cfg := &Config{
		DatabasePath: defaultDBPath,
		APIBaseURL:   "https://apigate.hondaservicosfinanceiros.com.br/bff-plataforma-vendas/v1",
		CrawlerAPIURL: "https://apigate.hondaservicosfinanceiros.com.br/bff-plataforma-vendas/v1/valorlance",
		Booking: BookingConfig{
			Modelo:              getStr(booking, "modelo", ""),
			Produto:             getIntList(booking, "produto"),
			ReservarModo:        strings.ToLower(getStr(booking, "reservar_modo", "b1")),
			Grupo:               getIntList(booking, "grupo"),
			CPF:                 getStrList(booking, "cpf"),
			CodEmpre:            getStrList(booking, "cod_empre"),
			Vencimento:          getIntList(booking, "vencimento"),
			IDCota:              getIntList(booking, "id_cota"),
			LoteriaFederal:      strings.TrimSpace(getStr(booking, "loteria_federal", "")),
			AcrescimoDecrescimo: strings.TrimSpace(getStr(booking, "acrescimo_decrescimo", "")),
			TipoGrupo:           strings.TrimSpace(getStr(booking, "tipo_grupo", "")),
			Limit:               getInt(booking, "limit", 0),
			DryRun:              getBool(booking, "dry_run", false),
			CooldownUserMS:      getInt(booking, "cooldown_user_ms", 200),
			WorkerCount:         getInt(booking, "worker_count_go", 24),
			RequestTimeoutMS:    getInt(booking, "request_timeout_ms", 7000),
			TokenPrincipal:      strings.TrimSpace(getStr(booking, "token_principal", "")),
		},
	}
	if v := strings.TrimSpace(getStr(raw["SYSTEM"], "database_path", "")); v != "" {
		if filepath.IsAbs(v) {
			cfg.DatabasePath = v
		} else {
			cfg.DatabasePath = filepath.Join(configDir, v)
		}
	}
	if v := strings.TrimSpace(getStr(raw["SYSTEM"], "api_base_url", "")); v != "" {
		cfg.APIBaseURL = v
	}
	if v := strings.TrimSpace(getStr(raw["CRAWLER"], "api_url", "")); v != "" {
		cfg.CrawlerAPIURL = v
	}
	applyEnvOverrides(cfg)
	if cfg.Booking.ReservarModo == "" {
		cfg.Booking.ReservarModo = "b1"
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}
	if v, ok := getEnvTrimmed("HONDAGO_TOKEN_PRINCIPAL"); ok {
		cfg.Booking.TokenPrincipal = v
	}
	if v, ok := getEnvTrimmed("HONDAGO_API_BASE_URL"); ok {
		cfg.APIBaseURL = v
	}
	if v, ok := getEnvTrimmed("HONDAGO_API_URL"); ok {
		cfg.CrawlerAPIURL = v
	}
	if v, ok := getEnvTrimmed("HONDAGO_DATABASE_URL"); ok {
		cfg.DatabaseURL = v
	}
}

func getEnvTrimmed(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	return v, true
}

func loadDotEnvIfPresent(configPath string) {
	dotEnvPath := filepath.Join(filepath.Dir(configPath), ".env")
	file, err := os.Open(dotEnvPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if key == "" {
			continue
		}
		val = strings.Trim(val, `"'`)

		// Respect explicit environment values already set by the process.
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
}

func parseINI(path string) (map[string]map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	sections := map[string]map[string]string{}
	current := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "\ufeff")
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if _, ok := sections[current]; !ok {
				sections[current] = map[string]string{}
			}
			continue
		}
		if current == "" {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		sections[current][k] = v
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan config: %w", err)
	}
	return sections, nil
}

func getStr(section map[string]string, key, fallback string) string {
	if section == nil {
		return fallback
	}
	if v, ok := section[key]; ok {
		return v
	}
	return fallback
}

func getInt(section map[string]string, key string, fallback int) int {
	raw := strings.TrimSpace(getStr(section, key, ""))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getBool(section map[string]string, key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(getStr(section, key, "")))
	if raw == "" {
		return fallback
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "sim" || raw == "y"
}

func getIntList(section map[string]string, key string) []int {
	raw := strings.TrimSpace(getStr(section, key, ""))
	items := splitList(raw)
	out := make([]int, 0, len(items))
	for _, item := range items {
		v, err := strconv.Atoi(item)
		if err == nil {
			out = append(out, v)
		}
	}
	return out
}

func getStrList(section map[string]string, key string) []string {
	raw := strings.TrimSpace(getStr(section, key, ""))
	return splitList(raw)
}

func splitList(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		p = strings.Trim(p, "'\"")
		if p != "" {
			out = append(out, p)
		}
	}
	// Plain non-list values should still be accepted as single element.
	if len(out) == 0 && value != "" {
		return []string{strings.Trim(value, "'\"")}
	}
	return out
}
