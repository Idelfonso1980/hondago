package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var bookingKeysOrder = []string{
	"modelo",
	"produto",
	"reservar_modo",
	"grupo",
	"cpf",
	"cod_empre",
	"vencimento",
	"id_cota",
	"loteria_federal",
	"acrescimo_decrescimo",
	"tipo_grupo",
	"limit",
	"dry_run",
	"cooldown_user_ms",
	"worker_count_go",
	"request_timeout_ms",
	"token_principal",
}

// SaveToINI persists BOOKING/SYSTEM data in INI format.
// Unknown sections from the existing file are preserved in a normalized format.
func SaveToINI(path string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config nil")
	}

	sections := map[string]map[string]string{}
	if _, err := os.Stat(path); err == nil {
		existing, err := parseINI(path)
		if err != nil {
			return err
		}
		sections = existing
	}

	if sections["BOOKING"] == nil {
		sections["BOOKING"] = map[string]string{}
	}
	setBooking(sections["BOOKING"], cfg.Booking)

	if sections["SYSTEM"] == nil {
		sections["SYSTEM"] = map[string]string{}
	}
	setSystem(path, sections["SYSTEM"], cfg)

	var b strings.Builder
	writeSection(&b, "BOOKING", sections["BOOKING"], bookingKeysOrder)
	b.WriteString("\n")
	writeSection(&b, "SYSTEM", sections["SYSTEM"], []string{"database_path", "api_base_url"})

	for section, kv := range sections {
		if section == "BOOKING" || section == "SYSTEM" {
			continue
		}
		if len(kv) == 0 {
			continue
		}
		b.WriteString("\n")
		writeSection(&b, section, kv, nil)
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func setBooking(out map[string]string, in BookingConfig) {
	out["modelo"] = strings.TrimSpace(in.Modelo)
	out["produto"] = formatIntList(in.Produto)
	out["reservar_modo"] = strings.TrimSpace(in.ReservarModo)
	out["grupo"] = formatIntList(in.Grupo)
	out["cpf"] = formatStrList(in.CPF)
	out["cod_empre"] = formatStrList(in.CodEmpre)
	out["vencimento"] = formatIntList(in.Vencimento)
	out["id_cota"] = formatIntList(in.IDCota)
	out["loteria_federal"] = strings.TrimSpace(in.LoteriaFederal)
	out["acrescimo_decrescimo"] = strings.TrimSpace(in.AcrescimoDecrescimo)
	out["tipo_grupo"] = strings.TrimSpace(in.TipoGrupo)
	out["limit"] = fmt.Sprintf("%d", in.Limit)
	out["dry_run"] = formatBool(in.DryRun)
	out["cooldown_user_ms"] = fmt.Sprintf("%d", in.CooldownUserMS)
	out["worker_count_go"] = fmt.Sprintf("%d", in.WorkerCount)
	out["request_timeout_ms"] = fmt.Sprintf("%d", in.RequestTimeoutMS)
	out["token_principal"] = strings.TrimSpace(in.TokenPrincipal)
}

func setSystem(configPath string, out map[string]string, in *Config) {
	databasePath := strings.TrimSpace(in.DatabasePath)
	if databasePath == "" {
		databasePath = "honda.db"
	}
	if filepath.IsAbs(databasePath) {
		configDir := filepath.Dir(configPath)
		if rel, err := filepath.Rel(configDir, databasePath); err == nil && rel != "" {
			databasePath = rel
		}
	}
	out["database_path"] = filepath.ToSlash(databasePath)
	out["api_base_url"] = strings.TrimSpace(in.APIBaseURL)
}

func writeSection(b *strings.Builder, name string, values map[string]string, orderedKeys []string) {
	b.WriteString("[")
	b.WriteString(name)
	b.WriteString("]\n")

	written := map[string]struct{}{}
	for _, key := range orderedKeys {
		if val, ok := values[key]; ok {
			b.WriteString(key)
			b.WriteString(" = ")
			b.WriteString(val)
			b.WriteString("\n")
			written[key] = struct{}{}
		}
	}

	for key, val := range values {
		if _, ok := written[key]; ok {
			continue
		}
		b.WriteString(key)
		b.WriteString(" = ")
		b.WriteString(val)
		b.WriteString("\n")
	}
}

func formatIntList(items []int) string {
	if len(items) == 0 {
		return ""
	}
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, v := range items {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%d", v))
	}
	sb.WriteString("]")
	return sb.String()
}

func formatStrList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, v := range items {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strings.TrimSpace(v))
	}
	sb.WriteString("]")
	return sb.String()
}

func formatBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
