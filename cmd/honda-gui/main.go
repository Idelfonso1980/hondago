package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"honda/go-engine/internal/applog"
	"honda/go-engine/internal/config"
	"honda/go-engine/internal/db"
	"honda/go-engine/internal/engine"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	defaultAPIBaseURL = "https://apigate.hondaservicosfinanceiros.com.br/bff-plataforma-vendas/v1"
	defaultAddr       = "0.0.0.0:8787"
	sessionCookieName = "honda_go_session"
	csrfCookieName    = "honda_go_csrf"
)

var (
	sessionAbsoluteTTL = 8 * time.Hour
	sessionIdleTTL     = 30 * time.Minute
)

var methodPolicy = map[string][]string{
	"/":                                     {http.MethodGet},
	"/api/app/login":                        {http.MethodPost},
	"/api/app/mfa-login":                    {http.MethodPost},
	"/api/app/logout":                       {http.MethodPost},
	"/api/app/session":                      {http.MethodGet},
	"/api/mfa/setup":                        {http.MethodGet},
	"/api/mfa/verify":                       {http.MethodPost},
	"/api/appusers":                         {http.MethodGet},
	"/api/appuser/get":                      {http.MethodGet},
	"/api/appuser/save":                     {http.MethodPost},
	"/api/appuser/delete":                   {http.MethodPost},
	"/api/rbac/matrix":                      {http.MethodGet},
	"/api/rbac/update":                      {http.MethodPost},
	"/api/audit/list":                       {http.MethodGet},
	"/api/config/load":                      {http.MethodGet},
	"/api/config/save":                      {http.MethodPost},
	"/api/run":                              {http.MethodPost},
	"/api/auth":                             {http.MethodPost},
	"/api/auth/users":                       {http.MethodGet},
	"/api/auth/user/get":                    {http.MethodGet},
	"/api/auth/user/find":                   {http.MethodGet},
	"/api/auth/user/save":                   {http.MethodPost},
	"/api/auth/user/delete":                 {http.MethodPost},
	"/api/auth/user/login":                  {http.MethodPost},
	"/api/auth/users/login":                 {http.MethodPost},
	"/api/solicitacoes":                     {http.MethodGet},
	"/api/solicitacoes/minhas":              {http.MethodGet},
	"/api/solicitacoes/get":                 {http.MethodGet},
	"/api/solicitacoes/save":                {http.MethodPost},
	"/api/solicitacoes/last-by-cpf":         {http.MethodGet},
	"/api/solicitacoes/delete":              {http.MethodPost},
	"/api/solicitacoes/reservar":            {http.MethodPost},
	"/api/solicitacoes/reservar-batch":      {http.MethodPost},
	"/api/dashboard/summary":                {http.MethodGet},
	"/api/dashboard/details":                {http.MethodGet},
	"/api/cotasreservadas":                  {http.MethodGet},
	"/api/cotasreservadas/delete":           {http.MethodPost},
	"/api/cotasreservadas/delete-batch":     {http.MethodPost},
	"/api/notificacoes/manual":              {http.MethodGet},
	"/api/notificacoes/manual/status":       {http.MethodPost},
	"/api/available_group_ids":              {http.MethodGet},
	"/api/available_group_ids/get":          {http.MethodGet},
	"/api/available_group_ids/save":         {http.MethodPost},
	"/api/available_group_ids/delete":       {http.MethodPost},
	"/api/available_group_ids/delete-batch": {http.MethodPost},
	"/api/available_group_ids/check":        {http.MethodGet},
	"/api/models":                           {http.MethodGet},
	"/api/models/get":                       {http.MethodGet},
	"/api/models/save":                      {http.MethodPost},
	"/api/models/delete":                    {http.MethodPost},
	"/api/produtos":                         {http.MethodGet},
	"/api/produtos/get":                     {http.MethodGet},
	"/api/produtos/save":                    {http.MethodPost},
	"/api/produtos/delete":                  {http.MethodPost},
	"/api/assembleias":                      {http.MethodGet},
	"/api/assembleias/get":                  {http.MethodGet},
	"/api/assembleias/perclance":            {http.MethodGet},
	"/api/assembleias/save":                 {http.MethodPost},
	"/api/assembleias/delete":               {http.MethodPost},
	"/api/active_groups":                    {http.MethodGet},
	"/api/active_groups/get":                {http.MethodGet},
	"/api/active_groups/parcelas":           {http.MethodGet},
	"/api/active_groups/save":               {http.MethodPost},
	"/api/active_groups/delete":             {http.MethodPost},
	"/api/active_groups/delete-batch":       {http.MethodPost},
	"/api/db/tables/create":                 {http.MethodPost},
	"/api/db/tables/clear":                  {http.MethodPost},
	"/api/db/tables/drop":                   {http.MethodPost},
	"/api/db/backup":                        {http.MethodGet},
	"/api/db/restore":                       {http.MethodPost},
	"/api/db/restore-capabilities":          {http.MethodGet},
	"/api/stop":                             {http.MethodPost},
	"/api/logs":                             {http.MethodGet},
	"/api/logs/clear":                       {http.MethodPost},
	"/api/diagnostics/log-files":            {http.MethodGet},
	"/api/diagnostics/log-file":             {http.MethodGet},
	"/api/diagnostics/log-delete":           {http.MethodPost},
	"/api/diagnostics/log-delete-batch":     {http.MethodPost},
	"/api/status":                           {http.MethodGet},
}

var criticalPermissionPolicy = map[string]string{
	"/api/config/save":                  "configs:manage",
	"/api/run":                          "configs:manage",
	"/api/auth":                         "configs:manage",
	"/api/auth/users":                   "users:read",
	"/api/auth/user/get":                "users:read",
	"/api/auth/user/find":               "users:read",
	"/api/auth/user/save":               "users:edit",
	"/api/auth/user/delete":             "users:delete",
	"/api/auth/user/login":              "users:edit",
	"/api/auth/users/login":             "users:edit",
	"/api/db/tables/create":             "configs:manage",
	"/api/db/tables/clear":              "configs:manage",
	"/api/db/tables/drop":               "configs:manage",
	"/api/db/backup":                    "configs:manage",
	"/api/db/restore":                   "configs:manage",
	"/api/db/restore-capabilities":      "configs:manage",
	"/api/rbac/matrix":                  "roles:manage",
	"/api/rbac/update":                  "roles:manage",
	"/api/audit/list":                   "audit:view",
	"/api/logs":                         "logs:read",
	"/api/logs/clear":                   "logs:delete",
	"/api/diagnostics/log-files":        "logs:read",
	"/api/diagnostics/log-file":         "logs:read",
	"/api/diagnostics/log-delete":       "logs:delete",
	"/api/diagnostics/log-delete-batch": "logs:delete",
}

type rateRule struct {
	Window time.Duration
	Limit  int
}

type fixedWindowRateLimiter struct {
	mu    sync.Mutex
	store map[string]rateState
}

type rateState struct {
	Count      int
	WindowEnds time.Time
}

type dbCapabilities struct {
	BackupAvailable  bool
	RestoreAvailable bool
	BackupReason     string
	RestoreReason    string
}

var endpointRateLimitPolicy = map[string]rateRule{
	"/api/app/login":      {Window: 1 * time.Minute, Limit: 12},
	"/api/app/mfa-login":  {Window: 1 * time.Minute, Limit: 20},
	"/api/mfa/verify":     {Window: 1 * time.Minute, Limit: 20},
	"/api/rbac/update":    {Window: 1 * time.Minute, Limit: 20},
	"/api/db/restore":     {Window: 1 * time.Minute, Limit: 5},
	"/api/db/tables/drop": {Window: 1 * time.Minute, Limit: 5},
}

var utcMinus3Loc = loadUTCMinus3Location()

func loadUTCMinus3Location() *time.Location {
	loc, err := time.LoadLocation("America/Fortaleza")
	if err == nil {
		return loc
	}
	return time.FixedZone("UTC-3", -3*60*60)
}

func nowDateTimeUTCMinus3() string {
	return time.Now().In(utcMinus3Loc).Format("2006-01-02 15:04:05")
}

func parseISODateInUTCMinus3(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, fmt.Errorf("holiday_date vazia")
	}
	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
		"02/01/2006",
		"02/01/2006 15:04",
		"02/01/2006 15:04:05",
	}
	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.ParseInLocation(layout, s, utcMinus3Loc)
		if err == nil {
			return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, utcMinus3Loc), nil
		}
	}
	return time.Time{}, fmt.Errorf("data invalida: %q", s)
}

func lastDayOfMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, utcMinus3Loc).Day()
}

func dueDateForMonth(year int, month time.Month, due_day int64) time.Time {
	day := int(due_day)
	if day < 1 {
		day = 1
	}
	maxDay := lastDayOfMonth(year, month)
	if day > maxDay {
		day = maxDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, utcMinus3Loc)
}

func isBusinessDay(d time.Time, holidays map[string]struct{}) bool {
	wd := d.Weekday()
	if wd == time.Saturday || wd == time.Sunday {
		return false
	}
	key := d.In(utcMinus3Loc).Format("2006-01-02")
	_, isHoliday := holidays[key]
	return !isHoliday
}

func subtractBusinessDays(from time.Time, days int, holidays map[string]struct{}) time.Time {
	d := from
	remaining := days
	for remaining > 0 {
		d = d.AddDate(0, 0, -1)
		if isBusinessDay(d, holidays) {
			remaining--
		}
	}
	return d
}

func monthsBetween(a time.Time, b time.Time) int64 {
	ay, am, _ := a.Date()
	by, bm, _ := b.Date()
	return int64((by-ay)*12 + int(bm-am))
}

func calculateParcelasFromGrupoAtivo(rec *db.GrupoAtivoRecord, holidays map[string]struct{}, now time.Time) int64 {
	if rec == nil || !rec.DataAssembleiaInaugural.Valid {
		return 0
	}
	start, err := parseISODateInUTCMinus3(rec.DataAssembleiaInaugural.String)
	if err != nil {
		return 0
	}
	totalParcelas := rec.Prazo
	if totalParcelas <= 0 {
		totalParcelas = rec.QtdParticipantes
	}
	if totalParcelas <= 0 {
		return 0
	}
	if rec.Vencimento <= 0 {
		return 0
	}
	ref := now.In(utcMinus3Loc)
	due := dueDateForMonth(ref.Year(), ref.Month(), rec.Vencimento)
	cutoff := subtractBusinessDays(due, 0, holidays)
	cycleRef := time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, utcMinus3Loc)
	if !ref.Before(cutoff) {
		cycleRef = cycleRef.AddDate(0, 1, 0)
	}
	startMonth := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, utcMinus3Loc)
	elapsed := monthsBetween(startMonth, cycleRef)
	if elapsed < 0 {
		elapsed = 0
	}
	remaining := totalParcelas - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

type app struct {
	mu            sync.Mutex
	configPath    string
	cookieSecure  bool
	limiter       *fixedWindowRateLimiter
	running       bool
	cancel        context.CancelFunc
	logBuffer     *ringBuffer
	startedAt     time.Time
	finishedAt    time.Time
	mfaTempTokens map[string]mfaTempToken
	store         *db.Store
	cfg           *config.Config
}

type mfaTempToken struct {
	UserID    int64
	ExpiresAt time.Time
}

type appSession struct {
	Token           string
	UserID          int64
	Username        string
	DisplayName     string
	CPF             string
	branch          string
	Role            string
	Permissions     []string
	AuthenticatedAt time.Time
}

type runResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type uiPayload struct {
	ConfigPath          string `json:"config_path"`
	DatabasePath        string `json:"database_path"`
	APIBaseURL          string `json:"api_base_url"`
	ReservarModo        string `json:"reservar_modo"`
	Modelo              string `json:"model_name"`
	Produto             string `json:"produto"`
	Grupo               string `json:"group_code"`
	CPF                 string `json:"cpf"`
	CodEmpre            string `json:"cod_empre"`
	Vencimento          string `json:"due_day"`
	IDCota              string `json:"requested_quota_id"`
	LoteriaFederal      string `json:"federal_lottery"`
	AcrescimoDecrescimo string `json:"acrescimo_decrescimo"`
	TipoGrupo           string `json:"group_type"`
	Limit               string `json:"limit"`
	DryRun              bool   `json:"dry_run"`
	CooldownUserMS      string `json:"cooldown_user_ms"`
	WorkerCount         string `json:"worker_count_go"`
	RequestTimeoutMS    string `json:"request_timeout_ms"`
}

type authUserPayload struct {
	ID                int64  `json:"id"`
	Search            string `json:"search,omitempty"`
	CPF               string `json:"cpf"`
	CodEmpresa        string `json:"company_code"`
	CodUsuario        string `json:"account_user"`
	CodConcessionaria string `json:"dealer_code"`
	Senha             string `json:"account_password"`
	Token             string `json:"token,omitempty"`
	TokenB3           string `json:"b3_token,omitempty"`
	LastRequest       string `json:"last_request_at,omitempty"`
	CooldownUntil     string `json:"cooldown_until,omitempty"`
	BlockedUntil      string `json:"blocked_until,omitempty"`
	InFlight          int64  `json:"in_flight,omitempty"`
	Error401Count     int64  `json:"error_401_count,omitempty"`
	Error429Count     int64  `json:"error_429_count,omitempty"`
	PriorityScore     string `json:"priority_score,omitempty"`
}

type authIDsPayload struct {
	IDs []int64 `json:"ids"`
}

type reservedCotaPayload struct {
	ID                 int64  `json:"id"`
	UsuarioReserva     string `json:"usuario_reserva,omitempty"`
	NumDocumentoPessoa string `json:"num_documento_pessoa,omitempty"`
	CodGrupo           string `json:"cod_grupo,omitempty"`
	CotaRD             string `json:"cota_rd,omitempty"`
	CodModelo          string `json:"cod_modelo,omitempty"`
	IDCotaReposicao    string `json:"id_cota_reposicao,omitempty"`
	CreatedAt          string `json:"created_at,omitempty"`
}

type reservedCotaIDsPayload struct {
	IDs []int64 `json:"ids"`
}

type solicitacaoPayload struct {
	ID                  int64  `json:"id"`
	Search              string `json:"search,omitempty"`
	DataHoraSolicitacao string `json:"requested_at"`
	Filial              string `json:"branch"`
	Vendedor            string `json:"seller_name"`
	CPF                 string `json:"cpf"`
	Modelo              string `json:"model_name"`
	Plano               string `json:"plan"`
	QtdeParcelas        string `json:"installments"`
	PercLance           string `json:"bid_percent"`
	ComRestricao        string `json:"with_restriction"`
	Grupo               string `json:"group_code"`
	QtdSolicitada       string `json:"qtd_solicitada"`
	Notes               string `json:"notes"`
	IDCota              string `json:"requested_quota_id"`
	GrupoAtendido       string `json:"served_group"`
	CotaRD              string `json:"quota_rd"`
	DataHoraAtendimento string `json:"served_at"`
	Situacao            string `json:"status"`
	LanceContemplacao   string `json:"contemplation_bid"`
}

type solicitacaoIDsPayload struct {
	IDs []int64 `json:"ids"`
}

type solicitacaoSuccessItemPayload struct {
	ID            int64  `json:"id"`
	Nome          string `json:"nome"`
	Filial        string `json:"branch"`
	CPF           string `json:"cpf,omitempty"`
	GrupoAtendido string `json:"served_group"`
	CotaRD        string `json:"cota_rd"`
	QtdeParcelas  string `json:"installments,omitempty"`
	Modelo        string `json:"model_name"`
	Licenciada    string `json:"licensed,omitempty"`
	ComRestricao  string `json:"with_restriction,omitempty"`
	SolicitadaEm  string `json:"requested_at,omitempty"`
	AtendidaEm    string `json:"served_at,omitempty"`
	SLA           string `json:"sla,omitempty"`
}

type manualNotificationPayload struct {
	ID            int64  `json:"id"`
	SolicitacaoID int64  `json:"solicitacao_id,omitempty"`
	CPF           string `json:"cpf,omitempty"`
	Vendedor      string `json:"seller_name,omitempty"`
	Filial        string `json:"branch,omitempty"`
	Canal         string `json:"channel,omitempty"`
	Mensagem      string `json:"message,omitempty"`
	Status        string `json:"status,omitempty"`
	CopiadaEm     string `json:"copied_at,omitempty"`
	EnviadaEm     string `json:"sent_at,omitempty"`
	UsuarioAcao   string `json:"action_user,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
}

type manualNotificationIDsPayload struct {
	IDs    []int64 `json:"ids"`
	Status string  `json:"status"`
}

type dashboardRankItem struct {
	Nome      string  `json:"nome"`
	Total     int64   `json:"total"`
	Atendidas int64   `json:"atendidas"`
	Taxa      float64 `json:"taxa"`
}

type dashboardSerieItem struct {
	Data        string `json:"holiday_date"`
	Solicitadas int64  `json:"solicitadas"`
	Atendidas   int64  `json:"atendidas"`
}

type dashboardHourItem struct {
	Hora        string `json:"hora"`
	Solicitadas int64  `json:"solicitadas"`
	Atendidas   int64  `json:"atendidas"`
}

type dashboardMetricCompare struct {
	Atual    float64 `json:"atual"`
	Anterior float64 `json:"anterior"`
	DeltaPct float64 `json:"delta_pct"`
}

type idsGrupoDisponivelPayload struct {
	ID            int64  `json:"id"`
	IDGrupo       int64  `json:"id_grupo"`
	Produto       string `json:"produto"`
	Vencimento    int64  `json:"due_day"`
	Prazo         int64  `json:"term_months"`
	Tipo          string `json:"group_kind"`
	Grupo         int64  `json:"group_code"`
	Cota          int64  `json:"quota"`
	R             int64  `json:"r"`
	D             int64  `json:"d"`
	ParcelasCalc  int64  `json:"parcelas_calc"`
	Booked        int64  `json:"booked"`
	CreatedAt     string `json:"created_at"`
	Participantes int64  `json:"participants"`
	Failed        int64  `json:"failed"`
}

type idsGrupoDisponivelIDsPayload struct {
	IDs []int64 `json:"ids"`
}

type modeloPayload struct {
	ID       int64  `json:"id"`
	IDModelo int64  `json:"model_api_id"`
	Modelo   string `json:"model_name"`
	Status   string `json:"status"`
}

type produtoPayload struct {
	ID        int64  `json:"id"`
	IDProduto int64  `json:"product_api_id"`
	Produto   string `json:"produto"`
	Status    string `json:"status"`
}

type assembleiaPayload struct {
	ID                 int64  `json:"id"`
	CotaRD             string `json:"quota_rd"`
	DataContemplacao   string `json:"contemplation_date"`
	TipoContemplacao   string `json:"contemplation_type"`
	DataDesclassificao string `json:"disqualification_date"`
	ClientName         string `json:"client_name"`
	PercLance          string `json:"bid_percent"`
	Vendedor           string `json:"seller_name"`
	Grupo              string `json:"group_code"`
	LoteriaFederal     string `json:"federal_lottery"`
	GrupoCotaRD        string `json:"group_quota_rd"`
}

type grupoAtivoPayload struct {
	ID                      int64  `json:"id"`
	Grupo                   int64  `json:"group_code"`
	Vencimento              int64  `json:"due_day"`
	QtdParticipantes        int64  `json:"participants_count"`
	PercLance               string `json:"bid_percent"`
	DataAssembleiaInaugural string `json:"first_assembly_date"`
	Plano                   string `json:"plan"`
	Prazo                   int64  `json:"term_months"`
	TipoGrupo               string `json:"group_type"`
	Modelos                 string `json:"modelos"`
	Status                  string `json:"status"`
	ParcelasCalculadas      int64  `json:"parcelas_calculadas"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
}

type grupoAtivoIDsPayload struct {
	IDs []int64 `json:"ids"`
}

type appLoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type appUserPayload struct {
	ID                  int64  `json:"id"`
	Search              string `json:"search,omitempty"`
	Username            string `json:"username"`
	DisplayName         string `json:"full_name"`
	Manager             string `json:"manager,omitempty"`
	Supervisor          string `json:"supervisor,omitempty"`
	CPF                 string `json:"cpf,omitempty"`
	Filial              string `json:"branch,omitempty"`
	Email               string `json:"email,omitempty"`
	Phone               string `json:"phone,omitempty"`
	MFAEnabled          bool   `json:"mfa_enabled"`
	MFASecret           string `json:"mfa_secret,omitempty"`
	Role                string `json:"role"`
	IsActive            bool   `json:"is_active"`
	Password            string `json:"password,omitempty"`
	FailedLoginAttempts int64  `json:"failed_login_attempts,omitempty"`
	LockedUntil         string `json:"locked_until,omitempty"`
	LastLoginAt         string `json:"last_login_at,omitempty"`
	UpdatedAt           string `json:"updated_at,omitempty"`
	CreatedAt           string `json:"created_at,omitempty"`
}

type diagnosticLogFile struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	SizeBytes  int64  `json:"size_bytes"`
}

type diagnosticLogNamesPayload struct {
	Names []string `json:"names"`
}

type ringBuffer struct {
	mu    sync.Mutex
	max   int
	lines []string
}

func newRingBuffer(max int) *ringBuffer {
	return &ringBuffer{max: max}
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = append(r.lines, string(p))
	if len(r.lines) > r.max {
		r.lines = r.lines[len(r.lines)-r.max:]
	}
	return len(p), nil
}

func (r *ringBuffer) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.lines, "")
}

func (r *ringBuffer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = nil
}

func main() {
	app := &app{
		configPath:    "config.ini",
		cookieSecure:  defaultCookieSecure(),
		limiter:       newFixedWindowRateLimiter(),
		logBuffer:     newRingBuffer(1000),
		mfaTempTokens: make(map[string]mfaTempToken),
	}
	defer app.closeStore()

	// Adiciona um flag para o endereÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§o de escuta
	listenAddr := flag.String("addr", defaultAddr, "EndereÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§o e porta para a GUI escutar (ex: 0.0.0.0:8787)")
	flag.Parse()

	mux := http.NewServeMux()
	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("falha ao carregar statics: %v", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", app.handleIndex)
	mux.HandleFunc("/api/app/login", app.handleAppLogin)
	mux.HandleFunc("/api/app/mfa-login", app.handleAppMFALogin)
	mux.HandleFunc("/api/app/logout", app.handleAppLogout)
	mux.HandleFunc("/api/app/session", app.handleAppSession)
	mux.HandleFunc("/api/mfa/setup", app.handleMFASetup)
	mux.HandleFunc("/api/mfa/verify", app.handleMFAVerify)
	mux.HandleFunc("/api/appusers", app.handleAppUsersSearch)
	mux.HandleFunc("/api/appuser/get", app.handleAppUserGet)
	mux.HandleFunc("/api/appuser/save", app.handleAppUserSave)
	mux.HandleFunc("/api/appuser/delete", app.handleAppUserDelete)
	mux.HandleFunc("/api/rbac/matrix", app.handleRBACMatrixGet)
	mux.HandleFunc("/api/rbac/update", app.handleRBACMatrixUpdate)
	mux.HandleFunc("/api/audit/list", app.handleAuditList)
	mux.HandleFunc("/api/config/load", app.handleLoadConfig)
	mux.HandleFunc("/api/config/save", app.handleSaveConfig)
	mux.HandleFunc("/api/run", app.handleRun)
	mux.HandleFunc("/api/auth", app.handleAuth)
	mux.HandleFunc("/api/auth/users", app.handleAuthUsersSearch)
	mux.HandleFunc("/api/auth/user/get", app.handleAuthUserGet)
	mux.HandleFunc("/api/auth/user/find", app.handleAuthUserFind)
	mux.HandleFunc("/api/auth/user/save", app.handleAuthUserSave)
	mux.HandleFunc("/api/auth/user/delete", app.handleAuthUserDelete)
	mux.HandleFunc("/api/auth/user/login", app.handleAuthUserLogin)
	mux.HandleFunc("/api/auth/users/login", app.handleAuthUsersLogin)
	mux.HandleFunc("/api/solicitacoes", app.handleSolicitacoesSearch)
	mux.HandleFunc("/api/solicitacoes/minhas", app.handleMinhasSolicitacoesSearch)
	mux.HandleFunc("/api/solicitacoes/get", app.handleSolicitacaoGet)
	mux.HandleFunc("/api/solicitacoes/save", app.handleSolicitacaoSave)
	mux.HandleFunc("/api/solicitacoes/last-by-cpf", app.handleSolicitacaoLastByCPF)
	mux.HandleFunc("/api/solicitacoes/delete", app.handleSolicitacaoDelete)
	mux.HandleFunc("/api/solicitacoes/reservar", app.handleSolicitacaoReservar)
	mux.HandleFunc("/api/solicitacoes/reservar-batch", app.handleSolicitacoesReservarBatch)
	mux.HandleFunc("/api/dashboard/summary", app.handleDashboardSummary)
	mux.HandleFunc("/api/dashboard/details", app.handleDashboardDetails)
	mux.HandleFunc("/api/cotasreservadas", app.handleReservedCotasSearch)
	mux.HandleFunc("/api/cotasreservadas/delete", app.handleReservedCotaDelete)
	mux.HandleFunc("/api/cotasreservadas/delete-batch", app.handleReservedCotasDeleteBatch)
	mux.HandleFunc("/api/notificacoes/manual", app.handleManualNotificationsSearch)
	mux.HandleFunc("/api/notificacoes/manual/status", app.handleManualNotificationsSetStatus)
	mux.HandleFunc("/api/available_group_ids", app.handleIDsGruposDisponiveisSearch)
	mux.HandleFunc("/api/available_group_ids/get", app.handleIDsGrupoDisponivelGet)
	mux.HandleFunc("/api/available_group_ids/save", app.handleIDsGrupoDisponivelSave)
	mux.HandleFunc("/api/available_group_ids/delete", app.handleIDsGrupoDisponivelDelete)
	mux.HandleFunc("/api/available_group_ids/delete-batch", app.handleIDsGruposDisponiveisDeleteBatch)
	mux.HandleFunc("/api/available_group_ids/check", app.handleCheckAvailableGroup)
	mux.HandleFunc("/api/models", app.handleModelosSearch)
	mux.HandleFunc("/api/models/get", app.handleModeloGet)
	mux.HandleFunc("/api/models/save", app.handleModeloSave)
	mux.HandleFunc("/api/models/delete", app.handleModeloDelete)
	mux.HandleFunc("/api/produtos", app.handleProdutosSearch)
	mux.HandleFunc("/api/produtos/get", app.handleProdutoGet)
	mux.HandleFunc("/api/produtos/save", app.handleProdutoSave)
	mux.HandleFunc("/api/produtos/delete", app.handleProdutoDelete)
	mux.HandleFunc("/api/assembleias", app.handleAssembleiasSearch)
	mux.HandleFunc("/api/assembleias/get", app.handleAssembleiaGet)
	mux.HandleFunc("/api/assembleias/perclance", app.handleAssembleiaPercLance)
	mux.HandleFunc("/api/assembleias/save", app.handleAssembleiaSave)
	mux.HandleFunc("/api/assembleias/delete", app.handleAssembleiaDelete)
	mux.HandleFunc("/api/active_groups", app.handleGruposAtivosSearch)
	mux.HandleFunc("/api/active_groups/get", app.handleGrupoAtivoGet)
	mux.HandleFunc("/api/active_groups/parcelas", app.handleGrupoAtivoParcelas)
	mux.HandleFunc("/api/active_groups/save", app.handleGrupoAtivoSave)
	mux.HandleFunc("/api/active_groups/delete", app.handleGrupoAtivoDelete)
	mux.HandleFunc("/api/active_groups/delete-batch", app.handleGruposAtivosDeleteBatch)
	mux.HandleFunc("/api/db/tables/create", app.handleDBTablesCreate)
	mux.HandleFunc("/api/db/tables/clear", app.handleDBTablesClear)
	mux.HandleFunc("/api/db/tables/drop", app.handleDBTablesDrop)
	mux.HandleFunc("/api/db/backup", app.handleDBBackup)
	mux.HandleFunc("/api/db/restore", app.handleDBRestore)
	mux.HandleFunc("/api/db/restore-capabilities", app.handleDBRestoreCapabilities)
	mux.HandleFunc("/api/stop", app.handleStop)
	mux.HandleFunc("/api/logs", app.handleLogs)
	mux.HandleFunc("/api/logs/clear", app.handleClearLogs)
	mux.HandleFunc("/api/diagnostics/log-files", app.handleDiagnosticLogFiles)
	mux.HandleFunc("/api/diagnostics/log-file", app.handleDiagnosticLogFile)
	mux.HandleFunc("/api/diagnostics/log-delete", app.handleDiagnosticLogDelete)
	mux.HandleFunc("/api/diagnostics/log-delete-batch", app.handleDiagnosticLogDeleteBatch)
	mux.HandleFunc("/api/status", app.handleStatus)

	server := &http.Server{
		Addr:              *listenAddr, // Usa o endereÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§o do flag
		Handler:           app.withAuth(mux),
		ReadHeaderTimeout: 15 * time.Second,
	}

	url := "http://" + *listenAddr
	log.Printf("[gui] interface web em %s", url)

	browserUrl := url
	if strings.HasPrefix(*listenAddr, "0.0.0.0:") {
		browserUrl = "http://localhost:" + strings.TrimPrefix(*listenAddr, "0.0.0.0:")
	}
	go openBrowser(browserUrl)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("gui server: %v", err)
	}
}

//go:embed web/*
var webFS embed.FS

func (a *app) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	content, err := webFS.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func (a *app) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applySecurityHeaders(w)
		path := r.URL.Path
		if !isMethodAllowed(path, r.Method) {
			writeJSON(w, http.StatusMethodNotAllowed, runResponse{OK: false, Message: "Metodo nao permitido"})
			return
		}
		if rule, ok := endpointRateLimitPolicy[path]; ok && a.limiter != nil {
			key := clientIPKey(r) + "|" + path
			if !a.limiter.Allow(key, rule) {
				w.Header().Set("Retry-After", "60")
				writeJSON(w, http.StatusTooManyRequests, runResponse{OK: false, Message: "Muitas requisicoes, tente novamente"})
				return
			}
		}
		if path == "/" || path == "/api/app/login" || path == "/api/app/session" || path == "/api/app/mfa-login" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(path, "/api/") {
			sess, ok := a.currentSession(r)
			if !ok {
				writeJSON(w, http.StatusUnauthorized, runResponse{OK: false, Message: "Nao autenticado"})
				return
			}
			if requiredPerm := strings.TrimSpace(criticalPermissionPolicy[path]); requiredPerm != "" && !sessionHasPermission(sess, requiredPerm) {
				writeJSON(w, http.StatusForbidden, runResponse{OK: false, Message: "Acesso negado: permissao necessaria: " + requiredPerm})
				return
			}
			if isMutatingMethod(r.Method) && !isCSRFAuthorized(r) {
				writeJSON(w, http.StatusForbidden, runResponse{OK: false, Message: "CSRF token invalido"})
				return
			}
			a.setSessionCookie(w, r, sess.Token)
			a.setCSRFCookie(w, r, readCSRFCookieValue(r))
		}
		next.ServeHTTP(w, r)
	})
}

func defaultCookieSecure() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("HONDAGO_COOKIE_SECURE")))
	switch raw {
	case "0", "false", "no", "n":
		return false
	case "1", "true", "yes", "y":
		return true
	default:
		return true
	}
}

func applySecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
}

func newFixedWindowRateLimiter() *fixedWindowRateLimiter {
	return &fixedWindowRateLimiter{
		store: make(map[string]rateState),
	}
}

func (l *fixedWindowRateLimiter) Allow(key string, rule rateRule) bool {
	if l == nil || strings.TrimSpace(key) == "" || rule.Limit <= 0 || rule.Window <= 0 {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	state, ok := l.store[key]
	if !ok || now.After(state.WindowEnds) {
		l.store[key] = rateState{Count: 1, WindowEnds: now.Add(rule.Window)}
		return true
	}
	if state.Count >= rule.Limit {
		return false
	}
	state.Count++
	l.store[key] = state
	return true
}

func clientIPKey(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	realIP := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	hostPort := strings.TrimSpace(r.RemoteAddr)
	if hostPort == "" {
		return "unknown"
	}
	if host, _, err := net.SplitHostPort(hostPort); err == nil && strings.TrimSpace(host) != "" {
		return host
	}
	return hostPort
}

func sessionHasPermission(sess *appSession, permission string) bool {
	if sess == nil {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(sess.Role))
	if role == "admin" {
		return true
	}
	for _, p := range sess.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func isMethodAllowed(path, method string) bool {
	if strings.HasPrefix(path, "/static/") {
		return method == http.MethodGet
	}
	allowed, ok := methodPolicy[path]
	if !ok {
		if strings.HasPrefix(path, "/api/") {
			return false
		}
		return method == http.MethodGet
	}
	for _, m := range allowed {
		if method == m {
			return true
		}
	}
	return false
}

func isMutatingMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func isCSRFAuthorized(r *http.Request) bool {
	c, err := r.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(c.Value) == "" {
		return false
	}
	headerToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	if headerToken == "" || len(headerToken) != len(c.Value) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(headerToken), []byte(c.Value)) == 1
}

func readCSRFCookieValue(r *http.Request) string {
	c, err := r.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(c.Value) == "" {
		token, _ := newSessionToken()
		return token
	}
	return c.Value
}

func isHTTPSRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func newSessionToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (a *app) currentSession(r *http.Request) (*appSession, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || strings.TrimSpace(c.Value) == "" {
		return nil, false
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		return nil, false
	}

	s, err := store.GetAppSession(r.Context(), c.Value)
	if err != nil {
		return nil, false
	}
	if s.AuthenticatedAt.IsZero() || time.Since(s.AuthenticatedAt) > sessionAbsoluteTTL {
		_ = store.DeleteAppSession(r.Context(), c.Value)
		return nil, false
	}

	return &appSession{
		Token:           s.Token,
		UserID:          s.UserID,
		Username:        s.Username,
		DisplayName:     s.DisplayName,
		CPF:             s.CPF.String,
		branch:          s.Filial.String,
		Role:            s.Role,
		Permissions:     s.Permissions,
		AuthenticatedAt: s.AuthenticatedAt,
	}, true
}

func (a *app) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	s, ok := a.currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, runResponse{OK: false, Message: "Nao autenticado"})
		return false
	}
	role := strings.ToLower(strings.TrimSpace(s.Role))
	if role != "admin" {
		log.Printf("[SECURITY] ALERTA: Usuario '%s' (Perfil: %s) tentou acessar endpoint restrito de ADMIN: %s", s.Username, s.Role, r.URL.Path)
		writeJSON(w, http.StatusForbidden, runResponse{OK: false, Message: "Acesso negado: perfil sem permissao"})
		return false
	}
	return true
}

func (a *app) requirePermission(w http.ResponseWriter, r *http.Request, permission string) bool {
	s, ok := a.currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, runResponse{OK: false, Message: "Nao autenticado"})
		return false
	}

	role := strings.ToLower(strings.TrimSpace(s.Role))
	if role == "admin" {
		return true // Admin bypass
	}

	for _, p := range s.Permissions {
		if p == permission {
			return true
		}
	}

	log.Printf("[SECURITY] Acesso Negado: Usuario '%s' (Perfil: %s) tentou '%s' em %s", s.Username, s.Role, permission, r.URL.Path)
	writeJSON(w, http.StatusForbidden, runResponse{OK: false, Message: "Acesso negado: permissao necessaria: " + permission})
	return false
}

func (a *app) logAudit(r *http.Request, action, entity, entityID, before, after string) {
	s, ok := a.currentSession(r)
	if !ok {
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		log.Printf("[AUDIT ERROR] Falha ao abrir banco para log: %v", err)
		return
	}

	audit := db.AuditLogRecord{
		Username:    s.Username,
		Action:      action,
		Entity:      entity,
		EntityID:    entityID,
		BeforeState: before,
		AfterState:  after,
	}
	if err := store.CreateAuditLog(r.Context(), audit); err != nil {
		log.Printf("[AUDIT ERROR] Falha ao criar log: %v", err)
	}
}

func (a *app) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := a.cookieSecure && isHTTPSRequest(r)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionIdleTTL / time.Second),
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *app) setCSRFCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := a.cookieSecure && isHTTPSRequest(r)
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionIdleTTL / time.Second),
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *app) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func appUserToPayload(r *db.AppUserRecord) appUserPayload {
	out := appUserPayload{
		ID:                  r.ID,
		Username:            r.Username,
		DisplayName:         r.DisplayName,
		Manager:             strings.TrimSpace(r.Manager.String),
		Supervisor:          strings.TrimSpace(r.Supervisor.String),
		CPF:                 strings.TrimSpace(r.CPF.String),
		Filial:              strings.TrimSpace(r.Filial.String),
		Role:                r.Role,
		IsActive:            r.IsActive != 0,
		FailedLoginAttempts: r.FailedLoginAttempts,
		LockedUntil:         nullTimeStr(r.LockedUntil),
		LastLoginAt:         nullTimeStr(r.LastLoginAt),
		UpdatedAt:           nullTimeStr(r.UpdatedAt),
		CreatedAt:           nullTimeStr(r.CreatedAt),
	}
	if r.Email.Valid {
		out.Email = r.Email.String
	}
	if r.Phone.Valid {
		out.Phone = r.Phone.String
	}
	out.MFAEnabled = r.MFAEnabled != 0
	if r.MFASecret.Valid {
		out.MFASecret = r.MFASecret.String
	}
	return out
}

func (a *app) handleAppLogin(w http.ResponseWriter, r *http.Request) {
	var p appLoginPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	username := strings.TrimSpace(p.Username)
	password := strings.TrimSpace(p.Password)
	if username == "" || password == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("usuario e account_password obrigatorios"))
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		log.Printf("[SECURITY] login open db error: %v", err)
		writeErr(w, http.StatusServiceUnavailable, fmt.Errorf("servico temporariamente indisponivel"))
		return
	}

	user, err := store.FindAppUserByUsername(r.Context(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusUnauthorized, fmt.Errorf("usuario ou account_password invalidos"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if user.IsActive == 0 {
		writeErr(w, http.StatusForbidden, fmt.Errorf("usuario inativo"))
		return
	}
	if user.LockedUntil.Valid && user.LockedUntil.Time.After(time.Now()) {
		writeErr(w, http.StatusForbidden, fmt.Errorf("usuario temporariamente bloqueado atÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â© %s", user.LockedUntil.Time.Format("2006-01-02 15:04:05")))
		return
	}
	if !db.VerifyPassword(user.PasswordHash, password) {
		_ = store.RegisterAppUserLoginFailure(r.Context(), user.ID)
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("usuario ou account_password invalidos"))
		return
	}
	if err := store.RegisterAppUserLoginSuccess(r.Context(), user.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	if user.MFAEnabled == 1 && user.MFASecret.Valid {
		tempToken, _ := newSessionToken()
		a.mu.Lock()
		a.mfaTempTokens[tempToken] = mfaTempToken{
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(5 * time.Minute),
		}
		a.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":           true,
			"mfa_required": true,
			"temp_token":   tempToken,
		})
		return
	}

	token, err := newSessionToken()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	display := strings.TrimSpace(user.DisplayName)
	if display == "" {
		display = user.Username
	}

	perms, _ := store.GetRolePermissions(r.Context(), user.Role)
	if perms == nil {
		perms = []string{}
	}

	sessRecord := &db.AppSessionRecord{
		Token:           token,
		UserID:          user.ID,
		Username:        user.Username,
		DisplayName:     display,
		CPF:             user.CPF,
		Filial:          user.Filial,
		Role:            user.Role,
		Permissions:     perms,
		AuthenticatedAt: time.Now(),
	}

	if err := store.SaveAppSession(r.Context(), sessRecord); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	a.setSessionCookie(w, r, token)
	a.setCSRFCookie(w, r, readCSRFCookieValue(r))

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"message":     "Login realizado",
		"full_name":   display,
		"username":    user.Username,
		"cpf":         strings.TrimSpace(user.CPF.String),
		"branch":      strings.TrimSpace(user.Filial.String),
		"role":        user.Role,
		"permissions": perms,
	})
}

func (a *app) handleAppLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(sessionCookieName)
	if err == nil && strings.TrimSpace(c.Value) != "" {
		_, store, err := a.openStoreFromCurrentConfig()
		if err == nil {
			_ = store.DeleteAppSession(r.Context(), c.Value)
		}
	}
	a.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "SessÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o encerrada"})
}

func (a *app) handleAppSession(w http.ResponseWriter, r *http.Request) {
	s, ok := a.currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, runResponse{OK: false, Message: "NÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o autenticado"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"authenticated": true,
		"user_id":       s.UserID,
		"username":      s.Username,
		"full_name":     s.DisplayName,
		"cpf":           s.CPF,
		"branch":        s.branch,
		"role":          s.Role,
		"permissions":   s.Permissions,
	})
}

func (a *app) handleAppUsersSearch(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	items, err := store.SearchAppUsers(r.Context(), q, 500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]appUserPayload, 0, len(items))
	for i := range items {
		out = append(out, appUserToPayload(&items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func (a *app) handleAppUserGet(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	item, err := store.GetAppUserByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("usuario nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, appUserToPayload(item))
}

func (a *app) handleAppUserSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	var p appUserPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	rec := db.AppUserRecord{
		ID:          p.ID,
		Username:    strings.TrimSpace(p.Username),
		DisplayName: strings.TrimSpace(p.DisplayName),
		Manager:     sql.NullString{String: strings.TrimSpace(p.Manager), Valid: strings.TrimSpace(p.Manager) != ""},
		Supervisor:  sql.NullString{String: strings.TrimSpace(p.Supervisor), Valid: strings.TrimSpace(p.Supervisor) != ""},
		CPF:         sql.NullString{String: onlyDigitsLocal(p.CPF), Valid: strings.TrimSpace(p.CPF) != ""},
		Filial:      sql.NullString{String: strings.TrimSpace(p.Filial), Valid: strings.TrimSpace(p.Filial) != ""},
		Email:       sql.NullString{String: strings.TrimSpace(p.Email), Valid: strings.TrimSpace(p.Email) != ""},
		Phone:       sql.NullString{String: onlyDigitsLocal(p.Phone), Valid: strings.TrimSpace(p.Phone) != ""},
		MFASecret:   sql.NullString{String: strings.TrimSpace(p.MFASecret), Valid: strings.TrimSpace(p.MFASecret) != ""},
		Role:        strings.TrimSpace(p.Role),
		IsActive:    0,
	}
	if p.MFAEnabled {
		rec.MFAEnabled = 1
	}
	if p.IsActive {
		rec.IsActive = 1
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	newID, err := store.SaveAppUser(r.Context(), rec, p.Password)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	// Auditoria
	action := "UPDATE_USER"
	if p.ID <= 0 {
		action = "CREATE_USER"
	}
	a.logAudit(r, action, "users", p.Username, "", fmt.Sprintf("Role: %s, Active: %v", p.Role, p.IsActive))

	item, err := store.GetAppUserByID(r.Context(), newID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	msg := "Usuario do sistema atualizado"
	if p.ID <= 0 {
		msg = "Usuario do sistema criado"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": msg,
		"user":    appUserToPayload(item),
		"id":      newID,
	})
}

func (a *app) handleAppUserDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requirePermission(w, r, "users:delete") {
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := store.DeleteAppUser(r.Context(), id); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	// Auditoria
	a.logAudit(r, "DELETE_USER", "users", strconv.FormatInt(id, 10), "", "")

	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Usuario do sistema removido"})
}

func (a *app) handleLoadConfig(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		a.mu.Lock()
		path = a.configPath
		a.mu.Unlock()
	}
	cfg, err := config.LoadFromINI(path)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	out := map[string]any{
		"config_path":          path,
		"database_path":        cfg.DatabasePath,
		"api_base_url":         cfg.APIBaseURL,
		"reservar_modo":        cfg.Booking.ReservarModo,
		"model_name":           cfg.Booking.Modelo,
		"products":             joinInt(cfg.Booking.Produto),
		"group_code":           joinInt(cfg.Booking.Grupo),
		"cpf":                  strings.Join(cfg.Booking.CPF, ","),
		"cod_empre":            strings.Join(cfg.Booking.CodEmpre, ","),
		"due_day":              joinInt(cfg.Booking.Vencimento),
		"requested_quota_id":   joinInt(cfg.Booking.IDCota),
		"federal_lottery":      cfg.Booking.LoteriaFederal,
		"acrescimo_decrescimo": cfg.Booking.AcrescimoDecrescimo,
		"group_type":           cfg.Booking.TipoGrupo,
		"limit":                strconv.Itoa(cfg.Booking.Limit),
		"dry_run":              cfg.Booking.DryRun,
		"cooldown_user_ms":     strconv.Itoa(cfg.Booking.CooldownUserMS),
		"worker_count_go":      strconv.Itoa(cfg.Booking.WorkerCount),
		"request_timeout_ms":   strconv.Itoa(cfg.Booking.RequestTimeoutMS),
	}

	a.mu.Lock()
	a.configPath = path
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var p uiPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	path := strings.TrimSpace(p.ConfigPath)
	if path == "" {
		path = "config.ini"
	}

	cfg, err := a.payloadToConfig(path, &p)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := config.SaveToINI(path, cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	// Auditoria
	a.logAudit(r, "UPDATE_CONFIG", "config", path, "", fmt.Sprintf("Mode: %s, API: %s", p.ReservarModo, p.APIBaseURL))

	a.mu.Lock()
	a.configPath = path
	a.mu.Unlock()
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Config salva"})
}

func (a *app) handleRun(w http.ResponseWriter, r *http.Request) {
	var p uiPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	path := strings.TrimSpace(p.ConfigPath)
	if path == "" {
		path = "config.ini"
	}

	cfg, err := a.payloadToConfig(path, &p)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := config.SaveToINI(path, cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		writeJSON(w, http.StatusConflict, runResponse{OK: false, Message: "Execucao ja em andamento"})
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.running = true
	a.configPath = path
	a.startedAt = time.Now()
	a.finishedAt = time.Time{}
	a.mu.Unlock()

	go a.runEngine(ctx, path)
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Execucao iniciada"})
}

func (a *app) handleAuth(w http.ResponseWriter, r *http.Request) {
	var p uiPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	path := strings.TrimSpace(p.ConfigPath)
	if path == "" {
		path = "config.ini"
	}

	cfg, err := a.payloadToConfig(path, &p)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := config.SaveToINI(path, cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		writeJSON(w, http.StatusConflict, runResponse{OK: false, Message: "Execucao ja em andamento"})
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.running = true
	a.configPath = path
	a.startedAt = time.Now()
	a.finishedAt = time.Time{}
	a.mu.Unlock()

	go a.runAuth(ctx, path)
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "AutenticaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o iniciada"})
}

func (a *app) handleAuthUsersSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	records, err := store.SearchAuthRecords(r.Context(), search, 500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]authUserPayload, 0, len(records))
	for i := range records {
		out = append(out, authRecordToPayload(&records[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "count": len(out)})
}

func (a *app) handleAuthUserGet(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	record, err := store.GetAuthRecordByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("usuario nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, authRecordToPayload(record))
}

func (a *app) handleAuthUserFind(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	if search == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("campo de busca obrigatÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â³rio"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	record, err := store.FindAuthRecord(r.Context(), search)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("usuario nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, authRecordToPayload(record))
}

func (a *app) handleAuthUserSave(w http.ResponseWriter, r *http.Request) {
	var p authUserPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	rec := db.AuthRecord{
		ID:                p.ID,
		CPF:               onlyDigitsLocal(p.CPF),
		CodEmpresa:        p.CodEmpresa,
		CodUsuario:        p.CodUsuario,
		CodConcessionaria: p.CodConcessionaria,
		Senha:             strings.TrimSpace(p.Senha),
	}
	if strings.TrimSpace(rec.CodConcessionaria) == "" {
		rec.CodConcessionaria = rec.CodEmpresa
	}
	if p.ID > 0 {
		// UI now returns masked secret; keep current password unless a new raw value was provided.
		if rec.Senha == "" || strings.Contains(rec.Senha, "*") {
			existing, getErr := store.GetAuthRecordByID(r.Context(), p.ID)
			if getErr != nil {
				writeErr(w, http.StatusInternalServerError, getErr)
				return
			}
			rec.Senha = existing.Senha
		}
		err = store.UpdateAuthRecord(r.Context(), rec)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		record, getErr := store.GetAuthRecordByID(r.Context(), p.ID)
		if getErr != nil {
			writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "UsuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡rio atualizado"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "UsuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡rio atualizado", "user": authRecordToPayload(record)})
		return
	}

	newID, err := store.InsertAuthRecord(r.Context(), rec)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	record, getErr := store.GetAuthRecordByID(r.Context(), newID)
	if getErr != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "UsuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡rio criado", "id": newID})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "UsuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡rio criado", "user": authRecordToPayload(record)})
}

func (a *app) handleAuthUserDelete(w http.ResponseWriter, r *http.Request) {
	var p authUserPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id do usuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡rio invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	if err := store.DeleteAuthRecord(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "UsuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡rio removido"})
}

func (a *app) handleAuthUserLogin(w http.ResponseWriter, r *http.Request) {
	var p authUserPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id do usuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡rio invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}

	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	e := engine.New(cfg, store)
	if err := e.UpdateTokenByUserID(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	record, err := store.FindAuthRecord(r.Context(), strconv.FormatInt(p.ID, 10))
	if err != nil {
		writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "AutenticaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o concluÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â­da"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "AutenticaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o concluÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â­da",
		"user":    authRecordToPayload(record),
	})
}

func (a *app) handleAuthUsersLogin(w http.ResponseWriter, r *http.Request) {
	var p authIDsPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(p.IDs) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nenhum usuario selecionado"))
		return
	}

	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	e := engine.New(cfg, store)
	okCount := 0
	failCount := 0
	for _, id := range p.IDs {
		if id <= 0 {
			failCount++
			continue
		}
		if err := e.UpdateTokenByUserID(r.Context(), id); err != nil {
			failCount++
			continue
		}
		okCount++
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         okCount > 0,
		"message":    fmt.Sprintf("AutenticaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o concluÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â­da: sucesso=%d falha=%d", okCount, failCount),
		"ok_count":   okCount,
		"fail_count": failCount,
	})
}

func solicitacaoToPayload(r *db.SolicitacaoRecord) solicitacaoPayload {
	out := solicitacaoPayload{
		ID:                r.ID,
		Filial:            r.Filial,
		Vendedor:          r.Vendedor,
		CPF:               r.CPF,
		Modelo:            r.Modelo,
		Plano:             r.Plano,
		ComRestricao:      r.ComRestricao,
		Notes:             r.Notes,
		CotaRD:            r.CotaRD,
		Situacao:          r.Situacao,
		LanceContemplacao: r.LanceContemplacao,
	}
	if r.DataHoraSolicitacao.Valid {
		out.DataHoraSolicitacao = r.DataHoraSolicitacao.String
	}
	if r.QtdeParcelas.Valid {
		out.QtdeParcelas = strconv.FormatInt(r.QtdeParcelas.Int64, 10)
	}
	if r.PercLance.Valid {
		out.PercLance = strconv.FormatFloat(r.PercLance.Float64, 'f', -1, 64)
	}
	if r.Grupo.Valid {
		out.Grupo = strconv.FormatInt(r.Grupo.Int64, 10)
	}
	if r.IDCota.Valid {
		out.IDCota = strconv.FormatInt(r.IDCota.Int64, 10)
	}
	if r.GrupoAtendido.Valid {
		out.GrupoAtendido = strconv.FormatInt(r.GrupoAtendido.Int64, 10)
	}
	if r.DataHoraAtendimento.Valid {
		out.DataHoraAtendimento = r.DataHoraAtendimento.String
	}
	return out
}

func parseNullInt64(v string) (sql.NullInt64, error) {
	s := strings.TrimSpace(v)
	if s == "" || s == "-" {
		return sql.NullInt64{}, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: n, Valid: true}, nil
}

func parseNullFloat64(v string) (sql.NullFloat64, error) {
	s := strings.TrimSpace(v)
	if s == "" || s == "-" {
		return sql.NullFloat64{}, nil
	}
	n, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil {
		return sql.NullFloat64{}, err
	}
	return sql.NullFloat64{Float64: n, Valid: true}, nil
}

func (a *app) handleSolicitacoesSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	column := strings.TrimSpace(r.URL.Query().Get("column"))
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	_, fromDate, toDate, err := dashboardPeriodRange(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, err := store.SearchSolicitacoes(r.Context(), search, column, statusFilter, fromDate, toDate, 1000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	sess, ok := a.currentSession(r)
	if ok && isGerenteRole(sess.Role) {
		filtered := make([]db.SolicitacaoRecord, 0, len(records))
		branch := strings.ToLower(strings.TrimSpace(sess.branch))
		for i := range records {
			if branch == "" || strings.ToLower(strings.TrimSpace(records[i].Filial)) == branch {
				filtered = append(filtered, records[i])
			}
		}
		records = filtered
	}
	if ok && isSupervisorRole(sess.Role) {
		subordinates, err := store.ListSubordinatesBySupervisor(r.Context(), sess.Username, 5000)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		filtered := make([]db.SolicitacaoRecord, 0, len(records))
		for i := range records {
			if matchesSupervisorSubordinate(&records[i], subordinates) {
				filtered = append(filtered, records[i])
			}
		}
		records = filtered
	}
	groupCounts, err := store.CountSolicitacoesByGrupoInPeriodo(r.Context(), fromDate, toDate)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	holidays, _ := store.ListActiveNationalHolidayDates(r.Context())
	now := time.Now().In(utcMinus3Loc)
	parcelasByGrupo := make(map[int64]string)
	out := make([]solicitacaoPayload, 0, len(records))
	for i := range records {
		item := solicitacaoToPayload(&records[i])
		groupCode := int64(0)
		if records[i].Grupo.Valid {
			groupCode = records[i].Grupo.Int64
		} else if gtxt := strings.TrimSpace(item.Grupo); gtxt != "" {
			if gparsed, perr := strconv.ParseInt(gtxt, 10, 64); perr == nil {
				groupCode = gparsed
			}
		}
		if groupCode > 0 {
			if c, ok := groupCounts[groupCode]; ok {
				item.QtdSolicitada = strconv.FormatInt(c, 10)
			} else {
				item.QtdSolicitada = "0"
			}

			// Alinha a coluna Parc da tela de Solicitacoes com o mesmo calculo de Grupos Ativos.
			if cached, ok := parcelasByGrupo[groupCode]; ok {
				item.QtdeParcelas = cached
			} else if ga, gerr := store.GetGrupoAtivoByGrupo(r.Context(), groupCode); gerr == nil && ga != nil {
				parcelasCalc := calculateParcelasFromGrupoAtivo(ga, holidays, now)
				if parcelasCalc > 0 {
					item.QtdeParcelas = strconv.FormatInt(parcelasCalc, 10)
					parcelasByGrupo[groupCode] = item.QtdeParcelas
				} else {
					parcelasByGrupo[groupCode] = item.QtdeParcelas
				}
			}
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func onlyDigitsLocal(v string) string {
	var b strings.Builder
	for _, ch := range v {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func normalizeRoleLocal(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func isSupervisorRole(role string) bool {
	return normalizeRoleLocal(role) == "supervisor"
}

func isGerenteRole(role string) bool {
	return normalizeRoleLocal(role) == "gerente"
}

func matchesSupervisorSubordinate(rec *db.SolicitacaoRecord, subordinates []db.AppUserRecord) bool {
	if rec == nil || len(subordinates) == 0 {
		return false
	}
	recCPF := onlyDigitsLocal(strings.TrimSpace(rec.CPF))
	recName := strings.ToLower(strings.TrimSpace(rec.Vendedor))
	for i := range subordinates {
		sub := &subordinates[i]
		subCPF := onlyDigitsLocal(strings.TrimSpace(sub.CPF.String))
		subUsername := strings.ToLower(strings.TrimSpace(sub.Username))
		subDisplay := strings.ToLower(strings.TrimSpace(sub.DisplayName))
		if recCPF != "" && subCPF != "" && recCPF == subCPF {
			return true
		}
		if recName != "" && (recName == subUsername || recName == subDisplay) {
			return true
		}
	}
	return false
}

func mergeUniqueSubordinates(items ...[]db.AppUserRecord) []db.AppUserRecord {
	out := make([]db.AppUserRecord, 0)
	seen := make(map[int64]struct{})
	for _, list := range items {
		for i := range list {
			id := list[i].ID
			if id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, list[i])
		}
	}
	return out
}

func buildSupervisorRequestsScope(ctx context.Context, store *db.Store, sess *appSession) (string, []any, error) {
	if store == nil || sess == nil || !isSupervisorRole(sess.Role) {
		return "", nil, nil
	}
	byUser, err := store.ListSubordinatesBySupervisor(ctx, sess.Username, 5000)
	if err != nil {
		return "", nil, err
	}
	byName, err := store.ListSubordinatesBySupervisor(ctx, sess.DisplayName, 5000)
	if err != nil {
		return "", nil, err
	}
	subordinates := mergeUniqueSubordinates(byUser, byName)
	if len(subordinates) == 0 {
		branch := strings.TrimSpace(sess.branch)
		if branch != "" {
			return "COALESCE(TRIM(branch),'') = ?", []any{branch}, nil
		}
		return "1=0", nil, nil
	}

	sellerKeys := make([]string, 0, len(subordinates)*2)
	cpfKeys := make([]string, 0, len(subordinates))
	sellerSeen := make(map[string]struct{})
	cpfSeen := make(map[string]struct{})
	for i := range subordinates {
		sub := &subordinates[i]
		u := strings.ToLower(strings.TrimSpace(sub.Username))
		if u != "" {
			if _, ok := sellerSeen[u]; !ok {
				sellerSeen[u] = struct{}{}
				sellerKeys = append(sellerKeys, u)
			}
		}
		d := strings.ToLower(strings.TrimSpace(sub.DisplayName))
		if d != "" {
			if _, ok := sellerSeen[d]; !ok {
				sellerSeen[d] = struct{}{}
				sellerKeys = append(sellerKeys, d)
			}
		}
		cpf := onlyDigitsLocal(strings.TrimSpace(sub.CPF.String))
		if cpf != "" {
			if _, ok := cpfSeen[cpf]; !ok {
				cpfSeen[cpf] = struct{}{}
				cpfKeys = append(cpfKeys, cpf)
			}
		}
	}
	if len(sellerKeys) == 0 && len(cpfKeys) == 0 {
		branch := strings.TrimSpace(sess.branch)
		if branch != "" {
			return "COALESCE(TRIM(branch),'') = ?", []any{branch}, nil
		}
		return "1=0", nil, nil
	}

	clauses := make([]string, 0, 2)
	args := make([]any, 0, len(sellerKeys)+len(cpfKeys)+1)
	if len(sellerKeys) > 0 {
		ph := strings.TrimRight(strings.Repeat("?,", len(sellerKeys)), ",")
		clauses = append(clauses, "LOWER(TRIM(COALESCE(seller_name,''))) IN ("+ph+")")
		for _, k := range sellerKeys {
			args = append(args, k)
		}
	}
	if len(cpfKeys) > 0 {
		ph := strings.TrimRight(strings.Repeat("?,", len(cpfKeys)), ",")
		clauses = append(clauses, "REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(COALESCE(cpf,''),'.',''),'-',''),'/',''),'(',''),')','') IN ("+ph+")")
		for _, k := range cpfKeys {
			args = append(args, k)
		}
	}

	scope := "(" + strings.Join(clauses, " OR ") + ")"
	branch := strings.TrimSpace(sess.branch)
	if branch != "" {
		scope = "(" + scope + " AND COALESCE(TRIM(branch),'') = ?)"
		args = append(args, branch)
	}
	return scope, args, nil
}

func (a *app) calculateMinhasSituacao(rec *db.SolicitacaoRecord) string {
	hasAtendimento := rec.DataHoraAtendimento.Valid && strings.TrimSpace(rec.DataHoraAtendimento.String) != ""
	hasCotaRD := strings.TrimSpace(rec.CotaRD) != ""

	// Check for SLA (if served_at > requested_at)
	hasSLA := false
	if hasAtendimento && rec.DataHoraSolicitacao.Valid {
		reqStr := strings.TrimSpace(rec.DataHoraSolicitacao.String)
		serStr := strings.TrimSpace(rec.DataHoraAtendimento.String)
		if reqStr != "" && serStr != "" {
			req, err1 := time.Parse("2006-01-02 15:04:05", reqStr)
			ser, err2 := time.Parse("2006-01-02 15:04:05", serStr)
			if err1 == nil && err2 == nil {
				if ser.Sub(req) > 0 {
					hasSLA = true
				}
			}
		}
	}

	// Solicitada: Quando nÃƒÂ£o tiver data de atendimento, SLA ou Grupo Cota R/D
	if !hasAtendimento && !hasSLA && !hasCotaRD {
		return "solicitada"
	}

	// Expiradas: 10 Dias apÃƒÂ³s a data de atendimento (e futuramente sem correspondÃƒÂªncia salles_cnh)
	if hasAtendimento {
		serStr := strings.TrimSpace(rec.DataHoraAtendimento.String)
		ser, err := time.Parse("2006-01-02 15:04:05", serStr)
		if err == nil {
			if time.Since(ser) > 10*24*time.Hour {
				// No futuro, cruzar com salles_cnh aqui. Por enquanto, se passou 10 dias, ÃƒÂ© expirada.
				return "expirada"
			}
		}
	}

	// Atendida: Inverso da anterior (e que nÃƒÂ£o expirou nem foi digitada ainda)
	return "atendida"
}

func (a *app) handleMinhasSolicitacoesSearch(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, runResponse{OK: false, Message: "NÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o autenticado"})
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("q"))
	statusFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if statusFilter == "" {
		statusFilter = "solicitada"
	}

	_, fromDate, toDate, err := dashboardPeriodRange(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, err := store.SearchSolicitacoes(r.Context(), search, "", "all", fromDate, toDate, 2000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	sessionCPF := onlyDigitsLocal(strings.TrimSpace(sess.CPF))
	sessionName := strings.ToLower(strings.TrimSpace(sess.DisplayName))
	sessionFilial := strings.ToLower(strings.TrimSpace(sess.branch))
	isAdmin := strings.EqualFold(strings.TrimSpace(sess.Role), "admin")
	isSupervisor := isSupervisorRole(sess.Role)
	isGerente := isGerenteRole(sess.Role)
	subordinates := []db.AppUserRecord{}
	if isSupervisor {
		subordinates, err = store.ListSubordinatesBySupervisor(r.Context(), sess.Username, 5000)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}

	out := make([]solicitacaoPayload, 0, len(records))
	for i := range records {
		rec := &records[i]

		if isSupervisor {
			if !matchesSupervisorSubordinate(rec, subordinates) {
				continue
			}
		}
		if isGerente {
			if sessionFilial != "" && strings.ToLower(strings.TrimSpace(rec.Filial)) != sessionFilial {
				continue
			}
		}

		if !isAdmin && !isSupervisor && !isGerente {
			recCPF := onlyDigitsLocal(strings.TrimSpace(rec.CPF))
			recName := strings.ToLower(strings.TrimSpace(rec.Vendedor))
			recFilial := strings.ToLower(strings.TrimSpace(rec.Filial))

			matchByCPF := sessionCPF != "" && recCPF != "" && recCPF == sessionCPF
			matchByName := sessionCPF == "" && sessionName != "" && recName == sessionName
			if !matchByCPF && !matchByName {
				continue
			}
			if sessionFilial != "" && recFilial != "" && recFilial != sessionFilial {
				continue
			}
		}

		sitNorm := a.calculateMinhasSituacao(rec)
		if statusFilter != "all" && sitNorm != statusFilter {
			continue
		}

		item := solicitacaoToPayload(rec)
		switch sitNorm {
		case "solicitada":
			item.Situacao = "Solicitada"
		case "digitada":
			item.Situacao = "Digitada"
		case "expirada":
			item.Situacao = "Expirada"
		case "atendida":
			item.Situacao = "Atendida"
		default:
			item.Situacao = strings.TrimSpace(rec.Situacao)
		}
		out = append(out, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func (a *app) handleSolicitacaoGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rec, err := store.GetSolicitacaoByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("solicitaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o nÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o encontrada"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	sess, ok := a.currentSession(r)
	if ok && isGerenteRole(sess.Role) {
		if strings.TrimSpace(sess.branch) != "" && !strings.EqualFold(strings.TrimSpace(sess.branch), strings.TrimSpace(rec.Filial)) {
			writeErr(w, http.StatusForbidden, fmt.Errorf("acesso negado"))
			return
		}
	}
	if ok && isSupervisorRole(sess.Role) {
		subordinates, err := store.ListSubordinatesBySupervisor(r.Context(), sess.Username, 5000)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		if !matchesSupervisorSubordinate(rec, subordinates) {
			writeErr(w, http.StatusForbidden, fmt.Errorf("acesso negado"))
			return
		}
	}
	writeJSON(w, http.StatusOK, solicitacaoToPayload(rec))
}

func (a *app) handleSolicitacaoLastByCPF(w http.ResponseWriter, r *http.Request) {
	cpf := onlyDigitsLocal(strings.TrimSpace(r.URL.Query().Get("cpf")))
	if cpf == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("cpf invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	// 1. Tenta buscar na tabela de usuÃƒÂ¡rios do sistema (vendedores cadastrados)
	u, err := store.GetAppUserByCPF(r.Context(), cpf)
	if err == nil {
		fmt.Printf("[DEBUG] UsuÃƒÂ¡rio encontrado na tabela users para CPF %s: %s\n", cpf, u.DisplayName)
		payload := solicitacaoPayload{
			CPF:      u.CPF.String,
			Filial:   u.Filial.String,
			Vendedor: u.DisplayName,
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "found": true, "item": payload})
		return
	}

	// 2. Tenta buscar na tabela de contas de API (usuÃƒÂ¡rios de autenticaÃƒÂ§ÃƒÂ£o)
	auth, err := store.GetAuthRecordByCPF(r.Context(), cpf)
	if err == nil {
		fmt.Printf("[DEBUG] Conta API encontrada para CPF %s: %s\n", cpf, auth.CodUsuario)
		payload := solicitacaoPayload{
			CPF:      auth.CPF,
			Filial:   auth.CodConcessionaria,
			Vendedor: auth.CodUsuario,
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "found": true, "item": payload})
		return
	}

	// 3. Fallback: busca na ÃƒÂºltima solicitaÃƒÂ§ÃƒÂ£o realizada para este CPF (histÃƒÂ³rico de clientes)
	rec, err := store.GetLastSolicitacaoByCPF(r.Context(), cpf)
	if err == nil {
		fmt.Printf("[DEBUG] HistÃƒÂ³rico encontrado na tabela requests para CPF %s: ID %d\n", cpf, rec.ID)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "found": true, "item": solicitacaoToPayload(rec)})
		return
	}

	// Se nÃƒÂ£o encontrar em nenhum lugar
	fmt.Printf("[DEBUG] Nenhuma informaÃƒÂ§ÃƒÂ£o encontrada em nenhuma tabela para o CPF: %s\n", cpf)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "found": false})
}

func (a *app) handleSolicitacaoSave(w http.ResponseWriter, r *http.Request) {
	var p solicitacaoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	qtdeParcelas, err := parseNullInt64(p.QtdeParcelas)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("installments invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	percLance, err := parseNullFloat64(p.PercLance)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("bid_percent invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	group_code, err := parseNullInt64(p.Grupo)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("group_code invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	grupoAtendido, err := parseNullInt64(p.GrupoAtendido)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("served_group invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	idCota, err := parseNullInt64(p.IDCota)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("requested_quota_id invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}

	rec := db.SolicitacaoRecord{
		ID: p.ID,
		DataHoraSolicitacao: sql.NullString{String: func() string {
			v := strings.TrimSpace(p.DataHoraSolicitacao)
			if v == "-" {
				return ""
			}
			return v
		}(), Valid: func() bool {
			v := strings.TrimSpace(p.DataHoraSolicitacao)
			return v != "" && v != "-"
		}()},
		Filial:        strings.TrimSpace(p.Filial),
		Vendedor:      strings.TrimSpace(p.Vendedor),
		CPF:           onlyDigitsLocal(p.CPF),
		Modelo:        strings.TrimSpace(p.Modelo),
		Plano:         strings.TrimSpace(p.Plano),
		QtdeParcelas:  qtdeParcelas,
		PercLance:     percLance,
		ComRestricao:  strings.TrimSpace(p.ComRestricao),
		Grupo:         group_code,
		Notes:         strings.TrimSpace(p.Notes),
		IDCota:        idCota,
		GrupoAtendido: grupoAtendido,
		CotaRD:        strings.TrimSpace(p.CotaRD),
		DataHoraAtendimento: sql.NullString{String: func() string {
			v := strings.TrimSpace(p.DataHoraAtendimento)
			if v == "-" {
				return ""
			}
			return v
		}(), Valid: func() bool {
			v := strings.TrimSpace(p.DataHoraAtendimento)
			return v != "" && v != "-"
		}()},
		Situacao:          strings.TrimSpace(p.Situacao),
		LanceContemplacao: strings.TrimSpace(p.LanceContemplacao),
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	newID, err := store.SaveSolicitacao(r.Context(), rec)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	item, err := store.GetSolicitacaoByID(r.Context(), newID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "SolicitaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o salva", "id": newID})
		return
	}
	msg := "SolicitaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o atualizada"
	if p.ID <= 0 {
		msg = "SolicitaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o criada"
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": msg, "item": solicitacaoToPayload(item), "id": newID})
}

func (a *app) handleSolicitacaoDelete(w http.ResponseWriter, r *http.Request) {
	var p solicitacaoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := store.DeleteSolicitacao(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "SolicitaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o removida"})
}

func parseCotaRD(v string) (string, string, string) {
	parts := strings.Split(strings.TrimSpace(v), "-")
	if len(parts) == 0 {
		return "", "", ""
	}
	get := func(i int) string {
		if i < 0 || i >= len(parts) {
			return ""
		}
		return strings.TrimSpace(parts[i])
	}
	return get(0), get(1), get(2)
}

func isSolicitacaoAtendida(rec *db.SolicitacaoRecord) bool {
	if rec == nil {
		return false
	}
	if rec.DataHoraAtendimento.Valid && strings.TrimSpace(rec.DataHoraAtendimento.String) != "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(rec.Situacao)), "atendid")
}

type reserveSolicitacaoError struct {
	Code    string
	message string
	CPF     string
}

func (e *reserveSolicitacaoError) Error() string {
	return e.message
}

func normalizeMatchText(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	repl := strings.NewReplacer(
		"ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡", "a", "ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â ", "a", "ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¢", "a", "ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£", "a",
		"ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â©", "e", "ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Âª", "e",
		"ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â­", "i",
		"ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â³", "o", "ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â´", "o", "ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Âµ", "o",
		"ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Âº", "u",
		"ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§", "c",
	)
	return repl.Replace(s)
}

func shouldMarkFailedByAPIBody(raw string) (bool, string) {
	body := normalizeMatchText(raw)
	if strings.Contains(body, "ocorreu um erro inesperado") {
		return true, "erro_inesperado"
	}
	if strings.Contains(body, "limite de chamadas excedido") {
		return false, "limite_chamadas_excedido"
	}
	if strings.Contains(body, "cpf atingiu o limite de reservas") {
		return false, "cpf_limite_reservas"
	}
	if strings.Contains(body, "model_name nao encontrado") {
		return false, "modelo_nao_encontrado"
	}
	return false, "sem_regra"
}

func (a *app) reserveSolicitacaoByID(ctx context.Context, cfg *config.Config, store *db.Store, id int64, logPath string) (string, *solicitacaoSuccessItemPayload, error) {
	a.uiLogfTo(logPath, "[go][solicitacao] iniciando id=%d", id)
	rec, err := store.GetSolicitacaoByID(ctx, id)
	if err != nil {
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro carregar registro: %v", id, err)
		return "", nil, err
	}
	if isSolicitacaoAtendida(rec) {
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d ignorada: ja atendida", id)
		return "Solicitacao ja atendida", nil, nil
	}

	group_code := ""
	if rec.GrupoAtendido.Valid {
		group_code = strconv.FormatInt(rec.GrupoAtendido.Int64, 10)
	}
	if group_code == "" && rec.Grupo.Valid {
		group_code = strconv.FormatInt(rec.Grupo.Int64, 10)
	}
	idCota := ""
	if rec.IDCota.Valid {
		idCota = strconv.FormatInt(rec.IDCota.Int64, 10)
	}
	quota, rVal, dVal := parseCotaRD(rec.CotaRD)

	cotaRow, err := store.FindAvailableCotaForSolicitacao(ctx, group_code, idCota, quota, rVal, dVal)
	if err != nil {
		if err == sql.ErrNoRows {
			a.uiLogfTo(logPath, "[go][solicitacao] id=%d sem quota disponivel (group_code=%s requested_quota_id=%s quota_rd=%s)", id, group_code, idCota, rec.CotaRD)
			return "", nil, fmt.Errorf("nenhuma quota disponivel encontrada para a solicitacao")
		}
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro consulta quota: %v", id, err)
		return "", nil, err
	}

	user, err := store.FindBookingUserByCPF(ctx, rec.CPF)
	if err != nil {
		if err == sql.ErrNoRows {
			a.uiLogfTo(logPath, "[go][solicitacao] id=%d cpf=%s sem token na auth", id, strings.TrimSpace(rec.CPF))
			return "", nil, &reserveSolicitacaoError{
				Code:    "no_token",
				message: "CPF sem token is_active na tabela auth",
				CPF:     strings.TrimSpace(rec.CPF),
			}
		}
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro consulta auth: %v", id, err)
		return "", nil, err
	}

	modeloSolicitado := strings.TrimSpace(rec.Modelo)
	model_api_id, err := store.FindIDModeloByNome(ctx, modeloSolicitado)
	if err != nil {
		if err == sql.ErrNoRows {
			a.uiLogfTo(logPath, "[go][solicitacao] id=%d model_name sem mapeamento na tabela Modelos: %s", id, modeloSolicitado)
			return "", nil, fmt.Errorf("model_name sem mapeamento na tabela Modelos: %s", modeloSolicitado)
		}
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro consulta model_name=%s: %v", id, modeloSolicitado, err)
		return "", nil, err
	}
	model_name := strconv.FormatInt(model_api_id, 10)
	payloadPreview := map[string]string{
		"CodEmpresa":        user.CodEmpresa,
		"CodUsuario":        user.CodUsuario,
		"CPFCNPJ":           user.CPF,
		"id_cota_reposicao": strconv.FormatInt(cotaRow.IDGrupo, 10),
		"model_api_id":      model_name,
		"product_api_id":    strings.TrimSpace(cotaRow.Produto),
	}
	payloadJSON, _ := json.Marshal(payloadPreview)
	a.uiLogfTo(logPath, "[go][solicitacao] id=%d payload=%s", id, string(payloadJSON))
	if cfg.Booking.DryRun {
		a.uiLogfTo(logPath, "[go][dry-run][solicitacao] id=%d payload=%s", id, string(payloadJSON))
		return "DRY_RUN payload=" + string(payloadJSON), nil, nil
	}

	e := engine.New(cfg, store)
	result, err := e.ReserveByUserNoPersist(ctx, user, engine.ReserveInput{
		IDCotaReposicao: strconv.FormatInt(cotaRow.IDGrupo, 10),
		IDModelo:        model_name,
		IDProduto:       strings.TrimSpace(cotaRow.Produto),
	})
	if err != nil {
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro reserva API: %v", id, err)
		return "", nil, err
	}
	if result.StatusCode != 200 {
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d status=%d body=%s", id, result.StatusCode, result.Body)
		shouldFail, reason := shouldMarkFailedByAPIBody(result.Body)
		if shouldFail {
			if ferr := store.MarkCotaFailed(ctx, cotaRow.ID); ferr != nil {
				a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro marcar failed (reason=%s): %v", id, reason, ferr)
			} else {
				a.uiLogfTo(logPath, "[go][solicitacao] id=%d failed=1 atualizado (reason=%s)", id, reason)
			}
		} else {
			a.uiLogfTo(logPath, "[go][solicitacao] id=%d failed nao atualizado (reason=%s)", id, reason)
		}
		return "", nil, fmt.Errorf("reserva nao concluida: status=%d", result.StatusCode)
	}
	if err := store.MarkCotaBooked(ctx, cotaRow.ID); err != nil {
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro marcar booked: %v", id, err)
		return "", nil, err
	}

	id_cota_reposicao := strconv.FormatInt(cotaRow.IDGrupo, 10)
	grupoAtendido := ""
	if cotaRow.Grupo > 0 {
		grupoAtendido = strconv.Itoa(cotaRow.Grupo)
	}
	if grupoAtendido == "" && rec.Grupo.Valid {
		grupoAtendido = strconv.FormatInt(rec.Grupo.Int64, 10)
	}
	cotaRDSaved := strings.TrimSpace(rec.CotaRD)
	if cotaRow.Cota > 0 || cotaRow.R > 0 || cotaRow.D > 0 {
		cotaRDSaved = fmt.Sprintf("%d-%d-%d", cotaRow.Cota, cotaRow.R, cotaRow.D)
	}
	if cotaRDSaved == "" && cotaRow.Cota > 0 {
		cotaRDSaved = strconv.Itoa(cotaRow.Cota)
	}
	modeloNome := strings.TrimSpace(rec.Modelo)
	if nomeByID, nerr := store.FindModeloNomeByIDModelo(ctx, model_api_id); nerr == nil && strings.TrimSpace(nomeByID) != "" {
		modeloNome = strings.TrimSpace(nomeByID)
	}
	if modeloNome == "" {
		modeloNome = strings.TrimSpace(model_name)
	}
	if _, err := store.SaveCotasReservadas(ctx, []db.ReservedCota{{
		RequestID:          id,
		UsuarioReserva:     strings.TrimSpace(user.CodUsuario),
		NumDocumentoPessoa: strings.TrimSpace(user.CPF),
		CodGrupo:           strings.TrimSpace(grupoAtendido),
		CotaRD:             strings.TrimSpace(cotaRDSaved),
		CodModelo:          strings.TrimSpace(modeloNome),
		IDCotaReposicao:    strings.TrimSpace(id_cota_reposicao),
	}}); err != nil {
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro salvar cotasreservadas deterministico: %v", id, err)
		return "", nil, err
	}

	now := nowDateTimeUTCMinus3()
	rec.DataHoraAtendimento = sql.NullString{String: now, Valid: true}
	rec.Situacao = "Atendido"
	if idVal, convErr := strconv.ParseInt(id_cota_reposicao, 10, 64); convErr == nil {
		rec.IDCota = sql.NullInt64{Int64: idVal, Valid: true}
	}
	if codGrupoSaved := strings.TrimSpace(grupoAtendido); codGrupoSaved != "" {
		if grupoVal, convErr := strconv.ParseInt(codGrupoSaved, 10, 64); convErr == nil {
			rec.GrupoAtendido = sql.NullInt64{Int64: grupoVal, Valid: true}
		}
	}
	if strings.TrimSpace(cotaRDSaved) != "" {
		rec.CotaRD = strings.TrimSpace(cotaRDSaved)
	}
	if err := store.MarkSolicitacaoAtendidaByID(
		ctx,
		rec.ID,
		rec.IDCota,
		rec.GrupoAtendido,
		rec.CotaRD,
		now,
		"Atendido",
	); err != nil {
		a.uiLogfTo(logPath, "[go][solicitacao] id=%d erro atualizar solicitacao: %v", id, err)
		return "", nil, err
	}

	a.uiLogfTo(logPath, "[go][solicitacao] id=%d concluida status=200", id)
	successItem := &solicitacaoSuccessItemPayload{
		ID:            rec.ID,
		Nome:          strings.TrimSpace(rec.Vendedor),
		Filial:        strings.TrimSpace(rec.Filial),
		CPF:           strings.TrimSpace(rec.CPF),
		GrupoAtendido: strings.TrimSpace(grupoAtendido),
		CotaRD:        strings.TrimSpace(rec.CotaRD),
		QtdeParcelas: func() string {
			if rec.QtdeParcelas.Valid {
				return strconv.FormatInt(rec.QtdeParcelas.Int64, 10)
			}
			return ""
		}(),
		Modelo:       strings.TrimSpace(modeloNome),
		Licenciada:   strings.TrimSpace(rec.Plano),
		ComRestricao: strings.TrimSpace(rec.ComRestricao),
		SolicitadaEm: formatDateTimeBRNoSeconds(strings.TrimSpace(rec.DataHoraSolicitacao.String)),
		AtendidaEm:   formatDateTimeBRNoSeconds(strings.TrimSpace(now)),
		SLA:          formatSLADuration(strings.TrimSpace(rec.DataHoraSolicitacao.String), strings.TrimSpace(now)),
	}
	if successItem.Nome == "" {
		successItem.Nome = strings.TrimSpace(rec.CPF)
	}
	return "Solicitacao atendida com sucesso", successItem, nil
}
func (a *app) handleSolicitacaoReservar(w http.ResponseWriter, r *http.Request) {
	var p solicitacaoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	logPath := a.newUILogFilePath("solicitacao")
	a.uiLogfTo(logPath, "[go] Log file: %s", logPath)
	msg, successItem, err := a.reserveSolicitacaoByID(r.Context(), cfg, store, p.ID, logPath)
	if err != nil {
		var rsErr *reserveSolicitacaoError
		if errors.As(err, &rsErr) && rsErr.Code == "no_token" {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":      false,
				"code":    rsErr.Code,
				"cpf":     rsErr.CPF,
				"message": rsErr.message,
				"id":      p.ID,
			})
			return
		}
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	successItems := make([]solicitacaoSuccessItemPayload, 0, 1)
	if successItem != nil {
		successItems = append(successItems, *successItem)
	}
	if len(successItems) > 0 {
		if _, err := a.enqueueManualNotifications(r.Context(), store, successItems); err != nil {
			a.uiLogf("[go][notificacao] erro enfileirar (solicitacao id=%d): %v", p.ID, err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": msg, "id": p.ID, "success_items": successItems})
}

func (a *app) handleSolicitacoesReservarBatch(w http.ResponseWriter, r *http.Request) {
	var p solicitacaoIDsPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(p.IDs) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nenhuma solicitaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o selecionada"))
		return
	}
	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	logPath := a.newUILogFilePath("solicitacao_batch")
	a.uiLogfTo(logPath, "[go] Log file: %s", logPath)

	okCount := 0
	skipCount := 0
	failCount := 0
	noTokenCount := 0
	successItems := make([]solicitacaoSuccessItemPayload, 0, len(p.IDs))
	for _, id := range p.IDs {
		if id <= 0 {
			failCount++
			continue
		}
		msg, successItem, err := a.reserveSolicitacaoByID(r.Context(), cfg, store, id, logPath)
		if err != nil {
			var rsErr *reserveSolicitacaoError
			if errors.As(err, &rsErr) && rsErr.Code == "no_token" {
				skipCount++
				noTokenCount++
				continue
			}
			failCount++
			continue
		}
		if strings.Contains(strings.ToLower(msg), "jÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡ atendida") {
			skipCount++
			continue
		}
		okCount++
		if successItem != nil {
			successItems = append(successItems, *successItem)
		}
	}
	if len(successItems) > 0 {
		if _, err := a.enqueueManualNotifications(r.Context(), store, successItems); err != nil {
			a.uiLogf("[go][notificacao] erro enfileirar em lote: %v", err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            okCount > 0,
		"ok_count":      okCount,
		"skip_count":    skipCount,
		"fail_count":    failCount,
		"no_token":      noTokenCount,
		"success_items": successItems,
		"message":       fmt.Sprintf("Reserva concluÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â­da: sucesso=%d puladas=%d (sem token=%d) falha=%d", okCount, skipCount, noTokenCount, failCount),
	})
}

func (a *app) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, runResponse{OK: false, Message: "Nao autenticado"})
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	period, fromDate, toDate, err := dashboardPeriodRange(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	filialSel := strings.TrimSpace(r.URL.Query().Get("branch"))
	supervisorScope := ""
	supervisorArgs := make([]any, 0)
	if sess != nil && isSupervisorRole(sess.Role) {
		if strings.TrimSpace(sess.branch) != "" {
			filialSel = strings.TrimSpace(sess.branch)
		}
		supervisorScope, supervisorArgs, err = buildSupervisorRequestsScope(r.Context(), store, sess)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	if sess != nil && isGerenteRole(sess.Role) && strings.TrimSpace(sess.branch) != "" {
		filialSel = strings.TrimSpace(sess.branch)
	}
	if sess != nil && !strings.EqualFold(strings.TrimSpace(sess.Role), "admin") && !isSupervisorRole(sess.Role) && !isGerenteRole(sess.Role) && strings.TrimSpace(sess.branch) != "" {
		filialSel = strings.TrimSpace(sess.branch)
	}

	atendidaCond := `( (status IS NOT NULL AND LOWER(TRIM(status)) LIKE 'atendid%') OR
		(served_at IS NOT NULL AND TRIM(CAST(served_at AS TEXT)) <> '') OR
		(served_date IS NOT NULL AND TRIM(CAST(served_date AS TEXT)) <> '') )`
	erroCond := `( status IS NOT NULL AND (LOWER(TRIM(status)) LIKE 'falh%' OR LOWER(TRIM(status)) LIKE 'erro%') )`
	baseDateExpr := `COALESCE(NULLIF(SUBSTR(TRIM(CAST(requested_date AS TEXT)), 1, 10), ''), SUBSTR(TRIM(CAST(requested_at AS TEXT)), 1, 10))`
	baseHourExpr := `COALESCE(NULLIF(TRIM(CAST(requested_time AS TEXT)), ''), SUBSTR(TRIM(CAST(requested_at AS TEXT)), 12, 2) || ':00:00')`
	dateConds := []string{
		baseDateExpr + ` IS NOT NULL`,
		`TRIM(` + baseDateExpr + `) <> ''`,
		baseDateExpr + ` >= ?`,
		baseDateExpr + ` <= ?`,
	}
	whereArgs := []any{fromDate, toDate}
	if filialSel != "" {
		dateConds = append(dateConds, `COALESCE(TRIM(branch),'') = ?`)
		whereArgs = append(whereArgs, filialSel)
	}
	if supervisorScope != "" {
		dateConds = append(dateConds, supervisorScope)
		whereArgs = append(whereArgs, supervisorArgs...)
	}
	whereClause := strings.Join(dateConds, " AND ")

	total, err := queryInt64Ctx(r.Context(), store, `SELECT COUNT(1) FROM requests WHERE `+whereClause, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	atendidas, err := queryInt64Ctx(r.Context(), store, `SELECT COUNT(1) FROM requests WHERE `+whereClause+` AND `+atendidaCond, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	naoAtendidas := total - atendidas
	if naoAtendidas < 0 {
		naoAtendidas = 0
	}

	var slaAvgMin sql.NullFloat64
	if err := store.DB.QueryRowContext(r.Context(), store.Rebind(`
SELECT AVG(`+slaMinutesExpr(store)+`)
FROM requests
WHERE `+whereClause+`
  AND `+atendidaCond+`
  AND requested_at IS NOT NULL AND TRIM(CAST(requested_at AS TEXT)) <> ''
  AND served_at IS NOT NULL AND TRIM(CAST(served_at AS TEXT)) <> ''
  AND `+slaNonNegativeCond(store)+``), whereArgs...).Scan(&slaAvgMin); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	slaSamples, err := querySLAMinutes(r.Context(), store, whereClause, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	slaMediana, slaP95 := percentilesFromSamples(slaSamples)

	var lanceAvg sql.NullFloat64
	if err := store.DB.QueryRowContext(r.Context(), store.Rebind(`
SELECT AVG(bid_percent) FROM requests
WHERE `+whereClause+`
  AND bid_percent IS NOT NULL`), whereArgs...).Scan(&lanceAvg); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	errosReserva, err := queryInt64Ctx(r.Context(), store, `SELECT COUNT(1) FROM requests WHERE `+whereClause+` AND `+erroCond, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	var taxaErro float64
	if total > 0 {
		taxaErro = (float64(errosReserva) * 100.0) / float64(total)
	}

	backlogConds := []string{`NOT (` + atendidaCond + `)`}
	backlogArgs := make([]any, 0, 1)
	if filialSel != "" {
		backlogConds = append(backlogConds, `COALESCE(TRIM(branch),'') = ?`)
		backlogArgs = append(backlogArgs, filialSel)
	}
	if supervisorScope != "" {
		backlogConds = append(backlogConds, supervisorScope)
		backlogArgs = append(backlogArgs, supervisorArgs...)
	}
	backlogAtual, err := queryInt64Ctx(r.Context(), store, `SELECT COUNT(1) FROM requests WHERE `+strings.Join(backlogConds, " AND "), backlogArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	filiais, err := queryDashboardRank(r.Context(), store, `
SELECT COALESCE(NULLIF(TRIM(branch), ''), '(Sem filial)') AS nome,
       COUNT(1) AS total,
       SUM(CASE WHEN `+atendidaCond+` THEN 1 ELSE 0 END) AS atendidas
FROM requests
WHERE `+whereClause+`
GROUP BY nome
ORDER BY total DESC
LIMIT 10`, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	vendedoresTop, err := queryDashboardRank(r.Context(), store, `
SELECT COALESCE(NULLIF(TRIM(seller_name), ''), '(Sem seller_name)') AS nome,
       COUNT(1) AS total,
       SUM(CASE WHEN `+atendidaCond+` THEN 1 ELSE 0 END) AS atendidas
FROM requests
WHERE `+whereClause+`
GROUP BY nome
ORDER BY total DESC
LIMIT 10`, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	vendedoresTaxa, err := queryDashboardRank(r.Context(), store, `
SELECT COALESCE(NULLIF(TRIM(seller_name), ''), '(Sem seller_name)') AS nome,
       COUNT(1) AS total,
       SUM(CASE WHEN `+atendidaCond+` THEN 1 ELSE 0 END) AS atendidas
FROM requests
WHERE `+whereClause+`
GROUP BY nome
HAVING COUNT(1) >= 1
ORDER BY (SUM(CASE WHEN `+atendidaCond+` THEN 1 ELSE 0 END) * 1.0 / COUNT(1)) DESC, COUNT(1) DESC
LIMIT 10`, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	gruposAtendidos, err := queryDashboardRank(r.Context(), store, `
SELECT COALESCE(NULLIF(CAST(served_group AS TEXT), ''), '(Sem group_code)') AS nome,
       COUNT(1) AS total,
       COUNT(1) AS atendidas
FROM requests
WHERE `+whereClause+` AND served_group IS NOT NULL
GROUP BY nome
ORDER BY total DESC
LIMIT 10`, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	filiaisBacklog, err := queryDashboardRank(r.Context(), store, `
SELECT COALESCE(NULLIF(TRIM(branch), ''), '(Sem filial)') AS nome,
       COUNT(1) AS total,
       0 AS atendidas
FROM requests
WHERE `+whereClause+`
  AND NOT (`+atendidaCond+`)
GROUP BY nome
ORDER BY total DESC
LIMIT 10`, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	serie, err := queryDashboardSerie(r.Context(), store, atendidaCond, whereClause, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	serieHora, err := queryDashboardSerieHora(r.Context(), store, atendidaCond, whereClause, baseHourExpr, whereArgs...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	ultimaAtendidaDelta, ultimaAtendidaEm, err := queryTempoDesdeUltimaAtendida(r.Context(), store.DB, filialSel, supervisorScope, supervisorArgs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	filiaisFiltro, err := queryDistinctFiliais(r.Context(), store.DB)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if sess != nil && isSupervisorRole(sess.Role) {
		b := strings.TrimSpace(sess.branch)
		if b != "" {
			filiaisFiltro = []string{b}
		}
	}
	if sess != nil && isGerenteRole(sess.Role) {
		b := strings.TrimSpace(sess.branch)
		if b != "" {
			filiaisFiltro = []string{b}
		}
	}

	var taxa float64
	if total > 0 {
		taxa = (float64(atendidas) * 100.0) / float64(total)
	}
	prevFrom, prevTo, err := previousPeriodRange(fromDate, toDate)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	prevConds := []string{
		baseDateExpr + ` IS NOT NULL`,
		`TRIM(` + baseDateExpr + `) <> ''`,
		baseDateExpr + ` >= ?`,
		baseDateExpr + ` <= ?`,
	}
	prevArgs := []any{prevFrom, prevTo}
	if filialSel != "" {
		prevConds = append(prevConds, `COALESCE(TRIM(branch),'') = ?`)
		prevArgs = append(prevArgs, filialSel)
	}
	if supervisorScope != "" {
		prevConds = append(prevConds, supervisorScope)
		prevArgs = append(prevArgs, supervisorArgs...)
	}
	prevWhere := strings.Join(prevConds, " AND ")
	prevTotal, _ := queryInt64Ctx(r.Context(), store, `SELECT COUNT(1) FROM requests WHERE `+prevWhere, prevArgs...)
	prevAtendidas, _ := queryInt64Ctx(r.Context(), store, `SELECT COUNT(1) FROM requests WHERE `+prevWhere+` AND `+atendidaCond, prevArgs...)
	prevNaoAtendidas := prevTotal - prevAtendidas
	if prevNaoAtendidas < 0 {
		prevNaoAtendidas = 0
	}
	var prevTaxa float64
	if prevTotal > 0 {
		prevTaxa = (float64(prevAtendidas) * 100.0) / float64(prevTotal)
	}
	var prevSla sql.NullFloat64
	_ = store.DB.QueryRowContext(r.Context(), store.Rebind(`
SELECT AVG(`+slaMinutesExpr(store)+`)
FROM requests
WHERE `+prevWhere+`
  AND `+atendidaCond+`
  AND requested_at IS NOT NULL AND TRIM(CAST(requested_at AS TEXT)) <> ''
  AND served_at IS NOT NULL AND TRIM(CAST(served_at AS TEXT)) <> ''
  AND `+slaNonNegativeCond(store)+``), prevArgs...).Scan(&prevSla)
	var prevLance sql.NullFloat64
	_ = store.DB.QueryRowContext(r.Context(), store.Rebind(`
SELECT AVG(bid_percent) FROM requests
WHERE `+prevWhere+`
  AND bid_percent IS NOT NULL`), prevArgs...).Scan(&prevLance)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"cards": map[string]any{
			"total_solicitacoes":              total,
			"atendidas":                       atendidas,
			"nao_atendidas":                   naoAtendidas,
			"backlog_atual":                   backlogAtual,
			"taxa_atendimento":                taxa,
			"sla_medio_min":                   nullFloat64(slaAvgMin),
			"sla_mediana_min":                 slaMediana,
			"sla_p95_min":                     slaP95,
			"lance_medio":                     nullFloat64(lanceAvg),
			"erros_reserva":                   errosReserva,
			"taxa_erro_reserva":               taxaErro,
			"tempo_desde_ultima_atendida_seg": ultimaAtendidaDelta,
			"ultima_atendida_em":              ultimaAtendidaEm,
		},
		"comparativo": map[string]any{
			"total_solicitacoes": dashboardMetricCompare{Atual: float64(total), Anterior: float64(prevTotal), DeltaPct: deltaPct(float64(total), float64(prevTotal))},
			"atendidas":          dashboardMetricCompare{Atual: float64(atendidas), Anterior: float64(prevAtendidas), DeltaPct: deltaPct(float64(atendidas), float64(prevAtendidas))},
			"nao_atendidas":      dashboardMetricCompare{Atual: float64(naoAtendidas), Anterior: float64(prevNaoAtendidas), DeltaPct: deltaPct(float64(naoAtendidas), float64(prevNaoAtendidas))},
			"taxa_atendimento":   dashboardMetricCompare{Atual: taxa, Anterior: prevTaxa, DeltaPct: deltaPct(taxa, prevTaxa)},
			"sla_medio_min":      dashboardMetricCompare{Atual: nullFloat64(slaAvgMin), Anterior: nullFloat64(prevSla), DeltaPct: deltaPct(nullFloat64(slaAvgMin), nullFloat64(prevSla))},
			"lance_medio":        dashboardMetricCompare{Atual: nullFloat64(lanceAvg), Anterior: nullFloat64(prevLance), DeltaPct: deltaPct(nullFloat64(lanceAvg), nullFloat64(prevLance))},
		},
		"filiais":          filiais,
		"vendedores":       vendedoresTop,
		"vendedores_taxa":  vendedoresTaxa,
		"grupos_atendidos": gruposAtendidos,
		"filiais_backlog":  filiaisBacklog,
		"serie":            serie,
		"serie_hora":       serieHora,
		"filiais_filtro":   filiaisFiltro,
		"periodo": map[string]any{
			"preset": period,
			"from":   fromDate,
			"to":     toDate,
			"branch": filialSel,
		},
	})
}

func (a *app) handleDashboardDetails(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.currentSession(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, runResponse{OK: false, Message: "Nao autenticado"})
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	_, fromDate, toDate, err := dashboardPeriodRange(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	value := strings.TrimSpace(r.URL.Query().Get("value"))
	filialSel := strings.TrimSpace(r.URL.Query().Get("branch"))
	supervisorScope := ""
	supervisorArgs := make([]any, 0)
	if sess != nil && isSupervisorRole(sess.Role) {
		if strings.TrimSpace(sess.branch) != "" {
			filialSel = strings.TrimSpace(sess.branch)
		}
		supervisorScope, supervisorArgs, err = buildSupervisorRequestsScope(r.Context(), store, sess)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
	}
	if sess != nil && isGerenteRole(sess.Role) && strings.TrimSpace(sess.branch) != "" {
		filialSel = strings.TrimSpace(sess.branch)
	}
	if sess != nil && !strings.EqualFold(strings.TrimSpace(sess.Role), "admin") && !isSupervisorRole(sess.Role) && !isGerenteRole(sess.Role) && strings.TrimSpace(sess.branch) != "" {
		filialSel = strings.TrimSpace(sess.branch)
	}

	atendidaCond := `( (status IS NOT NULL AND LOWER(TRIM(status)) LIKE 'atendid%') OR
		(served_at IS NOT NULL AND TRIM(CAST(served_at AS TEXT)) <> '') OR
		(served_date IS NOT NULL AND TRIM(CAST(served_date AS TEXT)) <> '') )`
	baseDateExpr := `COALESCE(NULLIF(SUBSTR(TRIM(CAST(requested_date AS TEXT)), 1, 10), ''), SUBSTR(TRIM(CAST(requested_at AS TEXT)), 1, 10))`
	conds := []string{
		baseDateExpr + ` IS NOT NULL`,
		`TRIM(` + baseDateExpr + `) <> ''`,
		baseDateExpr + ` >= ?`,
		baseDateExpr + ` <= ?`,
	}
	args := []any{fromDate, toDate}
	if filialSel != "" {
		conds = append(conds, `COALESCE(TRIM(branch),'') = ?`)
		args = append(args, filialSel)
	}
	if supervisorScope != "" {
		conds = append(conds, supervisorScope)
		args = append(args, supervisorArgs...)
	}
	switch kind {
	case "total":
		// base only
	case "atendidas":
		conds = append(conds, atendidaCond)
	case "nao_atendidas":
		conds = append(conds, `NOT (`+atendidaCond+`)`)
	case "branch":
		conds = append(conds, `COALESCE(NULLIF(TRIM(branch), ''), '(Sem filial)') = ?`)
		args = append(args, value)
	case "seller_name":
		conds = append(conds, `COALESCE(NULLIF(TRIM(seller_name), ''), '(Sem seller_name)') = ?`)
		args = append(args, value)
	case "group_code":
		conds = append(conds, `COALESCE(NULLIF(CAST(served_group AS TEXT), ''), '(Sem group_code)') = ?`)
		args = append(args, value)
	case "filial_backlog":
		conds = append(conds, `NOT (`+atendidaCond+`)`)
		conds = append(conds, `COALESCE(NULLIF(TRIM(branch), ''), '(Sem filial)') = ?`)
		args = append(args, value)
	default:
		writeErr(w, http.StatusBadRequest, fmt.Errorf("kind invalido"))
		return
	}

	query := `SELECT CAST(COALESCE(id, 0) AS INTEGER) AS id,
requested_at,
requested_date,
requested_time,
CAST(COALESCE(branch, '') AS TEXT) AS branch,
CAST(COALESCE(seller_name, '') AS TEXT) AS seller_name,
CAST(COALESCE(cpf, '') AS TEXT) AS cpf,
CAST(COALESCE(model_name, '') AS TEXT) AS model_name,
CAST(COALESCE(licensed, '') AS TEXT) AS plan,
CASE WHEN TRIM(COALESCE(CAST(installments AS TEXT), '')) = '' THEN NULL ELSE CAST(REPLACE(CAST(installments AS TEXT), ',', '.') AS INTEGER) END AS installments,
CASE WHEN TRIM(COALESCE(CAST(bid_percent AS TEXT), '')) = '' THEN NULL ELSE CAST(REPLACE(CAST(bid_percent AS TEXT), ',', '.') AS REAL) END AS bid_percent,
CAST(COALESCE(with_restriction, '') AS TEXT) AS with_restriction,
CASE WHEN TRIM(COALESCE(CAST(group_code AS TEXT), '')) = '' THEN NULL ELSE CAST(CAST(group_code AS TEXT) AS INTEGER) END AS group_code,
CAST(COALESCE(notes, '') AS TEXT) AS notes,
CASE WHEN TRIM(COALESCE(CAST(requested_quota_id AS TEXT), '')) = '' THEN NULL ELSE CAST(CAST(requested_quota_id AS TEXT) AS INTEGER) END AS requested_quota_id,
CASE WHEN TRIM(COALESCE(CAST(served_group AS TEXT), '')) = '' THEN NULL ELSE CAST(CAST(served_group AS TEXT) AS INTEGER) END AS served_group,
CAST(COALESCE(quota_rd, '') AS TEXT) AS quota_rd,
served_at,
served_date,
served_time,
CAST(COALESCE(status, '') AS TEXT) AS status,
CAST(COALESCE(contemplation_bid, '') AS TEXT) AS contemplation_bid
FROM requests
WHERE ` + strings.Join(conds, " AND ") + `
ORDER BY id DESC
LIMIT 200`
	rows, err := store.DB.QueryContext(r.Context(), store.Rebind(query), args...)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	out := make([]solicitacaoPayload, 0, 200)
	for rows.Next() {
		var rec db.SolicitacaoRecord
		if err := rows.Scan(
			&rec.ID, &rec.DataHoraSolicitacao, &rec.DataSolicitacao, &rec.HoraSolicitacao, &rec.Filial, &rec.Vendedor, &rec.CPF, &rec.Modelo, &rec.Plano, &rec.QtdeParcelas,
			&rec.PercLance, &rec.ComRestricao, &rec.Grupo, &rec.Notes, &rec.IDCota, &rec.GrupoAtendido, &rec.CotaRD,
			&rec.DataHoraAtendimento, &rec.DataAtendimento, &rec.HoraAtendimento, &rec.Situacao, &rec.LanceContemplacao,
		); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		out = append(out, solicitacaoToPayload(&rec))
	}
	if err := rows.Err(); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func previousPeriodRange(fromDate, toDate string) (string, string, error) {
	fromT, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		return "", "", err
	}
	toT, err := time.Parse("2006-01-02", toDate)
	if err != nil {
		return "", "", err
	}
	days := int(toT.Sub(fromT).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	prevTo := fromT.AddDate(0, 0, -1)
	prevFrom := prevTo.AddDate(0, 0, -(days - 1))
	return prevFrom.Format("2006-01-02"), prevTo.Format("2006-01-02"), nil
}

func deltaPct(current, previous float64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return ((current - previous) / previous) * 100.0
}

func queryInt64Ctx(ctx context.Context, store *db.Store, query string, args ...any) (int64, error) {
	var n sql.NullInt64
	if err := store.DB.QueryRowContext(ctx, store.Rebind(query), args...).Scan(&n); err != nil {
		return 0, err
	}
	if !n.Valid {
		return 0, nil
	}
	return n.Int64, nil
}

func queryDashboardRank(ctx context.Context, store *db.Store, query string, args ...any) ([]dashboardRankItem, error) {
	rows, err := store.DB.QueryContext(ctx, store.Rebind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]dashboardRankItem, 0, 10)
	for rows.Next() {
		var item dashboardRankItem
		if err := rows.Scan(&item.Nome, &item.Total, &item.Atendidas); err != nil {
			return nil, err
		}
		if item.Total > 0 {
			item.Taxa = (float64(item.Atendidas) * 100.0) / float64(item.Total)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func queryDashboardSerie(ctx context.Context, store *db.Store, atendidaCond, dateWhere string, args ...any) ([]dashboardSerieItem, error) {
	rows, err := store.DB.QueryContext(ctx, store.Rebind(`
WITH base AS (
  SELECT
    COALESCE(NULLIF(SUBSTR(TRIM(CAST(requested_date AS TEXT)), 1, 10), ''), SUBSTR(TRIM(CAST(requested_at AS TEXT)), 1, 10)) AS data_ref,
    CASE WHEN `+atendidaCond+` THEN 1 ELSE 0 END AS atendida
  FROM requests
  WHERE `+dateWhere+`
)
SELECT data_ref, COUNT(1) AS solicitadas, SUM(atendida) AS atendidas
FROM base
WHERE data_ref IS NOT NULL AND TRIM(CAST(data_ref AS TEXT)) <> ''
GROUP BY data_ref
ORDER BY data_ref DESC
 LIMIT 14`), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tmp := make([]dashboardSerieItem, 0, 14)
	for rows.Next() {
		var item dashboardSerieItem
		if err := rows.Scan(&item.Data, &item.Solicitadas, &item.Atendidas); err != nil {
			return nil, err
		}
		tmp = append(tmp, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Entrega em ordem crescente para o grÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡fico.
	out := make([]dashboardSerieItem, 0, len(tmp))
	for i := len(tmp) - 1; i >= 0; i-- {
		out = append(out, tmp[i])
	}
	return out, nil
}

func queryDashboardSerieHora(ctx context.Context, store *db.Store, atendidaCond, whereClause, hourExpr string, args ...any) ([]dashboardHourItem, error) {
	rows, err := store.DB.QueryContext(ctx, store.Rebind(`
SELECT SUBSTR(`+hourExpr+`, 1, 2) AS hora,
       COUNT(1) AS solicitadas,
       SUM(CASE WHEN `+atendidaCond+` THEN 1 ELSE 0 END) AS atendidas
FROM requests
WHERE `+whereClause+`
  AND `+hourExpr+` IS NOT NULL
  AND TRIM(`+hourExpr+`) <> ''
GROUP BY hora
ORDER BY hora ASC`), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]dashboardHourItem, 0, 24)
	for rows.Next() {
		var h string
		var item dashboardHourItem
		if err := rows.Scan(&h, &item.Solicitadas, &item.Atendidas); err != nil {
			return nil, err
		}
		h = strings.TrimSpace(h)
		if len(h) == 1 {
			h = "0" + h
		}
		if len(h) >= 2 {
			item.Hora = h[:2] + ":00"
		} else {
			item.Hora = "00:00"
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func querySLAMinutes(ctx context.Context, store *db.Store, whereClause string, args ...any) ([]float64, error) {
	rows, err := store.DB.QueryContext(ctx, store.Rebind(`
SELECT (`+slaMinutesExpr(store)+`) AS sla_min
FROM requests
WHERE `+whereClause+`
  AND requested_at IS NOT NULL AND TRIM(CAST(requested_at AS TEXT)) <> ''
  AND served_at IS NOT NULL AND TRIM(CAST(served_at AS TEXT)) <> ''
  AND `+slaNonNegativeCond(store)+``), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]float64, 0, 256)
	for rows.Next() {
		var v sql.NullFloat64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		if v.Valid {
			out = append(out, v.Float64)
		}
	}
	return out, rows.Err()
}

func slaMinutesExpr(store *db.Store) string {
	return "EXTRACT(EPOCH FROM (served_at - requested_at)) / 60.0"
}

func slaNonNegativeCond(store *db.Store) string {
	return "served_at >= requested_at"
}

func percentilesFromSamples(samples []float64) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	sorted := make([]float64, len(samples))
	copy(sorted, samples)
	sort.Float64s(sorted)
	return percentileLinear(sorted, 0.50), percentileLinear(sorted, 0.95)
}

func percentileLinear(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := p * float64(len(sorted)-1)
	lo := int(pos)
	hi := lo + 1
	if hi >= len(sorted) {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo] + (sorted[hi]-sorted[lo])*frac
}

func queryTempoDesdeUltimaAtendida(ctx context.Context, dbConn *sql.DB, branch string, extraScope string, extraArgs []any) (int64, string, error) {
	conds := []string{
		`(served_at IS NOT NULL AND TRIM(CAST(served_at AS TEXT)) <> '') OR
		 (served_date IS NOT NULL AND TRIM(CAST(served_date AS TEXT)) <> '')`,
	}
	args := make([]any, 0, 1+len(extraArgs))
	if strings.TrimSpace(branch) != "" {
		conds = append(conds, `COALESCE(TRIM(branch),'') = ?`)
		args = append(args, strings.TrimSpace(branch))
	}
	if strings.TrimSpace(extraScope) != "" {
		conds = append(conds, extraScope)
		args = append(args, extraArgs...)
	}
	row := dbConn.QueryRowContext(ctx, `
SELECT MAX(
  COALESCE(
    NULLIF(TRIM(CAST(served_at AS TEXT)), ''),
    CASE
      WHEN served_date IS NOT NULL AND TRIM(CAST(served_date AS TEXT)) <> '' THEN
        TRIM(CAST(served_date AS TEXT)) || ' ' || COALESCE(NULLIF(TRIM(CAST(served_time AS TEXT)), ''), '00:00:00')
      ELSE NULL
    END
  )
) FROM requests
WHERE `+strings.Join(conds, " AND "), args...)
	var raw sql.NullString
	if err := row.Scan(&raw); err != nil {
		return 0, "", err
	}
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return 0, "", nil
	}
	t, ok := parseFlexibleDateTime(strings.TrimSpace(raw.String))
	if !ok {
		return 0, raw.String, nil
	}
	sec := int64(time.Since(t).Seconds())
	if sec < 0 {
		sec = 0
	}
	return sec, t.Format("2006-01-02 15:04:05"), nil
}

func parseFlexibleDateTime(raw string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, utcMinus3Loc); err == nil {
			// ForÃƒÂ§a o horÃƒÂ¡rio do "relÃƒÂ³gio" para o fuso de BrasÃƒÂ­lia, mesmo que a string tenha 'Z' ou offset.
			// Isso garante que comparamos os nÃƒÂºmeros que o usuÃƒÂ¡rio vÃƒÂª na tela.
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), utcMinus3Loc)
			return t, true
		}
	}
	return time.Time{}, false
}

func formatDateTimeBRNoSeconds(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	t, ok := parseFlexibleDateTime(s)
	if !ok {
		return s
	}
	return t.Format("02/01/2006 15:04")
}

func formatSLADuration(startRaw, endRaw string) string {
	start, okStart := parseFlexibleDateTime(strings.TrimSpace(startRaw))
	end, okEnd := parseFlexibleDateTime(strings.TrimSpace(endRaw))
	if !okStart || !okEnd {
		return ""
	}
	// As datas jÃƒÂ¡ sÃƒÂ£o interpretadas em utcMinus3Loc pela parseFlexibleDateTime
	if end.Before(start) {
		return ""
	}
	totalMin := int(end.Sub(start).Minutes())
	h := totalMin / 60
	m := totalMin % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

func queryDistinctFiliais(ctx context.Context, dbConn *sql.DB) ([]string, error) {
	rows, err := dbConn.QueryContext(ctx, `
SELECT DISTINCT TRIM(branch) AS branch
FROM requests
WHERE branch IS NOT NULL AND TRIM(branch) <> ''
ORDER BY branch ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, 64)
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func dashboardPeriodRange(r *http.Request) (string, string, string, error) {
	now := time.Now().In(utcMinus3Loc)
	today := now.Format("2006-01-02")
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))
	if from == "" {
		from = monthStart
	}
	if to == "" {
		to = today
	}
	if _, err := time.Parse("2006-01-02", from); err != nil {
		return "", "", "", fmt.Errorf("from invalido")
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		return "", "", "", fmt.Errorf("to invalido")
	}
	if from > to {
		return "", "", "", fmt.Errorf("from deve ser menor ou igual a to")
	}
	return "range", from, to, nil
}

func reservedCotaRecordToPayload(r *db.ReservedCotaRecord) reservedCotaPayload {
	return reservedCotaPayload{
		ID:                 r.ID,
		UsuarioReserva:     r.UsuarioReserva,
		NumDocumentoPessoa: r.NumDocumentoPessoa,
		CodGrupo:           r.CodGrupo,
		CotaRD:             r.CotaRD,
		CodModelo:          r.CodModelo,
		IDCotaReposicao:    r.IDCotaReposicao,
		CreatedAt:          strings.TrimSpace(r.CreatedAt.String),
	}
}

func (a *app) handleReservedCotasSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	_, fromDate, toDate, err := dashboardPeriodRange(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, err := store.SearchReservedCotas(r.Context(), search, fromDate, toDate, 1000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]reservedCotaPayload, 0, len(records))
	for i := range records {
		out = append(out, reservedCotaRecordToPayload(&records[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "count": len(out)})
}

func (a *app) handleReservedCotaDelete(w http.ResponseWriter, r *http.Request) {
	var p reservedCotaPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	if err := store.DeleteReservedCota(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Registro removido"})
}

func (a *app) handleReservedCotasDeleteBatch(w http.ResponseWriter, r *http.Request) {
	var p reservedCotaIDsPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(p.IDs) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nenhum registro selecionado"))
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	count, err := store.DeleteReservedCotasByIDs(r.Context(), p.IDs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"count":   count,
		"message": fmt.Sprintf("Registros removidos: %d", count),
	})
}

func manualNotificationRecordToPayload(r *db.ManualNotificationRecord) manualNotificationPayload {
	out := manualNotificationPayload{
		ID:          r.ID,
		CPF:         r.CPF,
		Vendedor:    r.Vendedor,
		Filial:      r.Filial,
		Canal:       r.Canal,
		Mensagem:    r.Mensagem,
		Status:      r.Status,
		UsuarioAcao: r.UsuarioAcao,
	}
	if r.SolicitacaoID.Valid {
		out.SolicitacaoID = r.SolicitacaoID.Int64
	}
	if r.CopiadaEm.Valid {
		out.CopiadaEm = strings.TrimSpace(r.CopiadaEm.String)
	}
	if r.EnviadaEm.Valid {
		out.EnviadaEm = strings.TrimSpace(r.EnviadaEm.String)
	}
	if r.CreatedAt.Valid {
		out.CreatedAt = strings.TrimSpace(r.CreatedAt.String)
	}
	if r.UpdatedAt.Valid {
		out.UpdatedAt = strings.TrimSpace(r.UpdatedAt.String)
	}
	return out
}

func buildManualNotificationMessage(item solicitacaoSuccessItemPayload) string {
	nome := strings.TrimSpace(item.Nome)
	if nome == "" {
		nome = "client_name"
	}
	branch := strings.ToUpper(strings.TrimSpace(item.Filial))
	group_code := strings.TrimSpace(item.GrupoAtendido)
	if group_code == "" {
		group_code = "-"
	}
	quota_rd := strings.TrimSpace(item.CotaRD)
	if quota_rd == "" {
		quota_rd = "-"
	}
	installments := strings.TrimSpace(item.QtdeParcelas)
	if installments == "" {
		installments = "-"
	}
	model_name := strings.TrimSpace(item.Modelo)
	if model_name == "" {
		model_name = "-"
	}
	licenciada := strings.TrimSpace(item.Licenciada)
	if licenciada == "" {
		licenciada = "-"
	}
	comRestricao := strings.TrimSpace(item.ComRestricao)
	if comRestricao == "" {
		comRestricao = "-"
	}
	solicitada := strings.TrimSpace(item.SolicitadaEm)
	atendida := strings.TrimSpace(item.AtendidaEm)
	sla := strings.TrimSpace(item.SLA)
	lines := []string{
		"Ol\u00e1, " + nome + func() string {
			if branch != "" {
				return " - " + branch
			}
			return ""
		}(),
		"Sua solicita\u00e7\u00e3o foi atendida na seguinte condi\u00e7\u00e3o:",
	}
	if branch != "TER" {
		lines = append(lines, "Grupo: "+group_code)
	}
	lines = append(lines, "Cota-R-D: "+quota_rd)
	lines = append(lines, "Parcelas: "+installments)
	lines = append(lines, "Modelo: "+model_name)
	lines = append(lines, "Licenciada: "+licenciada)
	lines = append(lines, "Restri\u00e7\u00e3o: "+comRestricao)
	if solicitada != "" {
		lines = append(lines, "Solicitada: "+solicitada)
	}
	if atendida != "" {
		lines = append(lines, "Atendida: "+atendida)
	}
	if sla != "" {
		lines = append(lines, "SLA: "+sla)
	}
	return strings.Join(lines, "\n")
}

func (a *app) enqueueManualNotifications(ctx context.Context, store *db.Store, items []solicitacaoSuccessItemPayload) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	rows := make([]db.ManualNotificationRecord, 0, len(items))
	for _, it := range items {
		msg := strings.TrimSpace(buildManualNotificationMessage(it))
		if msg == "" {
			continue
		}
		row := db.ManualNotificationRecord{
			SolicitacaoID: sql.NullInt64{Int64: it.ID, Valid: it.ID > 0},
			CPF:           strings.TrimSpace(it.CPF),
			Vendedor:      strings.TrimSpace(it.Nome),
			Filial:        strings.TrimSpace(it.Filial),
			Canal:         "whatsapp",
			Mensagem:      msg,
			Status:        "pendente",
		}
		rows = append(rows, row)
	}
	return store.InsertManualNotifications(ctx, rows)
}

func (a *app) handleManualNotificationsSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	solicitacaoID, _ := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("solicitacao_id")), 10, 64)
	_, fromDate, toDate, err := dashboardPeriodRange(r)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	sess, ok := a.currentSession(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("nao autenticado"))
		return
	}
	cpfOnly := ""
	roleNorm := strings.ToLower(strings.TrimSpace(sess.Role))
	if roleNorm == "seller_name" || roleNorm == "vendedor" {
		cpfOnly = strings.TrimSpace(onlyDigitsLocal(sess.CPF))
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, err := store.SearchManualNotifications(r.Context(), search, status, fromDate, toDate, solicitacaoID, cpfOnly, 1000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]manualNotificationPayload, 0, len(records))
	for i := range records {
		out = append(out, manualNotificationRecordToPayload(&records[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func (a *app) handleManualNotificationsSetStatus(w http.ResponseWriter, r *http.Request) {
	var p manualNotificationIDsPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(p.IDs) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nenhuma notificacao selecionada"))
		return
	}
	status := strings.ToLower(strings.TrimSpace(p.Status))
	if status != "copiada" && status != "enviada_manual" && status != "cancelada" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("status invalido"))
		return
	}
	sess, ok := a.currentSession(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("nao autenticado"))
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	usuarioAcao := strings.TrimSpace(sess.DisplayName)
	if usuarioAcao == "" {
		usuarioAcao = strings.TrimSpace(sess.Username)
	}
	count, err := store.MarkManualNotificationsStatus(r.Context(), p.IDs, status, usuarioAcao)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"count":   count,
		"message": fmt.Sprintf("NotificaÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Âµes atualizadas: %d", count),
	})
}

func idsGrupoDisponivelToPayload(r *db.IDsGrupoDisponivelRecord) idsGrupoDisponivelPayload {
	out := idsGrupoDisponivelPayload{
		ID:            r.ID,
		IDGrupo:       r.IDGrupo,
		Produto:       r.Produto,
		Vencimento:    r.Vencimento,
		Prazo:         r.Prazo,
		Tipo:          r.Tipo,
		Grupo:         r.Grupo,
		Cota:          r.Cota,
		R:             r.R,
		D:             r.D,
		Booked:        r.Booked,
		Participantes: r.Participantes,
		Failed:        r.Failed,
	}
	if r.CreatedAt.Valid {
		out.CreatedAt = r.CreatedAt.String
	}
	return out
}

func (a *app) parcelasCalculadasPorGrupos(ctx context.Context, store *db.Store, grupos []int64) (map[int64]int64, error) {
	result := make(map[int64]int64)
	groupRecords, err := store.GetGruposAtivosByGrupos(ctx, grupos)
	if err != nil {
		return nil, err
	}
	holidays, err := store.ListActiveNationalHolidayDates(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().In(utcMinus3Loc)
	for group_code, rec := range groupRecords {
		r := rec
		result[group_code] = calculateParcelasFromGrupoAtivo(&r, holidays, now)
	}
	return result, nil
}

func (a *app) handleIDsGruposDisponiveisSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	column := strings.TrimSpace(r.URL.Query().Get("column"))
	offset, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("offset")))
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	if limit <= 0 {
		limit = 200
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, total, err := store.SearchIDsGruposDisponiveis(r.Context(), search, column, offset, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]idsGrupoDisponivelPayload, 0, len(records))
	for i := range records {
		out = append(out, idsGrupoDisponivelToPayload(&records[i]))
	}
	grupos := make([]int64, 0, len(records))
	for i := range records {
		if records[i].Grupo > 0 {
			grupos = append(grupos, records[i].Grupo)
		}
	}
	if len(grupos) > 0 {
		calcMap, err := a.parcelasCalculadasPorGrupos(r.Context(), store, grupos)
		if err == nil {
			for i := range out {
				if v, ok := calcMap[out[i].Grupo]; ok {
					out[i].ParcelasCalc = v
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  out,
		"count":  len(out),
		"total":  total,
		"offset": offset,
		"limit":  limit,
	})
}

func (a *app) handleIDsGrupoDisponivelGet(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	record, err := store.GetIDsGrupoDisponivelByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("registro nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := idsGrupoDisponivelToPayload(record)
	if record.Grupo > 0 {
		if calcMap, calcErr := a.parcelasCalculadasPorGrupos(r.Context(), store, []int64{record.Grupo}); calcErr == nil {
			if v, ok := calcMap[record.Grupo]; ok {
				out.ParcelasCalc = v
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *app) handleCheckAvailableGroup(w http.ResponseWriter, r *http.Request) {
	code, _ := strconv.Atoi(r.URL.Query().Get("code"))
	if code <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("codigo invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rec, err := store.GetAvailableGroupInfoByCode(r.Context(), code)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "found": false})
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "found": true, "item": rec})
}

func (a *app) handleIDsGrupoDisponivelSave(w http.ResponseWriter, r *http.Request) {
	var p idsGrupoDisponivelPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	rec := db.IDsGrupoDisponivelRecord{
		ID:            p.ID,
		IDGrupo:       p.IDGrupo,
		Produto:       p.Produto,
		Vencimento:    p.Vencimento,
		Prazo:         p.Prazo,
		Tipo:          p.Tipo,
		Grupo:         p.Grupo,
		Cota:          p.Cota,
		R:             p.R,
		D:             p.D,
		Booked:        p.Booked,
		CreatedAt:     sql.NullString{String: strings.TrimSpace(p.CreatedAt), Valid: true},
		Participantes: p.Participantes,
		Failed:        p.Failed,
	}
	if !rec.CreatedAt.Valid || strings.TrimSpace(rec.CreatedAt.String) == "" {
		rec.CreatedAt = sql.NullString{String: nowDateTimeUTCMinus3(), Valid: true}
	}
	if p.ID > 0 {
		if err := store.UpdateIDsGrupoDisponivel(r.Context(), rec); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Registro atualizado"})
		return
	}
	newID, err := store.InsertIDsGrupoDisponivel(r.Context(), rec)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"id":      newID,
		"message": "Registro criado",
	})
}

func (a *app) handleIDsGrupoDisponivelDelete(w http.ResponseWriter, r *http.Request) {
	var p idsGrupoDisponivelPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := store.DeleteIDsGrupoDisponivel(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Registro removido"})
}

func (a *app) handleIDsGruposDisponiveisDeleteBatch(w http.ResponseWriter, r *http.Request) {
	var p idsGrupoDisponivelIDsPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(p.IDs) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nenhum registro selecionado"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	count, err := store.DeleteIDsGruposDisponiveisByIDs(r.Context(), p.IDs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"count":   count,
		"message": fmt.Sprintf("Registros removidos: %d", count),
	})
}

func modeloToPayload(r *db.ModeloRecord) modeloPayload {
	return modeloPayload{
		ID:       r.ID,
		IDModelo: r.IDModelo,
		Modelo:   r.Modelo,
		Status:   r.Status,
	}
}

func produtoToPayload(r *db.ProdutoRecord) produtoPayload {
	return produtoPayload{
		ID:        r.ID,
		IDProduto: r.IDProduto,
		Produto:   r.Produto,
		Status:    r.Status,
	}
}

func assembleiaToPayload(r *db.AssembleiaRecord) assembleiaPayload {
	out := assembleiaPayload{
		ID:               r.ID,
		CotaRD:           r.CotaRD,
		TipoContemplacao: r.TipoContemplacao,
		ClientName:       r.ClientName,
		Vendedor:         r.Vendedor,
		GrupoCotaRD:      r.GrupoCotaRD,
	}
	if r.DataContemplacao.Valid {
		out.DataContemplacao = strings.TrimSpace(r.DataContemplacao.String)
	}
	if r.DataDesclassificao.Valid {
		out.DataDesclassificao = strings.TrimSpace(r.DataDesclassificao.String)
	}
	if r.PercLance.Valid {
		out.PercLance = strconv.FormatFloat(r.PercLance.Float64, 'f', -1, 64)
	}
	if r.Grupo.Valid {
		out.Grupo = strconv.FormatInt(r.Grupo.Int64, 10)
	}
	if r.LoteriaFederal.Valid {
		out.LoteriaFederal = strconv.FormatInt(r.LoteriaFederal.Int64, 10)
	}
	return out
}

func grupoAtivoToPayload(r *db.GrupoAtivoRecord) grupoAtivoPayload {
	out := grupoAtivoPayload{
		ID:               r.ID,
		Grupo:            r.Grupo,
		Vencimento:       r.Vencimento,
		QtdParticipantes: r.QtdParticipantes,
		Plano:            r.Plano,
		Prazo:            r.Prazo,
		TipoGrupo:        r.TipoGrupo,
		Modelos:          r.Modelos,
		Status:           r.Status,
	}
	if r.DataAssembleiaInaugural.Valid {
		out.DataAssembleiaInaugural = strings.TrimSpace(r.DataAssembleiaInaugural.String)
	}
	if r.PercLance.Valid {
		out.PercLance = strconv.FormatFloat(r.PercLance.Float64, 'f', -1, 64)
	}
	if r.CreatedAt.Valid {
		out.CreatedAt = strings.TrimSpace(r.CreatedAt.String)
	}
	if r.UpdatedAt.Valid {
		out.UpdatedAt = strings.TrimSpace(r.UpdatedAt.String)
	}
	return out
}

func (a *app) handleModelosSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, err := store.SearchModelos(r.Context(), search, 1000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]modeloPayload, 0, len(records))
	for i := range records {
		out = append(out, modeloToPayload(&records[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func (a *app) handleModeloGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rec, err := store.GetModeloByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("registro nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, modeloToPayload(rec))
}

func (a *app) handleModeloSave(w http.ResponseWriter, r *http.Request) {
	var p modeloPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	id, err := store.SaveModelo(r.Context(), db.ModeloRecord{
		ID:       p.ID,
		IDModelo: p.IDModelo,
		Modelo:   strings.TrimSpace(p.Modelo),
		Status:   strings.TrimSpace(p.Status),
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	msg := "Modelo atualizado"
	if p.ID <= 0 {
		msg = "Modelo criado"
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "message": msg})
}

func (a *app) handleModeloDelete(w http.ResponseWriter, r *http.Request) {
	var p modeloPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := store.DeleteModelo(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Modelo removido"})
}

func (a *app) handleProdutosSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, err := store.SearchProdutos(r.Context(), search, 1000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]produtoPayload, 0, len(records))
	for i := range records {
		out = append(out, produtoToPayload(&records[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func (a *app) handleProdutoGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rec, err := store.GetProdutoByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("registro nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, produtoToPayload(rec))
}

func (a *app) handleProdutoSave(w http.ResponseWriter, r *http.Request) {
	var p produtoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	id, err := store.SaveProduto(r.Context(), db.ProdutoRecord{
		ID:        p.ID,
		IDProduto: p.IDProduto,
		Produto:   strings.TrimSpace(p.Produto),
		Status:    strings.TrimSpace(p.Status),
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	msg := "Produto atualizado"
	if p.ID <= 0 {
		msg = "Produto criado"
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "message": msg})
}

func (a *app) handleProdutoDelete(w http.ResponseWriter, r *http.Request) {
	var p produtoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := store.DeleteProduto(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Produto removido"})
}

func (a *app) handleAssembleiasSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, err := store.SearchAssembleias(r.Context(), search, 2000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	out := make([]assembleiaPayload, 0, len(records))
	for i := range records {
		out = append(out, assembleiaToPayload(&records[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func (a *app) handleAssembleiaGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rec, err := store.GetAssembleiaByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("registro nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, assembleiaToPayload(rec))
}

func (a *app) handleAssembleiaPercLance(w http.ResponseWriter, r *http.Request) {
	group_code, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("group_code")), 10, 64)
	if err != nil || group_code <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("group_code invalido"))
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	var perc sql.NullFloat64
	err = store.DB.QueryRowContext(
		r.Context(),
		store.Rebind(`SELECT CASE WHEN TRIM(COALESCE(CAST(bid_percent AS TEXT), '')) = '' THEN NULL
		             ELSE CAST(REPLACE(CAST(bid_percent AS TEXT), ',', '.') AS REAL)
		        END AS bid_percent
		   FROM assemblies
		  WHERE COALESCE(CAST(group_code AS TEXT), '')=?
		     OR CAST(COALESCE(group_quota_rd, '') AS TEXT) LIKE ?
		  ORDER BY id DESC
		  LIMIT 1`),
		strconv.FormatInt(group_code, 10),
		strconv.FormatInt(group_code, 10)+"-%",
	).Scan(&perc)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":          true,
				"group_code":  group_code,
				"bid_percent": "",
				"found":       false,
			})
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	value := ""
	if perc.Valid {
		value = strconv.FormatFloat(perc.Float64, 'f', -1, 64)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"group_code":  group_code,
		"bid_percent": value,
		"found":       perc.Valid,
	})
}

func (a *app) handleAssembleiaSave(w http.ResponseWriter, r *http.Request) {
	var p assembleiaPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 &&
		strings.TrimSpace(p.CotaRD) == "" &&
		strings.TrimSpace(p.DataContemplacao) == "" &&
		strings.TrimSpace(p.TipoContemplacao) == "" &&
		strings.TrimSpace(p.DataDesclassificao) == "" &&
		strings.TrimSpace(p.ClientName) == "" &&
		strings.TrimSpace(p.PercLance) == "" &&
		strings.TrimSpace(p.Vendedor) == "" &&
		strings.TrimSpace(p.Grupo) == "" &&
		strings.TrimSpace(p.LoteriaFederal) == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("preencha ao menos um campo para salvar"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	percLance, err := parseNullFloat64(p.PercLance)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("bid_percent invalido"))
		return
	}
	group_code, err := parseNullInt64(p.Grupo)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("group_code invalido"))
		return
	}
	lotFed, err := parseNullInt64(p.LoteriaFederal)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("federal_lottery invalido"))
		return
	}

	id, err := store.SaveAssembleia(r.Context(), db.AssembleiaRecord{
		ID:                 p.ID,
		CotaRD:             strings.TrimSpace(p.CotaRD),
		DataContemplacao:   sql.NullString{String: strings.TrimSpace(p.DataContemplacao), Valid: strings.TrimSpace(p.DataContemplacao) != ""},
		TipoContemplacao:   strings.TrimSpace(p.TipoContemplacao),
		DataDesclassificao: sql.NullString{String: strings.TrimSpace(p.DataDesclassificao), Valid: strings.TrimSpace(p.DataDesclassificao) != ""},
		ClientName:         strings.TrimSpace(p.ClientName),
		PercLance:          percLance,
		Vendedor:           strings.TrimSpace(p.Vendedor),
		Grupo:              group_code,
		LoteriaFederal:     lotFed,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	msg := "assemblies atualizada"
	if p.ID <= 0 {
		msg = "assemblies criada"
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "message": msg})
}

func (a *app) handleAssembleiaDelete(w http.ResponseWriter, r *http.Request) {
	var p assembleiaPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := store.DeleteAssembleia(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "assemblies removida"})
}

func (a *app) handleGruposAtivosSearch(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	column := strings.TrimSpace(r.URL.Query().Get("column"))
	filters := strings.TrimSpace(r.URL.Query().Get("filters"))

	// Filtro híbrido: "parc" é calculado dinamicamente em memória.
	// Removemos "parc" da query SQL e aplicamos após calcular ParcelasCalculadas.
	sqlFilters, parcFilters := splitGAFiltersForSQL(filters)
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	records, err := store.SearchGruposAtivos(r.Context(), search, column, sqlFilters, 2000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	holidays, _ := store.ListActiveNationalHolidayDates(r.Context())
	now := time.Now().In(utcMinus3Loc)
	out := make([]grupoAtivoPayload, 0, len(records))
	for i := range records {
		item := grupoAtivoToPayload(&records[i])
		rec := records[i]
		item.ParcelasCalculadas = calculateParcelasFromGrupoAtivo(&rec, holidays, now)
		if len(parcFilters) > 0 && !matchesParcFilter(item.ParcelasCalculadas, parcFilters) {
			continue
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": out, "count": len(out)})
}

func splitGAFiltersForSQL(filters string) (string, []int64) {
	raw := strings.TrimSpace(filters)
	if raw == "" {
		return "", nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ';' || r == ',' })
	keep := make([]string, 0, len(parts))
	parc := make([]int64, 0, 4)
	for _, p := range parts {
		token := strings.TrimSpace(p)
		if token == "" {
			continue
		}
		pair := strings.SplitN(token, ":", 2)
		if len(pair) != 2 {
			keep = append(keep, token)
			continue
		}
		key := strings.ToLower(strings.TrimSpace(pair[0]))
		val := strings.TrimSpace(pair[1])
		if key == "parc" {
			n, err := strconv.ParseInt(val, 10, 64)
			if err == nil && n >= 0 {
				parc = append(parc, n)
			}
			continue
		}
		keep = append(keep, token)
	}
	return strings.Join(keep, ";"), parc
}

func matchesParcFilter(parcelas int64, accepted []int64) bool {
	if len(accepted) == 0 {
		return true
	}
	for _, n := range accepted {
		if parcelas == n {
			return true
		}
	}
	return false
}

func (a *app) handleGrupoAtivoGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rec, err := store.GetGrupoAtivoByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeErr(w, http.StatusNotFound, fmt.Errorf("registro nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	item := grupoAtivoToPayload(rec)
	holidays, _ := store.ListActiveNationalHolidayDates(r.Context())
	item.ParcelasCalculadas = calculateParcelasFromGrupoAtivo(rec, holidays, time.Now().In(utcMinus3Loc))
	writeJSON(w, http.StatusOK, item)
}

func (a *app) handleGrupoAtivoParcelas(w http.ResponseWriter, r *http.Request) {
	group_code, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("group_code")), 10, 64)
	if err != nil || group_code <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("group_code invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	rec, err := store.GetGrupoAtivoByGrupo(r.Context(), group_code)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "group_code": group_code, "parcelas": 0, "found": false})
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	holidays, err := store.ListActiveNationalHolidayDates(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	parcelas := calculateParcelasFromGrupoAtivo(rec, holidays, time.Now().In(utcMinus3Loc))
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"found":      true,
		"group_code": group_code,
		"parcelas":   parcelas,
	})
}

func (a *app) handleGrupoAtivoSave(w http.ResponseWriter, r *http.Request) {
	var p grupoAtivoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	id, err := store.SaveGrupoAtivo(r.Context(), db.GrupoAtivoRecord{
		ID:                      p.ID,
		Grupo:                   p.Grupo,
		Vencimento:              p.Vencimento,
		QtdParticipantes:        p.QtdParticipantes,
		DataAssembleiaInaugural: sql.NullString{String: strings.TrimSpace(p.DataAssembleiaInaugural), Valid: strings.TrimSpace(p.DataAssembleiaInaugural) != ""},
		Plano:                   strings.TrimSpace(p.Plano),
		Prazo:                   p.Prazo,
		TipoGrupo:               strings.TrimSpace(p.TipoGrupo),
		Modelos:                 strings.TrimSpace(p.Modelos),
		Status:                  strings.TrimSpace(p.Status),
		CreatedAt:               sql.NullString{String: strings.TrimSpace(p.CreatedAt), Valid: strings.TrimSpace(p.CreatedAt) != ""},
		UpdatedAt:               sql.NullString{String: strings.TrimSpace(p.UpdatedAt), Valid: strings.TrimSpace(p.UpdatedAt) != ""},
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	msg := "Grupo is_active atualizado"
	if p.ID <= 0 {
		msg = "Grupo is_active criado"
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "message": msg})
}

func (a *app) handleGrupoAtivoDelete(w http.ResponseWriter, r *http.Request) {
	var p grupoAtivoPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if p.ID <= 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("id invalido"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if err := store.DeleteGrupoAtivo(r.Context(), p.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Grupo is_active removido"})
}

func (a *app) handleGruposAtivosDeleteBatch(w http.ResponseWriter, r *http.Request) {
	var p grupoAtivoIDsPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(p.IDs) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nenhum registro selecionado"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	count, err := store.DeleteGruposAtivosByIDs(r.Context(), p.IDs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"count":   count,
		"message": fmt.Sprintf("Registros removidos: %d", count),
	})
}

func (a *app) handleDBTablesCreate(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	if err := store.EnsureLegacySchema(r.Context()); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	// Auditoria
	a.logAudit(r, "DB_CREATE_TABLES", "database", strings.TrimSpace(cfg.DatabaseURL), "", "")

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": fmt.Sprintf("Estrutura validada: %d tabela(s) sincronizada(s)", len(db.LegacyTableNames())),
		"db_path": strings.TrimSpace(cfg.DatabaseURL),
	})
}

func (a *app) handleDBTablesClear(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	count, err := store.ClearLegacyData(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	// Auditoria
	a.logAudit(r, "DB_CLEAR_DATA", "database", strings.TrimSpace(cfg.DatabaseURL), "", "")

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": fmt.Sprintf("Dados removidos de %d tabela(s)", count),
		"db_path": strings.TrimSpace(cfg.DatabaseURL),
		"count":   count,
	})
}

func (a *app) handleDBTablesDrop(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	count, err := store.DropLegacyTables(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	// Auditoria
	a.logAudit(r, "DB_DROP_TABLES", "database", strings.TrimSpace(cfg.DatabaseURL), "", "")

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": fmt.Sprintf("Tabela(s) removida(s): %d", count),
		"db_path": strings.TrimSpace(cfg.DatabaseURL),
		"count":   count,
	})
}

func (a *app) handleDBBackup(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	if store.IsPostgres() {
		fileName := fmt.Sprintf("honda_go_postgres_backup_%s.sql", time.Now().Format("20060102_150405"))
		var buf strings.Builder
		if err := writePostgresDumpSQL(r.Context(), cfg.DatabaseURL, &buf); err != nil {
			// Fallback hospedado: gera dump SQL interno sem depender de pg_dump/docker local.
			if strings.Contains(strings.ToLower(err.Error()), "pg_dump") || strings.Contains(strings.ToLower(err.Error()), "docker") {
				buf.Reset()
				if berr := store.BackupSQL(r.Context(), &buf); berr != nil {
					writeErr(w, http.StatusInternalServerError, fmt.Errorf("falha ao gerar backup postgres: %w", berr))
					return
				}
			} else {
				writeErr(w, http.StatusInternalServerError, fmt.Errorf("falha ao gerar backup postgres: %w", err))
				return
			}
		}
		w.Header().Set("Content-Type", "application/sql")
		w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
		_, _ = io.WriteString(w, buf.String())
		a.logAudit(r, "DB_BACKUP", "database", "", "", fileName)
	} else {
		fileName := fmt.Sprintf("honda_go_backup_%s.sql", time.Now().Format("20060102_150405"))
		w.Header().Set("Content-Type", "application/sql")
		w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
		if err := store.BackupSQL(r.Context(), w); err != nil {
			log.Printf("[ERROR] Backup failed: %v", err)
		}
		a.logAudit(r, "DB_BACKUP", "database", "", "", fileName)
	}
}

func writePostgresDumpSQL(ctx context.Context, databaseURL string, w io.Writer) error {
	dsn := strings.TrimSpace(databaseURL)
	if dsn == "" {
		return fmt.Errorf("HONDAGO_DATABASE_URL nao configurado")
	}
	u, err := neturl.Parse(dsn)
	if err != nil {
		return fmt.Errorf("database_url invalida: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("database_url invalida para postgres")
	}

	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return fmt.Errorf("database_url sem nome do banco")
	}

	user := ""
	pass := ""
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}

	args := []string{
		"--host", host,
		"--port", port,
		"--username", user,
		"--dbname", dbName,
		"--format=plain",
		"--encoding=UTF8",
		"--inserts",
		"--clean",
		"--if-exists",
		"--no-owner",
		"--no-privileges",
	}
	var raw strings.Builder
	if err := runPgDump(ctx, "pg_dump", nil, pass, args, &raw); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "pg_dump nao encontrado") {
			// Fallback para ambiente Docker local (container padrao do projeto).
			container := strings.TrimSpace(os.Getenv("HONDAGO_POSTGRES_CONTAINER"))
			if container == "" {
				container = "hondago-postgres"
			}
			dockerArgs := []string{"exec", "-i"}
			if pass != "" {
				dockerArgs = append(dockerArgs, "-e", "PGPASSWORD="+pass)
			}
			dockerArgs = append(dockerArgs, container, "pg_dump")
			dockerArgs = append(dockerArgs, args...)
			raw.Reset()
			if derr := runPgDump(ctx, "docker", dockerArgs, "", nil, &raw); derr == nil {
				sanitized := sanitizePgDumpForGenericImport(raw.String())
				_, _ = io.WriteString(w, sanitized)
				return nil
			} else {
				return fmt.Errorf("pg_dump local indisponivel e fallback docker falhou (%s): %w", container, derr)
			}
		}
		return err
	}
	sanitized := sanitizePgDumpForGenericImport(raw.String())
	_, _ = io.WriteString(w, sanitized)
	return nil
}

func sanitizePgDumpForGenericImport(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	// Remove meta-comandos exclusivos do psql (ex.: \restrict/\unrestrict),
	// mantendo o dump compatível com importadores SQL genéricos (Adminer).
	var out strings.Builder
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, `\restrict `) || strings.HasPrefix(trimmed, `\unrestrict `) {
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

func runPgDump(ctx context.Context, bin string, fullArgs []string, pass string, pgArgs []string, w io.Writer) error {
	args := fullArgs
	if args == nil {
		args = pgArgs
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = os.Environ()
	if pass != "" {
		cmd.Env = append(cmd.Env, "PGPASSWORD="+pass)
	}
	cmd.Stdout = w
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if errors.Is(err, exec.ErrNotFound) {
			if bin == "pg_dump" {
				return fmt.Errorf("pg_dump nao encontrado no sistema")
			}
			return fmt.Errorf("%s nao encontrado no sistema", bin)
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (a *app) handleDBRestore(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("arquivo nao enviado: %w", err))
		return
	}
	defer file.Close()

	cfg, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	if store.IsPostgres() {
		payload, readErr := io.ReadAll(file)
		if readErr != nil {
			writeErr(w, http.StatusBadRequest, fmt.Errorf("erro ao ler arquivo de restore: %w", readErr))
			return
		}
		if err := restorePostgresSQL(r.Context(), cfg.DatabaseURL, bytes.NewReader(payload)); err != nil {
			// Fallback hospedado: restore SQL interno sem psql/docker local.
			if strings.Contains(strings.ToLower(err.Error()), "psql") || strings.Contains(strings.ToLower(err.Error()), "docker") {
				if rerr := store.RestoreSQL(r.Context(), bytes.NewReader(payload)); rerr != nil {
					writeErr(w, http.StatusInternalServerError, fmt.Errorf("erro ao restaurar postgres: %w", rerr))
					return
				}
			} else {
				writeErr(w, http.StatusInternalServerError, fmt.Errorf("erro ao restaurar postgres: %w", err))
				return
			}
		}
	} else {
		if err := store.RestoreSQL(r.Context(), file); err != nil {
			writeErr(w, http.StatusInternalServerError, fmt.Errorf("erro ao restaurar: %w", err))
			return
		}
	}

	a.logAudit(r, "DB_RESTORE", "database", "", "", "restore_performed")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Banco de dados restaurado com sucesso"})
}

func (a *app) handleDBRestoreCapabilities(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	caps := getDBCapabilities(store)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                true,
		"backup_available":  caps.BackupAvailable,
		"restore_available": caps.RestoreAvailable,
		"backup_reason":     caps.BackupReason,
		"restore_reason":    caps.RestoreReason,
		"reason":            caps.RestoreReason,
	})
}

func getDBCapabilities(store *db.Store) dbCapabilities {
	if !store.IsPostgres() {
		return dbCapabilities{
			BackupAvailable:  true,
			RestoreAvailable: true,
		}
	}
	return dbCapabilities{
		BackupAvailable:  true,
		RestoreAvailable: true,
		BackupReason:     "",
		RestoreReason:    "",
	}
}

func restorePostgresSQL(ctx context.Context, databaseURL string, r io.Reader) error {
	dsn := strings.TrimSpace(databaseURL)
	if dsn == "" {
		return fmt.Errorf("HONDAGO_DATABASE_URL nao configurado")
	}
	u, err := neturl.Parse(dsn)
	if err != nil {
		return fmt.Errorf("database_url invalida: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("database_url invalida para postgres")
	}

	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return fmt.Errorf("database_url sem nome do banco")
	}

	user := ""
	pass := ""
	if u.User != nil {
		user = u.User.Username()
		pass, _ = u.User.Password()
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}

	args := []string{
		"--host", host,
		"--port", port,
		"--username", user,
		"--dbname", dbName,
		"--single-transaction",
		"--set", "ON_ERROR_STOP=1",
	}
	payload, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("erro ao ler arquivo de restore: %w", err)
	}
	if err := runPSQLRestore(ctx, "psql", nil, pass, args, payload); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "psql nao encontrado") {
			container := strings.TrimSpace(os.Getenv("HONDAGO_POSTGRES_CONTAINER"))
			if container == "" {
				container = "hondago-postgres"
			}
			dockerArgs := []string{"exec", "-i"}
			if pass != "" {
				dockerArgs = append(dockerArgs, "-e", "PGPASSWORD="+pass)
			}
			dockerArgs = append(dockerArgs, container, "psql")
			dockerArgs = append(dockerArgs, args...)
			if derr := runPSQLRestore(ctx, "docker", dockerArgs, "", nil, payload); derr == nil {
				return nil
			} else {
				return fmt.Errorf("psql local indisponivel e fallback docker falhou (%s): %w", container, derr)
			}
		}
		return err
	}
	return nil
}

func runPSQLRestore(ctx context.Context, bin string, fullArgs []string, pass string, psqlArgs []string, payload []byte) error {
	args := fullArgs
	if args == nil {
		args = psqlArgs
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = os.Environ()
	if pass != "" {
		cmd.Env = append(cmd.Env, "PGPASSWORD="+pass)
	}
	cmd.Stdin = strings.NewReader(string(payload))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		if errors.Is(err, exec.ErrNotFound) {
			if bin == "psql" {
				return fmt.Errorf("psql nao encontrado no sistema")
			}
			return fmt.Errorf("%s nao encontrado no sistema", bin)
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (a *app) handleStop(w http.ResponseWriter, _ *http.Request) {
	a.mu.Lock()
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
		writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Parada solicitada"})
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: false, Message: "Nenhuma execuÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â§ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â£o em andamento"})
}

func (a *app) openStoreFromCurrentConfig() (*config.Config, *db.Store, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	cfgPath := strings.TrimSpace(a.configPath)
	if cfgPath == "" {
		cfgPath = "config.ini"
	}

	cfg, err := config.LoadFromINI(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("load config: %w", err)
	}
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, nil, fmt.Errorf("HONDAGO_DATABASE_URL nao configurado; este build opera somente com Postgres")
	}

	// Reutiliza apenas se a conexao-alvo nao mudou.
	if a.store != nil && a.cfg != nil && strings.TrimSpace(a.cfg.DatabaseURL) == strings.TrimSpace(cfg.DatabaseURL) {
		a.cfg = cfg
		return a.cfg, a.store, nil
	}

	if a.store != nil {
		_ = a.store.Close()
	}

	store, err := db.OpenWithURL(cfg.DatabasePath, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("open db: %w", err)
	}

	a.cfg = cfg
	a.store = store
	return a.cfg, a.store, nil
}

func (a *app) closeStore() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store != nil {
		_ = a.store.Close()
		a.store = nil
	}
}

func (a *app) databasePathFromPayload(r *http.Request) (string, error) {
	var p uiPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return "", err
	}

	cfgPath := strings.TrimSpace(p.ConfigPath)
	if cfgPath == "" {
		a.mu.Lock()
		cfgPath = strings.TrimSpace(a.configPath)
		a.mu.Unlock()
	}
	if cfgPath == "" {
		cfgPath = "config.ini"
	}
	cfgDir := filepath.Dir(cfgPath)

	if dbPath := strings.TrimSpace(p.DatabasePath); dbPath != "" {
		if filepath.IsAbs(dbPath) {
			return filepath.Clean(dbPath), nil
		}
		return filepath.Clean(filepath.Join(cfgDir, dbPath)), nil
	}

	cfg, err := config.LoadFromINI(cfgPath)
	if err == nil && strings.TrimSpace(cfg.DatabasePath) != "" {
		return filepath.Clean(cfg.DatabasePath), nil
	}
	return filepath.Clean(filepath.Join(cfgDir, "honda.db")), nil
}

func authRecordToPayload(r *db.AuthRecord) authUserPayload {
	out := authUserPayload{
		ID:                r.ID,
		CPF:               r.CPF,
		CodEmpresa:        r.CodEmpresa,
		CodUsuario:        r.CodUsuario,
		CodConcessionaria: r.CodConcessionaria,
		Senha:             maskSecret(r.Senha),
		InFlight:          nullInt64(r.InFlight),
		Error401Count:     nullInt64(r.Error401Count),
		Error429Count:     nullInt64(r.Error429Count),
	}
	if r.Token.Valid {
		out.Token = maskSecret(r.Token.String)
	}
	if r.TokenB3.Valid {
		out.TokenB3 = maskSecret(r.TokenB3.String)
	}
	out.LastRequest = nullTimeStr(r.LastRequest)
	out.CooldownUntil = nullTimeStr(r.CooldownUntil)
	out.BlockedUntil = nullTimeStr(r.BlockedUntil)
	if r.PriorityScore.Valid {
		out.PriorityScore = strconv.FormatFloat(r.PriorityScore.Float64, 'f', 2, 64)
	}
	return out
}

func maskSecret(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return strings.Repeat("*", len(v))
	}
	return v[:2] + strings.Repeat("*", len(v)-4) + v[len(v)-2:]
}

func nullInt64(v sql.NullInt64) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}

func nullFloat64(v sql.NullFloat64) float64 {
	if !v.Valid {
		return 0
	}
	return v.Float64
}

func nullTimeStr(v sql.NullTime) string {
	if !v.Valid {
		return ""
	}
	return v.Time.Format("2006-01-02 15:04:05")
}

func (a *app) handleLogs(w http.ResponseWriter, r *http.Request) {
	if !a.requirePermission(w, r, "logs:read") {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"logs": a.logBuffer.String()})
}

func (a *app) uiLogf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s %s\n", time.Now().In(utcMinus3Loc).Format("2006/01/02 15:04:05"), msg)
	_, _ = a.logBuffer.Write([]byte(line))
	a.appendToUILogFile(line)
	log.Print(msg)
}

func (a *app) uiLogfTo(filePath, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s %s\n", time.Now().In(utcMinus3Loc).Format("2006/01/02 15:04:05"), msg)
	_, _ = a.logBuffer.Write([]byte(line))
	a.appendToLogFile(filePath, line)
	log.Print(msg)
}

func (a *app) newUILogFilePath(tag string) string {
	logDir := a.logDirPath()
	safeTag := strings.TrimSpace(strings.ToLower(tag))
	if safeTag == "" {
		safeTag = "ui_actions"
	}
	safeTag = strings.ReplaceAll(safeTag, " ", "_")
	fileName := fmt.Sprintf("honda_go_%s_%s.log", safeTag, time.Now().In(utcMinus3Loc).Format("20060102_150405_000"))
	return filepath.Join(logDir, fileName)
}

func (a *app) appendToUILogFile(line string) {
	filePath := a.newUILogFilePath("ui_actions")
	a.appendToLogFile(filePath, line)
}

func (a *app) appendToLogFile(filePath, line string) {
	if strings.TrimSpace(filePath) == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

func (a *app) handleClearLogs(w http.ResponseWriter, r *http.Request) {
	if !a.requirePermission(w, r, "logs:delete") {
		return
	}
	a.logBuffer.Clear()
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Log limpo"})
}

func (a *app) handleStatus(w http.ResponseWriter, _ *http.Request) {
	a.mu.Lock()
	running := a.running
	startedAt := a.startedAt
	finishedAt := a.finishedAt
	a.mu.Unlock()

	startedAtStr := ""
	finishedAtStr := ""
	if !startedAt.IsZero() {
		startedAtStr = startedAt.Format(time.RFC3339)
	}
	if !finishedAt.IsZero() {
		finishedAtStr = finishedAt.Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"running":     running,
		"started_at":  startedAtStr,
		"finished_at": finishedAtStr,
	})
}

func (a *app) handleDiagnosticLogFiles(w http.ResponseWriter, r *http.Request) {
	if !a.requirePermission(w, r, "logs:read") {
		return
	}
	logDir := a.logDirPath()
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]any{"log_dir": logDir, "files": []diagnosticLogFile{}})
			return
		}
		writeErr(w, http.StatusInternalServerError, fmt.Errorf("listar logs: %w", err))
		return
	}

	files := make([]diagnosticLogFile, 0, len(entries))
	type sortable struct {
		item diagnosticLogFile
		t    time.Time
	}
	tmp := make([]sortable, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		tmp = append(tmp, sortable{
			item: diagnosticLogFile{
				Name:       name,
				ModifiedAt: info.ModTime().In(utcMinus3Loc).Format("2006-01-02 15:04:05"),
				SizeBytes:  info.Size(),
			},
			t: info.ModTime().In(utcMinus3Loc),
		})
	}

	sort.Slice(tmp, func(i, j int) bool { return tmp[i].t.After(tmp[j].t) })
	for _, item := range tmp {
		files = append(files, item.item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"log_dir": logDir, "files": files})
}

func (a *app) handleDiagnosticLogFile(w http.ResponseWriter, r *http.Request) {
	if !a.requirePermission(w, r, "logs:read") {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	_, absTarget, safeName, err := a.resolveDiagnosticLogTarget(name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	content, err := os.ReadFile(absTarget)
	if err != nil {
		if os.IsNotExist(err) {
			writeErr(w, http.StatusNotFound, fmt.Errorf("arquivo nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\""+safeName+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (a *app) handleDiagnosticLogDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requirePermission(w, r, "logs:delete") {
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	_, absTarget, _, err := a.resolveDiagnosticLogTarget(name)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := os.Remove(absTarget); err != nil {
		if os.IsNotExist(err) {
			writeErr(w, http.StatusNotFound, fmt.Errorf("arquivo nao encontrado"))
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, runResponse{OK: true, Message: "Arquivo removido"})
}

func (a *app) handleDiagnosticLogDeleteBatch(w http.ResponseWriter, r *http.Request) {
	if !a.requirePermission(w, r, "logs:delete") {
		return
	}
	var p diagnosticLogNamesPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if len(p.Names) == 0 {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("nenhum arquivo selecionado"))
		return
	}

	removed := 0
	for _, name := range p.Names {
		_, absTarget, _, err := a.resolveDiagnosticLogTarget(strings.TrimSpace(name))
		if err != nil {
			continue
		}
		if err := os.Remove(absTarget); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			continue
		}
		removed++
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      removed > 0,
		"count":   removed,
		"message": fmt.Sprintf("Arquivos removidos: %d", removed),
	})
}

func (a *app) resolveDiagnosticLogTarget(name string) (string, string, string, error) {
	if name == "" {
		return "", "", "", fmt.Errorf("nome do arquivo obrigatÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â³rio")
	}

	safeName := filepath.Base(name)
	if safeName != name || safeName == "." || safeName == ".." {
		return "", "", "", fmt.Errorf("nome de arquivo invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido")
	}
	if !strings.HasSuffix(strings.ToLower(safeName), ".log") {
		return "", "", "", fmt.Errorf("arquivo deve ter extensao .log")
	}

	logDir := a.logDirPath()
	target := filepath.Join(logDir, safeName)

	absDir, err := filepath.Abs(logDir)
	if err != nil {
		return "", "", "", err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", "", err
	}
	prefix := absDir + string(os.PathSeparator)
	if absTarget != absDir && !strings.HasPrefix(absTarget, prefix) {
		return "", "", "", fmt.Errorf("acesso negado ao arquivo")
	}
	return absDir, absTarget, safeName, nil
}

func (a *app) logDirPath() string {
	a.mu.Lock()
	cfgPath := strings.TrimSpace(a.configPath)
	a.mu.Unlock()

	if cfgPath == "" {
		cfgPath = "config.ini"
	}
	absCfg, err := filepath.Abs(cfgPath)
	if err == nil {
		return filepath.Join(filepath.Dir(absCfg), "log")
	}

	wd, err := os.Getwd()
	if err == nil {
		return filepath.Join(wd, "log")
	}
	return "log"
}

func (a *app) runEngine(ctx context.Context, configPath string) {
	defer func() {
		a.mu.Lock()
		a.running = false
		a.cancel = nil
		a.finishedAt = time.Now()
		a.mu.Unlock()
	}()

	logFile, logPath, err := applog.ConfigureFileLogging(configPath)
	if err != nil {
		a.logBuffer.Write([]byte(fmt.Sprintf("[gui] erro log: %v\n", err)))
		return
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(os.Stdout, logFile, a.logBuffer))
	log.SetFlags(log.Ldate | log.Ltime)
	log.Printf("[go] Log file: %s", logPath)

	start := time.Now()
	cfg, err := config.LoadFromINI(configPath)
	if err != nil {
		log.Printf("[go] erro load config: %v", err)
		return
	}

	store, err := db.OpenWithURL(cfg.DatabasePath, cfg.DatabaseURL)
	if err != nil {
		log.Printf("[go] erro ao abrir db: %v", err)
		return
	}

	log.Printf("[go] starting engine mode=%s db=%s", cfg.Booking.ReservarModo, cfg.DatabasePath)
	e := engine.New(cfg, store)
	if err := e.Run(ctx); err != nil {
		log.Printf("[go] run engine: %v", err)
		return
	}
	log.Printf("[go] finished in %s", time.Since(start))
}

func (a *app) runAuth(ctx context.Context, configPath string) {
	defer func() {
		a.mu.Lock()
		a.running = false
		a.cancel = nil
		a.finishedAt = time.Now()
		a.mu.Unlock()
	}()

	logFile, logPath, err := applog.ConfigureFileLogging(configPath)
	if err != nil {
		a.logBuffer.Write([]byte(fmt.Sprintf("[gui] erro log: %v\n", err)))
		return
	}
	defer logFile.Close()

	log.SetOutput(io.MultiWriter(os.Stdout, logFile, a.logBuffer))
	log.SetFlags(log.Ldate | log.Ltime)
	log.Printf("[go] Log file: %s", logPath)

	start := time.Now()
	cfg, err := config.LoadFromINI(configPath)
	if err != nil {
		log.Printf("[go] erro load config: %v", err)
		return
	}

	store, err := db.OpenWithURL(cfg.DatabasePath, cfg.DatabaseURL)
	if err != nil {
		log.Printf("[go] erro ao abrir db: %v", err)
		return
	}

	log.Printf("[go] starting auth update db=%s", cfg.DatabasePath)
	e := engine.New(cfg, store)
	if err := e.UpdateTokens(ctx); err != nil {
		log.Printf("[go] update tokens: %v", err)
		return
	}
	log.Printf("[go] auth update finished in %s", time.Since(start))
}

func (a *app) payloadToConfig(configPath string, p *uiPayload) (*config.Config, error) {
	loaded, err := config.LoadFromINI(configPath)
	if err != nil {
		configDir := filepath.Dir(configPath)
		loaded = &config.Config{
			DatabasePath: filepath.Join(configDir, "honda.db"),
			APIBaseURL:   defaultAPIBaseURL,
		}
	}
	if strings.TrimSpace(loaded.APIBaseURL) == "" {
		loaded.APIBaseURL = defaultAPIBaseURL
	}

	product_name, err := parseIntList(p.Produto)
	if err != nil {
		return nil, fmt.Errorf("Produto: %w", err)
	}
	group_code, err := parseIntList(p.Grupo)
	if err != nil {
		return nil, fmt.Errorf("group_code: %w", err)
	}
	due_day, err := parseIntList(p.Vencimento)
	if err != nil {
		return nil, fmt.Errorf("due_day: %w", err)
	}
	idCota, err := parseIntList(p.IDCota)
	if err != nil {
		return nil, fmt.Errorf("requested_quota_id: %w", err)
	}
	limit, err := parseIntOrDefault(p.Limit, 0)
	if err != nil {
		return nil, fmt.Errorf("limit: %w", err)
	}
	cooldown, err := parseIntOrDefault(p.CooldownUserMS, 200)
	if err != nil {
		return nil, fmt.Errorf("cooldown_user_ms: %w", err)
	}
	workers, err := parseIntOrDefault(p.WorkerCount, 24)
	if err != nil {
		return nil, fmt.Errorf("worker_count_go: %w", err)
	}
	timeout, err := parseIntOrDefault(p.RequestTimeoutMS, 7000)
	if err != nil {
		return nil, fmt.Errorf("request_timeout_ms: %w", err)
	}

	mode := strings.ToLower(strings.TrimSpace(p.ReservarModo))
	if mode != "b4" {
		mode = "b1"
	}

	loaded.Booking = config.BookingConfig{
		Modelo:              strings.TrimSpace(p.Modelo),
		Produto:             product_name,
		ReservarModo:        mode,
		Grupo:               group_code,
		CPF:                 parseStrList(p.CPF),
		CodEmpre:            parseStrList(p.CodEmpre),
		Vencimento:          due_day,
		IDCota:              idCota,
		LoteriaFederal:      strings.TrimSpace(p.LoteriaFederal),
		AcrescimoDecrescimo: strings.TrimSpace(p.AcrescimoDecrescimo),
		TipoGrupo:           strings.TrimSpace(p.TipoGrupo),
		Limit:               limit,
		DryRun:              p.DryRun,
		CooldownUserMS:      cooldown,
		WorkerCount:         workers,
		RequestTimeoutMS:    timeout,
		TokenPrincipal:      loaded.Booking.TokenPrincipal,
	}

	if v := strings.TrimSpace(p.DatabasePath); v != "" {
		if filepath.IsAbs(v) {
			loaded.DatabasePath = v
		} else {
			loaded.DatabasePath = filepath.Join(filepath.Dir(configPath), v)
		}
	}
	if v := strings.TrimSpace(p.APIBaseURL); v != "" {
		loaded.APIBaseURL = v
	}

	return loaded, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Prevenir cache em navegadores para APIs
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, runResponse{OK: false, Message: sanitizeClientErrorMessage(status, err)})
}

var (
	reIPv4Port       = regexp.MustCompile(`\b\d{1,3}(?:\.\d{1,3}){3}:\d+\b`)
	reLocalhostPort  = regexp.MustCompile(`(?i)\blocalhost:\d+\b`)
	reSQLStateSuffix = regexp.MustCompile(`\s*\(SQLSTATE [0-9A-Z]+\)\s*$`)
	reMultiSpace     = regexp.MustCompile(`\s{2,}`)
)

func sanitizeClientErrorMessage(status int, err error) string {
	if err == nil {
		if status >= http.StatusInternalServerError {
			return "Serviço temporariamente indisponível. Tente novamente em instantes."
		}
		return "Erro inesperado."
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		if status >= http.StatusInternalServerError {
			return "Serviço temporariamente indisponível. Tente novamente em instantes."
		}
		return "Erro inesperado."
	}

	// 5xx nunca deve expor detalhes internos (host/IP/porta/driver/SQLSTATE).
	if status >= http.StatusInternalServerError {
		return "Serviço temporariamente indisponível. Tente novamente em instantes."
	}

	// 4xx: mantém mensagem funcional, removendo detalhes técnicos sensíveis.
	s := msg
	s = reSQLStateSuffix.ReplaceAllString(s, "")
	s = reIPv4Port.ReplaceAllString(s, "[host:porta]")
	s = reLocalhostPort.ReplaceAllString(s, "[localhost:porta]")
	lower := strings.ToLower(s)
	if strings.Contains(lower, "failed to connect") ||
		strings.Contains(lower, "dial tcp") ||
		strings.Contains(lower, "connection refused") {
		return "Serviço temporariamente indisponível. Tente novamente em instantes."
	}
	s = reMultiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func parseStrList(raw string) []string {
	s := normalizeListRaw(raw)
	if s == "" {
		return nil
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(strings.Trim(part, "\"'"))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseIntList(raw string) ([]int, error) {
	items := parseStrList(raw)
	if len(items) == 0 {
		return nil, nil
	}
	out := make([]int, 0, len(items))
	for _, item := range items {
		v, err := strconv.Atoi(item)
		if err != nil {
			return nil, fmt.Errorf("valor invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido %q", item)
		}
		out = append(out, v)
	}
	return out, nil
}

func parseIntOrDefault(raw string, fallback int) (int, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("valor invÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã¢â‚¬Â ÃƒÂ¢Ã¢â€šÂ¬Ã¢â€žÂ¢ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬ÃƒÂ¢Ã¢â‚¬Å¾Ã‚Â¢ÃƒÆ’Ã†â€™Ãƒâ€ Ã¢â‚¬â„¢ÃƒÆ’Ã‚Â¢ÃƒÂ¢Ã¢â‚¬Å¡Ã‚Â¬Ãƒâ€¦Ã‚Â¡ÃƒÆ’Ã†â€™ÃƒÂ¢Ã¢â€šÂ¬Ã…Â¡ÃƒÆ’Ã¢â‚¬Å¡Ãƒâ€šÃ‚Â¡lido %q", s)
	}
	return v, nil
}

func normalizeListRaw(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") && len(s) >= 2 {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

func joinInt(items []int) string {
	if len(items) == 0 {
		return ""
	}
	out := make([]string, 0, len(items))
	for _, v := range items {
		out = append(out, strconv.Itoa(v))
	}
	return strings.Join(out, ",")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}

func (a *app) handleRBACMatrixGet(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	roles, err := store.GetAllRolesAndPermissions(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"roles": roles,
	})
}

func (a *app) handleRBACMatrixUpdate(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	var payload struct {
		Roles []db.RoleMatrix `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	normalizePerms := func(in []string) []string {
		set := make(map[string]struct{}, len(in))
		for _, p := range in {
			v := strings.TrimSpace(p)
			if v == "" {
				continue
			}
			set[v] = struct{}{}
		}
		out := make([]string, 0, len(set))
		for p := range set {
			out = append(out, p)
		}
		sort.Strings(out)
		return out
	}
	equalPermSets := func(aSet, bSet []string) bool {
		if len(aSet) != len(bSet) {
			return false
		}
		for i := range aSet {
			if aSet[i] != bSet[i] {
				return false
			}
		}
		return true
	}

	for _, role := range payload.Roles {
		currentPerms, err := store.GetRolePermissions(r.Context(), role.RoleName)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		before := normalizePerms(currentPerms)
		after := normalizePerms(role.Permissions)
		if equalPermSets(before, after) {
			continue
		}

		if err := store.UpdateRolePermissions(r.Context(), role.RoleName, after); err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			return
		}

		// Auditoria
		a.logAudit(r, "UPDATE_RBAC", "roles", role.RoleName, strings.Join(before, ","), strings.Join(after, ","))

		// Invalida sessÃƒÂµes no banco para este perfil para forÃƒÂ§ar recarregamento das permissÃƒÂµes
		if err := store.DeleteAppSessionsByRole(r.Context(), role.RoleName); err != nil {
			log.Printf("[ERROR] Falha ao invalidar sessÃƒÂµes para perfil %s: %v", role.RoleName, err)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Matriz de permissÃƒÂµes atualizada com sucesso",
	})
}

func (a *app) handleAuditList(w http.ResponseWriter, r *http.Request) {
	if !a.requirePermission(w, r, "audit:view") {
		return
	}
	q := r.URL.Query().Get("q")
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	logs, err := store.SearchAuditLogs(r.Context(), q, 500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"logs": logs,
	})
}

func (a *app) handleMFASetup(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.currentSession(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("nÃƒÂ£o autenticado"))
		return
	}
	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	user, err := store.GetAppUserByID(r.Context(), sess.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "HondaGo",
		AccountName: user.Username,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"secret": key.Secret(),
		"url":    key.URL(),
	})
}

func (a *app) handleMFAVerify(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.currentSession(r)
	if !ok {
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("nÃƒÂ£o autenticado"))
		return
	}

	type payload struct {
		Secret string `json:"secret"`
		Code   string `json:"code"`
	}
	var p payload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	p.Secret = strings.TrimSpace(p.Secret)
	p.Code = strings.TrimSpace(p.Code)

	if p.Secret == "" || p.Code == "" {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("segredo e cÃƒÂ³digo sÃƒÂ£o obrigatÃƒÂ³rios"))
		return
	}

	valid := totp.Validate(p.Code, p.Secret)
	if !valid {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("cÃƒÂ³digo invÃƒÂ¡lido"))
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	user, err := store.GetAppUserByID(r.Context(), sess.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	user.MFAEnabled = 1
	user.MFASecret = sql.NullString{String: p.Secret, Valid: true}
	if _, err := store.SaveAppUser(r.Context(), *user, ""); err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	a.logAudit(r, "MFA_ENABLED", "users", user.Username, "", "MFA ativado pelo usuÃƒÂ¡rio")

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "MFA ativado com sucesso",
	})
}

func (a *app) handleAppMFALogin(w http.ResponseWriter, r *http.Request) {
	type payload struct {
		Token string `json:"temp_token"`
		Code  string `json:"code"`
	}
	var p payload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	a.mu.Lock()
	temp, exists := a.mfaTempTokens[p.Token]
	a.mu.Unlock()

	if !exists || temp.ExpiresAt.Before(time.Now()) {
		reason := "nÃƒÂ£o encontrado"
		if exists {
			reason = "expirado"
		}
		log.Printf("[SECURITY] MFA: Tentativa de login com temp_token invÃƒÂ¡lido ou expirado. Token: %s, Motivo: %s", maskSecret(p.Token), reason)
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("sessÃƒÂ£o expirada ou invÃƒÂ¡lida (mfa_token_%s)", reason))
		return
	}

	_, store, err := a.openStoreFromCurrentConfig()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	user, err := store.GetAppUserByID(r.Context(), temp.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	if user.MFAEnabled == 0 || !user.MFASecret.Valid {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("MFA nÃƒÂ£o configurado para este usuÃƒÂ¡rio"))
		return
	}

	p.Code = strings.TrimSpace(p.Code)
	valid, _ := totp.ValidateCustom(p.Code, user.MFASecret.String, time.Now().UTC(), totp.ValidateOpts{
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if !valid {
		writeErr(w, http.StatusUnauthorized, fmt.Errorf("cÃƒÂ³digo MFA invÃƒÂ¡lido"))
		return
	}

	// Sucesso! Deletar token temporÃƒÂ¡rio
	a.mu.Lock()
	delete(a.mfaTempTokens, p.Token)
	a.mu.Unlock()

	// Login final
	token, _ := newSessionToken()
	perms, _ := store.GetRolePermissions(r.Context(), user.Role)
	if perms == nil {
		perms = []string{}
	}

	sessRecord := &db.AppSessionRecord{
		Token:           token,
		UserID:          user.ID,
		Username:        user.Username,
		DisplayName:     user.DisplayName,
		CPF:             user.CPF,
		Filial:          user.Filial,
		Role:            user.Role,
		Permissions:     perms,
		AuthenticatedAt: time.Now(),
	}
	_ = store.SaveAppSession(r.Context(), sessRecord)
	a.setSessionCookie(w, r, token)
	a.setCSRFCookie(w, r, readCSRFCookieValue(r))

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"full_name":   user.DisplayName,
		"username":    user.Username,
		"cpf":         user.CPF.String,
		"branch":      user.Filial.String,
		"role":        user.Role,
		"permissions": perms,
	})
}
