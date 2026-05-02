package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"honda/go-engine/internal/config"
	"honda/go-engine/internal/db"
)

const (
	notFoundMessage       = "Cota Disponível Reposição não encontrada."
	unexpectedErrorMarker = "erro inesperado"
	modelNotFoundMarker   = "modelo de reposição não encontrado"
)

var nonFailed422Codes = []string{"40015.02", "728.02"}

type bookingPayload struct {
	CodEmpresa      string `json:"codEmpresa"`
	CodUsuario      string `json:"codUsuario"`
	CPFCNPJ         string `json:"cpfCnpj"`
	IDCotaReposicao string `json:"idCotaReposicao"`
	IDModelo        string `json:"idModelo"`
	IDProduto       string `json:"idProduto"`
}

type reservedCotaAPIItem struct {
	UsuarioReserva     string `json:"usuarioReserva"`
	NumDocumentoPessoa string `json:"numDocumentoPessoa"`
	CodGrupo           string `json:"codGrupo"`
	CotaRD             string `json:"cotaRD"`
	CodModelo          string `json:"codModelo"`
}

type logInfoPayload struct {
	CPFVendedor string `json:"cpfVendedor"`
	CodUsuario  string `json:"codUsuario"`
	CodEmpresa  string `json:"codEmpresa"`
	Plataforma  string `json:"plataforma"`
	Token       string `json:"token"`
	Evento      string `json:"evento"`
	Abertura    string `json:"abertura"`
	Fechamento  string `json:"fechamento"`
}

type ReserveInput struct {
	IDCotaReposicao string
	IDModelo        string
	IDProduto       string
}

type ReserveResult struct {
	StatusCode      int
	Body            string
	CotaRD          string
	CodGrupo        string
	IDCotaReposicao string
}

type Metrics struct {
	Total      int64
	Completed  int64
	Success    int64
	HTTPStatus sync.Map // map[int]*int64
}

type Engine struct {
	cfg   *config.Config
	store *db.Store
	http  *http.Client
}

func New(cfg *config.Config, store *db.Store) *Engine {
	timeout := time.Duration(cfg.Booking.RequestTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 7 * time.Second
	}
	transport := &http.Transport{
		MaxIdleConns:        512,
		MaxIdleConnsPerHost: 256,
		MaxConnsPerHost:     512,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Engine{
		cfg:   cfg,
		store: store,
		http:  &http.Client{Timeout: timeout, Transport: transport},
	}
}

func (e *Engine) Run(ctx context.Context) error {
	if err := e.store.EnsureLegacySchema(ctx); err != nil {
		return err
	}

	users, err := e.store.LoadUsers(ctx, e.cfg.Booking.CPF, e.cfg.Booking.CodEmpre)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("nenhum usuário/token disponível")
	}

	log.Printf(
		"[go][filters] produto=%v vencimento=%v grupo=%v id_cota=%v tipo_grupo=%q loteria_federal=%q",
		e.cfg.Booking.Produto,
		e.cfg.Booking.Vencimento,
		e.cfg.Booking.Grupo,
		e.cfg.Booking.IDCota,
		e.cfg.Booking.TipoGrupo,
		e.cfg.Booking.LoteriaFederal,
	)

	cotas, err := e.store.LoadCotas(ctx, db.CotaFilter{
		Produto:    e.cfg.Booking.Produto,
		Vencimento: e.cfg.Booking.Vencimento,
		Grupo:      e.cfg.Booking.Grupo,
		IDCota:     e.cfg.Booking.IDCota,
		TipoGrupo:  e.cfg.Booking.TipoGrupo,
		Limit:      0, // limit será aplicado após loteria
	})
	if err != nil {
		return err
	}
	log.Printf("[go][filters] total_base=%d", len(cotas))
	if len(cotas) == 0 {
		log.Println("[go] nenhuma cota encontrada com filtros base")
		return nil
	}

	if len(e.cfg.Booking.IDCota) > 0 {
		log.Printf("[go][filters] id_cota informado; ignorando filtro de loteria para priorizar selecao explicita.")
	} else {
		cotas = applyLoteriaFilter(cotas, e.cfg.Booking.ReservarModo, e.cfg.Booking.LoteriaFederal, e.cfg.Booking.AcrescimoDecrescimo)
		if len(cotas) == 0 {
			log.Println("[go] nenhuma cota após filtro de loteria")
			return nil
		}
	}

	if e.cfg.Booking.Limit > 0 && len(cotas) > e.cfg.Booking.Limit {
		cotas = cotas[:e.cfg.Booking.Limit]
	}

	workerCount := e.cfg.Booking.WorkerCount
	if workerCount <= 0 {
		workerCount = min(24, len(users))
	}
	if workerCount > len(cotas) {
		workerCount = len(cotas)
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	pool := newUserPool(users, e.store, time.Duration(e.cfg.Booking.CooldownUserMS)*time.Millisecond)
	metrics := &Metrics{}
	metrics.Total = int64(len(cotas))

	log.Printf("[go] mode=%s total_cotas=%d users=%d workers=%d dry_run=%v",
		e.cfg.Booking.ReservarModo, len(cotas), len(users), workerCount, e.cfg.Booking.DryRun)

	if e.cfg.Booking.DryRun {
		e.runDryRun(cotas, users, metrics)
		log.Printf("[go][metrics] total=%d completed=%d success=%d", metrics.Total, metrics.Completed, metrics.Success)
		metrics.HTTPStatus.Range(func(key, value any) bool {
			code := key.(int)
			count := atomic.LoadInt64(value.(*int64))
			log.Printf("[go][metrics] status=%d count=%d", code, count)
			return true
		})
		return nil
	}

	jobs := make(chan db.Cota, len(cotas))
	wg := sync.WaitGroup{}
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for cota := range jobs {
				if err := e.processCota(ctx, pool, cota, metrics); err != nil {
					log.Printf("[go][worker=%d] cota=%d err=%v", workerID, cota.IDGrupo, err)
				}
			}
		}(i + 1)
	}
	for _, c := range cotas {
		jobs <- c
	}
	close(jobs)
	wg.Wait()

	log.Printf("[go][metrics] total=%d completed=%d success=%d", metrics.Total, metrics.Completed, metrics.Success)
	metrics.HTTPStatus.Range(func(key, value any) bool {
		code := key.(int)
		count := atomic.LoadInt64(value.(*int64))
		log.Printf("[go][metrics] status=%d count=%d", code, count)
		return true
	})

	return nil
}

func (e *Engine) runDryRun(cotas []db.Cota, users []db.User, m *Metrics) {
	if len(users) == 0 {
		return
	}
	idx := 0
	for _, cota := range cotas {
		user := users[idx%len(users)]
		idx++
		payload := bookingPayload{
			CodEmpresa:      user.CodEmpresa,
			CodUsuario:      user.CodUsuario,
			CPFCNPJ:         user.CPF,
			IDCotaReposicao: fmt.Sprintf("%d", cota.IDGrupo),
			IDModelo:        e.cfg.Booking.Modelo,
			IDProduto:       cota.Produto,
		}
		log.Printf("[go][dry-run] user=%s cota=%d payload=%+v", user.CodUsuario, cota.IDGrupo, payload)
		atomic.AddInt64(&m.Completed, 1)
		addStatus(m, 0)
	}
}

func (e *Engine) processCota(ctx context.Context, pool *userPool, cota db.Cota, m *Metrics) error {
	user, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer pool.Release(ctx, user)

	payload := bookingPayload{
		CodEmpresa:      user.CodEmpresa,
		CodUsuario:      user.CodUsuario,
		CPFCNPJ:         user.CPF,
		IDCotaReposicao: fmt.Sprintf("%d", cota.IDGrupo),
		IDModelo:        e.cfg.Booking.Modelo,
		IDProduto:       cota.Produto,
	}

	body, _ := json.Marshal(payload)
	log.Printf("[go] payload user=%s cota=%d body=%s", user.CodUsuario, cota.IDGrupo, string(body))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.cfg.APIBaseURL+"/cotasReposicao", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("versao", "0.1.0")
	req.Header.Set("Authorization", "Bearer "+user.Token)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := e.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody := readBody(resp)
	respTextLower := strings.ToLower(respBody)

	atomic.AddInt64(&m.Completed, 1)
	addStatus(m, resp.StatusCode)

	_ = e.store.MarkUserStatus(ctx, user.ID, resp.StatusCode)

	if resp.StatusCode == 200 {
		atomic.AddInt64(&m.Success, 1)
		if _, err := e.persistReservationDeterministic(ctx, user, cota); err != nil {
			log.Printf("[go] save deterministic user=%s cota=%d err=%v", user.CodUsuario, cota.IDGrupo, err)
		}
		return e.store.MarkCotaBooked(ctx, cota.ID)
	}

	log.Printf("[go] cota=%d user=%s status=%d body=%s", cota.IDGrupo, user.CodUsuario, resp.StatusCode, truncate(respBody, 400))

	if shouldSkipFailed422(resp.StatusCode, respBody) {
		return nil
	}
	if resp.StatusCode != 500 && strings.Contains(respBody, notFoundMessage) {
		return e.store.MarkCotaFailed(ctx, cota.ID)
	}
	if resp.StatusCode == 422 && strings.Contains(respTextLower, modelNotFoundMarker) {
		return e.store.MarkCotaFailed(ctx, cota.ID)
	}
	if resp.StatusCode == 500 && strings.Contains(respTextLower, unexpectedErrorMarker) {
		return e.store.MarkCotaFailed(ctx, cota.ID)
	}
	return nil
}

func (e *Engine) persistReservationDeterministic(ctx context.Context, user *db.User, cota db.Cota) (*db.ReservedCotaRecord, error) {
	modeloNome := strings.TrimSpace(e.cfg.Booking.Modelo)
	if idVal, err := strconv.ParseInt(modeloNome, 10, 64); err == nil {
		if nome, nerr := e.store.FindModeloNomeByIDModelo(ctx, idVal); nerr == nil && strings.TrimSpace(nome) != "" {
			modeloNome = strings.TrimSpace(nome)
		}
	}

	row := db.ReservedCota{
		UsuarioReserva:     user.CodUsuario,
		NumDocumentoPessoa: user.CPF,
		CodGrupo:           strconv.Itoa(cota.Grupo),
		CotaRD:             fmt.Sprintf("%d-%d-%d", cota.Cota, cota.R, cota.D),
		CodModelo:          modeloNome,
		IDCotaReposicao:    strconv.FormatInt(cota.IDGrupo, 10),
	}

	if _, err := e.store.SaveCotasReservadas(ctx, []db.ReservedCota{row}); err != nil {
		return nil, err
	}
	return e.store.GetLatestReservedCotaByIDCotaReposicao(ctx, row.IDCotaReposicao, user.CPF)
}

func (e *Engine) ReserveByUser(ctx context.Context, user *db.User, in ReserveInput) (*ReserveResult, error) {
	return e.reserveByUser(ctx, user, in, true)
}

func (e *Engine) ReserveByUserNoPersist(ctx context.Context, user *db.User, in ReserveInput) (*ReserveResult, error) {
	return e.reserveByUser(ctx, user, in, false)
}

func (e *Engine) reserveByUser(ctx context.Context, user *db.User, in ReserveInput, persistReserved bool) (*ReserveResult, error) {
	if user == nil {
		return nil, fmt.Errorf("usuario vazio")
	}
	payload := bookingPayload{
		CodEmpresa:      user.CodEmpresa,
		CodUsuario:      user.CodUsuario,
		CPFCNPJ:         user.CPF,
		IDCotaReposicao: strings.TrimSpace(in.IDCotaReposicao),
		IDModelo:        strings.TrimSpace(in.IDModelo),
		IDProduto:       strings.TrimSpace(in.IDProduto),
	}
	if payload.CodEmpresa == "" || payload.CodUsuario == "" || payload.CPFCNPJ == "" || payload.IDCotaReposicao == "" || payload.IDModelo == "" || payload.IDProduto == "" {
		return nil, fmt.Errorf("payload incompleto")
	}

	body, _ := json.Marshal(payload)
	log.Printf("[go] payload user=%s cota=%s body=%s", user.CodUsuario, payload.IDCotaReposicao, string(body))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.cfg.APIBaseURL+"/cotasReposicao", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("versao", "0.1.0")
	req.Header.Set("Authorization", "Bearer "+user.Token)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody := readBody(resp)

	_ = e.store.MarkUserStatus(ctx, user.ID, resp.StatusCode)

	reservedCotaRD := ""
	reservedCodGrupo := ""
	reservedIDCotaReposicao := ""
	if resp.StatusCode == 200 && persistReserved {
		saved, err := e.saveCotasReservadas(ctx, user, payload)
		if err != nil {
			log.Printf("[go] save cotasreservadas user=%s cota=%s err=%v", user.CodUsuario, payload.IDCotaReposicao, err)
		} else {
			reservedCotaRD = strings.TrimSpace(saved.CotaRD)
			reservedCodGrupo = strings.TrimSpace(saved.CodGrupo)
			reservedIDCotaReposicao = strings.TrimSpace(saved.IDCotaReposicao)
		}
	}
	return &ReserveResult{
		StatusCode:      resp.StatusCode,
		Body:            respBody,
		CotaRD:          reservedCotaRD,
		CodGrupo:        reservedCodGrupo,
		IDCotaReposicao: reservedIDCotaReposicao,
	}, nil
}

func (e *Engine) saveCotasReservadas(ctx context.Context, user *db.User, payload bookingPayload) (*db.ReservedCotaRecord, error) {
	if err := e.sendLogInfo(ctx, user, "abertura RC"); err != nil {
		log.Printf("[go] loginfo user=%s err=%v", user.CodUsuario, err)
	}

	q := url.Values{}
	q.Set("codconcessionaria", user.CodEmpresa)
	q.Set("cpfcnpj", user.CPF)
	url := e.cfg.APIBaseURL + "/cotasReposicoesReservadas?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("versao", "0.1.0")
	req.Header.Set("Authorization", "Bearer "+user.Token)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		log.Printf("[go] cotasreservadas user=%s status=%d body=%s", user.CodUsuario, resp.StatusCode, truncate(respBody, 250))
		return nil, nil
	}

	items, err := parseReservedCotasResponse(respBody)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	rows := make([]db.ReservedCota, 0, len(items))
	for _, item := range items {
		rows = append(rows, db.ReservedCota{
			UsuarioReserva:     item.UsuarioReserva,
			NumDocumentoPessoa: item.NumDocumentoPessoa,
			CodGrupo:           item.CodGrupo,
			CotaRD:             item.CotaRD,
			CodModelo:          item.CodModelo,
			IDCotaReposicao:    payload.IDCotaReposicao,
		})
	}

	inserted, err := e.store.SaveCotasReservadas(ctx, rows)
	if err != nil {
		return nil, err
	}
	log.Printf("[go] cotasreservadas user=%s fetched=%d inserted=%d idCotaReposicao=%s",
		user.CodUsuario, len(items), inserted, payload.IDCotaReposicao)

	latest, err := e.store.GetLatestReservedCotaByIDCotaReposicao(ctx, payload.IDCotaReposicao, user.CPF)
	if err == nil {
		return latest, nil
	}
	if err != sql.ErrNoRows {
		log.Printf("[go] cotasreservadas user=%s idCotaReposicao=%s err latest=%v", user.CodUsuario, payload.IDCotaReposicao, err)
	}
	return nil, nil
}

func (e *Engine) sendLogInfo(ctx context.Context, user *db.User, evento string) error {
	payload := logInfoPayload{
		CPFVendedor: user.CPF,
		CodUsuario:  user.CodUsuario,
		CodEmpresa:  user.CodEmpresa,
		Plataforma:  "DESKTOP",
		Token:       user.Token,
		Evento:      strings.TrimSpace(evento),
		Abertura:    time.Now().Format("02/01/2006, 15:04:05"),
		Fechamento:  "NA",
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.cfg.APIBaseURL+"/loginfo", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("versao", "0.1.0")
	req.Header.Set("Authorization", "Bearer "+user.Token)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := e.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status=%d body=%s", resp.StatusCode, truncate(readBody(resp), 250))
	}
	return nil
}

func parseReservedCotasResponse(body string) ([]reservedCotaAPIItem, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, nil
	}

	var generic any
	if err := json.Unmarshal([]byte(body), &generic); err != nil {
		return nil, err
	}
	out := make([]reservedCotaAPIItem, 0, 16)
	extractReservedItems(generic, &out)
	return out, nil
}

func extractReservedItems(v any, out *[]reservedCotaAPIItem) {
	switch cur := v.(type) {
	case map[string]any:
		item := reservedCotaAPIItem{
			UsuarioReserva:     lookupString(cur, "usuarioReserva"),
			NumDocumentoPessoa: lookupString(cur, "numDocumentoPessoa"),
			CodGrupo:           lookupString(cur, "codGrupo"),
			CotaRD:             lookupString(cur, "cotaRD"),
			CodModelo:          lookupString(cur, "codModelo"),
		}
		if strings.TrimSpace(item.CodGrupo) != "" || strings.TrimSpace(item.CotaRD) != "" {
			*out = append(*out, item)
		}
		for _, vv := range cur {
			extractReservedItems(vv, out)
		}
	case []any:
		for _, vv := range cur {
			extractReservedItems(vv, out)
		}
	}
}

func lookupString(m map[string]any, key string) string {
	for k, v := range m {
		if strings.EqualFold(k, key) {
			switch t := v.(type) {
			case string:
				return t
			case float64:
				return fmt.Sprintf("%.0f", t)
			default:
				return strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
	}
	return ""
}

func applyLoteriaFilter(cotas []db.Cota, mode, loteriaFederal, acrescimo string) []db.Cota {
	lot := strings.TrimSpace(loteriaFederal)
	if lot == "" {
		return cotas
	}
	lotInt, err := parseInt(lot)
	if err != nil || lotInt <= 0 {
		log.Printf("[go] loteria_federal inválida: %q", loteriaFederal)
		return cotas
	}

	seenP := map[int]struct{}{}
	participantes := []int{}
	for _, c := range cotas {
		if c.Participantes <= 0 {
			continue
		}
		if _, ok := seenP[c.Participantes]; ok {
			continue
		}
		seenP[c.Participantes] = struct{}{}
		participantes = append(participantes, c.Participantes)
	}
	if len(participantes) == 0 {
		return cotas
	}

	baseCotas := make([]int, 0, len(participantes))
	for _, p := range participantes {
		baseCotas = append(baseCotas, calcLoteriaFederal(lotInt, p))
	}

	allowed := map[int]struct{}{}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "b4":
		raio := 0
		v := strings.TrimSpace(acrescimo)
		if v != "" {
			if iv, err := parseInt(v); err == nil {
				raio = abs(iv)
			}
		}
		for _, c := range baseCotas {
			for i := c - raio; i <= c+raio; i++ {
				allowed[i] = struct{}{}
			}
		}
		expanded := make([]int, 0, len(allowed))
		for c := range allowed {
			expanded = append(expanded, c)
		}
		sort.Ints(expanded)
		log.Printf("[go][b4] cotas_sorteadas=%v raio=%d total_expandidas=%d cotas_expandidas=%v", baseCotas, raio, len(allowed), expanded)
	default: // b1
		off := 0
		v := strings.TrimSpace(acrescimo)
		if v != "" {
			if strings.HasPrefix(v, "+") || strings.HasPrefix(v, "-") {
				if iv, err := parseInt(v); err == nil {
					off = iv
				}
			} else {
				log.Printf("[go][b1] acrescimo_decrescimo inválido sem sinal: %q", v)
			}
		}
		for _, c := range baseCotas {
			allowed[c+off] = struct{}{}
		}
		log.Printf("[go][b1] cotas_sorteadas=%v offset=%d", baseCotas, off)
	}

	filtered := make([]db.Cota, 0, len(cotas))
	for _, c := range cotas {
		if _, ok := allowed[c.Cota]; ok {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func calcLoteriaFederal(numeroLoteria, participantes int) int {
	if participantes <= 0 {
		return 0
	}
	divisao := float64(numeroLoteria) / float64(participantes)
	decimal := divisao - float64(int(divisao))
	result := decimal * float64(participantes)
	return int(math.Round(result*10000) / 10000)
}

func shouldSkipFailed422(status int, body string) bool {
	if status != 422 {
		return false
	}
	for _, code := range nonFailed422Codes {
		if strings.Contains(body, code) {
			return true
		}
	}
	return false
}

func parseInt(raw string) (int, error) {
	var sign int = 1
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		sign = -1
		s = s[1:]
	}
	if s == "" {
		return 0, fmt.Errorf("invalid")
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid")
		}
		n = n*10 + int(ch-'0')
	}
	return sign * n, nil
}

func readBody(resp *http.Response) string {
	data, err := io.ReadAll(resp.Body)
	if err != nil || len(data) == 0 {
		return ""
	}
	return string(data)
}

func addStatus(m *Metrics, code int) {
	ptrAny, _ := m.HTTPStatus.LoadOrStore(code, new(int64))
	ptr := ptrAny.(*int64)
	atomic.AddInt64(ptr, 1)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
