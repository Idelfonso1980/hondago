package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"honda/go-engine/internal/db"
)

const (
	loginPath = "/login"
)

type authPayload struct {
	CodUsuario         string `json:"codUsuario"`
	CodEmpresa         string `json:"codEmpresa"`
	Senha              string `json:"senha"`
	VersaoApp          string `json:"versaoApp"`
	SistemaOperacional string `json:"sistemaOperacional"`
}

type loginResponse struct {
	Retorno struct {
		TokenDigital string `json:"tokenDigital"`
	} `json:"retorno"`
}

func (e *Engine) UpdateTokens(ctx context.Context) error {
	tokenPrincipal := strings.TrimSpace(e.cfg.Booking.TokenPrincipal)
	if tokenPrincipal == "" {
		return fmt.Errorf("token_principal vazio (defina HONDAGO_TOKEN_PRINCIPAL no .env ou BOOKING.token_principal no config.ini)")
	}

	users, err := e.store.LoadUsersForAuth(ctx, e.cfg.Booking.CPF, e.cfg.Booking.CodEmpre)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("nenhum usuario encontrado para autenticacao")
	}

	log.Printf("[go][auth] iniciando atualizacao de token para %d usuario(s)", len(users))
	okCount := 0
	failCount := 0

	for _, user := range users {
		if err := ctx.Err(); err != nil {
			return err
		}

		token, status, respBody, debugRaw, err := e.loginB3(ctx, user, tokenPrincipal)
		if err != nil {
			failCount++
			_ = e.store.ClearUserToken(ctx, user.ID)
			log.Printf("[go][auth] user=%s cpf=%s erro=%v", user.CodUsuario, user.CPF, err)
			if debugRaw != "" {
				log.Printf("[go][auth][debug_raw] %s", debugRaw)
			}
			continue
		}
		if status != http.StatusOK || token == "" {
			failCount++
			_ = e.store.ClearUserToken(ctx, user.ID)
			log.Printf("[go][auth] user=%s cpf=%s status=%d body=%s", user.CodUsuario, user.CPF, status, truncate(respBody, 400))
			if debugRaw != "" {
				log.Printf("[go][auth][debug_raw] %s", debugRaw)
			}
			continue
		}
		if err := e.store.SetUserToken(ctx, user.ID, token); err != nil {
			failCount++
			log.Printf("[go][auth] user=%s cpf=%s salvar_token_err=%v", user.CodUsuario, user.CPF, err)
			continue
		}

		okCount++
		log.Printf("[go][auth] user=%s cpf=%s token_atualizado", user.CodUsuario, user.CPF)
	}

	log.Printf("[go][auth] finalizado sucesso=%d falha=%d total=%d", okCount, failCount, len(users))
	if okCount == 0 {
		return fmt.Errorf("nenhum token atualizado com sucesso")
	}
	return nil
}

func (e *Engine) UpdateTokenByUserID(ctx context.Context, userID int64) error {
	tokenPrincipal := strings.TrimSpace(e.cfg.Booking.TokenPrincipal)
	if tokenPrincipal == "" {
		return fmt.Errorf("token_principal vazio (defina HONDAGO_TOKEN_PRINCIPAL no .env ou BOOKING.token_principal no config.ini)")
	}
	user, err := e.store.LoadUserForAuthByID(ctx, userID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(user.CodUsuario) == "" || strings.TrimSpace(user.CodEmpresa) == "" || strings.TrimSpace(user.Senha) == "" {
		return fmt.Errorf("usuario sem credenciais completas para login")
	}

	token, status, respBody, debugRaw, err := e.loginB3(ctx, *user, tokenPrincipal)
	if err != nil {
		_ = e.store.ClearUserToken(ctx, user.ID)
		if debugRaw != "" {
			return fmt.Errorf("%w | outbound=%s", err, debugRaw)
		}
		return err
	}
	if status != http.StatusOK || token == "" {
		_ = e.store.ClearUserToken(ctx, user.ID)
		if debugRaw != "" {
			return fmt.Errorf("falha login status=%d body=%s | outbound=%s", status, truncate(respBody, 400), debugRaw)
		}
		return fmt.Errorf("falha login status=%d body=%s", status, truncate(respBody, 400))
	}
	return e.store.SetUserToken(ctx, user.ID, token)
}

func (e *Engine) loginB3(ctx context.Context, user db.User, tokenPrincipal string) (string, int, string, string, error) {
	payload := authPayload{
		CodUsuario:         user.CodUsuario,
		CodEmpresa:         user.CodEmpresa,
		Senha:              user.Senha,
		VersaoApp:          "1.0.0",
		SistemaOperacional: "DESKTOP",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, "", "", err
	}

	url := strings.TrimRight(e.cfg.APIBaseURL, "/") + loginPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", 0, "", "", err
	}
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("origin", "https://vendadigital.consorciohonda.com.br")
	req.Header.Set("referer", "https://vendadigital.consorciohonda.com.br/")
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-language", "pt-BR,pt;q=0.9")
	req.Header.Set("versao", "0.1.0")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+tokenPrincipal)
	debugRaw := buildOutboundDebug(url, payload, req.Header.Get("Authorization"), req.Header.Get("origin"), req.Header.Get("referer"), req.Header.Get("user-agent"))

	resp, err := e.http.Do(req)
	if err != nil {
		return "", 0, "", debugRaw, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	respBody := string(data)
	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, respBody, debugRaw, nil
	}

	var parsed loginResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", resp.StatusCode, respBody, debugRaw, err
	}
	return strings.TrimSpace(parsed.Retorno.TokenDigital), resp.StatusCode, respBody, debugRaw, nil
}

func buildOutboundDebug(url string, payload authPayload, authorization, origin, referer, userAgent string) string {
	if !debugAuthRawEnabled() {
		return ""
	}
	out := map[string]any{
		"url":    url,
		"method": http.MethodPost,
		"headers": map[string]string{
			"Authorization":   authorization,
			"origin":          origin,
			"referer":         referer,
			"user-agent":      userAgent,
			"accept":          "application/json",
			"accept-language": "pt-BR,pt;q=0.9",
			"versao":          "0.1.0",
			"Content-Type":    "application/json",
		},
		"json": payload,
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func debugAuthRawEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("HONDAGO_DEBUG_AUTH_RAW")))
	return v == "1" || v == "true" || v == "yes" || v == "y" || v == "sim"
}
