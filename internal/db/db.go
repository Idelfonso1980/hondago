package db

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type Store struct {
	DB     *sql.DB
	driver string
}

var utcMinus3Loc = loadUTCMinus3Location()

func loadUTCMinus3Location() *time.Location {
	loc, err := time.LoadLocation("America/Fortaleza")
	if err == nil {
		return loc
	}
	return time.FixedZone("UTC-3", -3*60*60)
}

func nowUTCMinus3() time.Time {
	return time.Now().In(utcMinus3Loc)
}

func formatDateTimeUTCMinus3(t time.Time) string {
	return t.In(utcMinus3Loc).Format("2006-01-02 15:04:05")
}

type User struct {
	ID            int64
	CPF           string
	CodEmpresa    string
	CodUsuario    string
	Senha         string
	Token         string
	LastRequest   sql.NullTime
	CooldownUntil sql.NullTime
	InFlight      sql.NullInt64
	Error401Count sql.NullInt64
	Error429Count sql.NullInt64
	BlockedUntil  sql.NullTime
	PriorityScore sql.NullFloat64
}

type AuthRecord struct {
	ID                int64
	CPF               string
	CodEmpresa        string
	CodUsuario        string
	CodConcessionaria string
	Senha             string
	Token             sql.NullString
	TokenB3           sql.NullString
	LastRequest       sql.NullTime
	CooldownUntil     sql.NullTime
	InFlight          sql.NullInt64
	Error401Count     sql.NullInt64
	Error429Count     sql.NullInt64
	BlockedUntil      sql.NullTime
	PriorityScore     sql.NullFloat64
}

type AppUserRecord struct {
	ID                  int64
	Username            string
	PasswordHash        string
	DisplayName         string
	CPF                 sql.NullString
	Filial              sql.NullString
	Email               sql.NullString
	Phone               sql.NullString
	MFAEnabled          int64
	MFASecret           sql.NullString
	Role                string
	IsActive            int64
	FailedLoginAttempts int64
	LockedUntil         sql.NullTime
	LastLoginAt         sql.NullTime
	UpdatedAt           sql.NullTime
	CreatedAt           sql.NullTime
}

type AuditLogRecord struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	Action      string `json:"action"`
	Entity      string `json:"entity"`
	EntityID    string `json:"entity_id"`
	BeforeState string `json:"before_state"`
	AfterState  string `json:"after_state"`
	CreatedAt   string `json:"created_at"`
}

type AppSessionRecord struct {
	Token           string
	UserID          int64
	Username        string
	DisplayName     string
	CPF             sql.NullString
	Filial          sql.NullString
	Role            string
	Permissions     []string
	AuthenticatedAt time.Time
}

type Cota struct {
	ID            int64
	IDGrupo       int64
	Produto       string
	Vencimento    int
	Tipo          string
	Grupo         int
	Cota          int
	R             int
	D             int
	Participantes int
}

type ReservedCota struct {
	RequestID          int64
	UsuarioReserva     string
	NumDocumentoPessoa string
	CodGrupo           string
	CotaRD             string
	CodModelo          string
	IDCotaReposicao    string
}

type ReservedCotaRecord struct {
	ID                 int64
	RequestID          int64
	UsuarioReserva     string
	NumDocumentoPessoa string
	CodGrupo           string
	CotaRD             string
	CodModelo          string
	IDCotaReposicao    string
	CreatedAt          sql.NullString
}

type ManualNotificationRecord struct {
	ID            int64
	SolicitacaoID sql.NullInt64
	CPF           string
	Vendedor      string
	Filial        string
	Canal         string
	Mensagem      string
	Status        string
	CopiadaEm     sql.NullString
	EnviadaEm     sql.NullString
	UsuarioAcao   string
	CreatedAt     sql.NullString
	UpdatedAt     sql.NullString
}

type IDsGrupoDisponivelRecord struct {
	ID            int64
	IDGrupo       int64
	Produto       string
	Vencimento    int64
	Prazo         int64
	Tipo          string
	Grupo         int64
	Cota          int64
	R             int64
	D             int64
	Booked        int64
	CreatedAt     sql.NullString
	Participantes int64
	Failed        int64
}

type ModeloRecord struct {
	ID       int64
	IDModelo int64
	Modelo   string
	Status   string
}

type ProdutoRecord struct {
	ID        int64
	IDProduto int64
	Produto   string
	Status    string
}

type GrupoAtivoRecord struct {
	ID                      int64
	Grupo                   int64
	Vencimento              int64
	QtdParticipantes        int64
	DataAssembleiaInaugural sql.NullString
	Plano                   string
	Prazo                   int64
	TipoGrupo               string
	Modelos                 string
	Status                  string
	CreatedAt               sql.NullString
	UpdatedAt               sql.NullString
}

type AssembleiaRecord struct {
	ID                 int64
	CotaRD             string
	DataContemplacao   sql.NullString
	TipoContemplacao   string
	DataDesclassificao sql.NullString
	ClientName         string
	PercLance          sql.NullFloat64
	Vendedor           string
	Grupo              sql.NullInt64
	LoteriaFederal     sql.NullInt64
	GrupoCotaRD        string
}

type SolicitacaoRecord struct {
	ID                  int64
	DataHoraSolicitacao sql.NullString
	DataSolicitacao     sql.NullString
	HoraSolicitacao     sql.NullString
	Filial              string
	Vendedor            string
	CPF                 string
	Modelo              string
	Plano               string
	QtdeParcelas        sql.NullInt64
	PercLance           sql.NullFloat64
	ComRestricao        string
	Grupo               sql.NullInt64
	Notes               string
	IDCota              sql.NullInt64
	GrupoAtendido       sql.NullInt64
	CotaRD              string
	DataHoraAtendimento sql.NullString
	DataAtendimento     sql.NullString
	HoraAtendimento     sql.NullString
	Situacao            string
	LanceContemplacao   string
}

type requestRelationIDs struct {
	RequesterUserID  sql.NullInt64
	VendorIdentityID sql.NullInt64
	APIAccountID     sql.NullInt64
}

type CotaFilter struct {
	Produto    []int
	Vencimento []int
	Grupo      []int
	IDCota     []int
	TipoGrupo  string
	Limit      int
}

func Open(path string) (*Store, error) {
	return OpenWithURL(path, "")
}

func OpenWithURL(path, databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) != "" {
		return openPostgres(databaseURL)
	}
	return openSQLite(path)
}

func openSQLite(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		db.Close()
		return nil, err
	}
	store := &Store{DB: db, driver: "sqlite"}
	if err := store.EnsureLegacySchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.EnsureDefaultAppUser(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	if err := store.MigrateAuthPasswordsToEncrypted(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func openPostgres(databaseURL string) (*Store, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	store := &Store{DB: db, driver: "pgx"}
	if err := store.EnsureLegacySchema(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.EnsureDefaultAppUser(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.MigrateAuthPasswordsToEncrypted(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) bind(query string) string {
	if s == nil || s.driver != "pgx" {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	arg := 1
	for _, ch := range query {
		if ch == '?' {
			b.WriteString(fmt.Sprintf("$%d", arg))
			arg++
			continue
		}
		b.WriteRune(ch)
	}
	return b.String()
}

func (s *Store) Rebind(query string) string {
	return s.bind(query)
}

func (s *Store) IsPostgres() bool {
	return s != nil && s.driver == "pgx"
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) MigrateAuthPasswordsToEncrypted(ctx context.Context) error {
	// If key is not configured yet, keep backward compatibility and skip migration.
	if _, err := getSecretsKey(); err != nil {
		return nil
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT id, account_password FROM api_accounts WHERE account_password IS NOT NULL AND account_password <> ''")
	if err != nil {
		return err
	}
	defer rows.Close()

	type item struct {
		id  int64
		pwd string
	}
	var pending []item
	for rows.Next() {
		var id int64
		var pwd string
		if err := rows.Scan(&id, &pwd); err != nil {
			return err
		}
		pwd = strings.TrimSpace(pwd)
		if pwd == "" || strings.HasPrefix(pwd, encryptedPrefix) {
			continue
		}
		pending = append(pending, item{id: id, pwd: pwd})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range pending {
		enc, err := encryptAtRest(p.pwd)
		if err != nil {
			return err
		}
		if _, err := s.DB.ExecContext(ctx, s.bind("UPDATE api_accounts SET account_password=? WHERE id=?"), enc, p.id); err != nil {
			return err
		}
	}
	return nil
}

func HashPassword(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("account_password vazia")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(trimmed), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(hash, raw string) bool {
	if strings.TrimSpace(hash) == "" || strings.TrimSpace(raw) == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}

func (s *Store) EnsureDefaultAppUser(ctx context.Context) error {
	var totalUsers int64
	err := s.DB.QueryRowContext(ctx, "SELECT COUNT(1) FROM users").Scan(&totalUsers)
	if err == nil && totalUsers > 0 {
		return nil
	}
	if err != nil {
		return err
	}

	bootstrapUser := strings.TrimSpace(os.Getenv("HONDAGO_BOOTSTRAP_ADMIN_USER"))
	if bootstrapUser == "" {
		bootstrapUser = "admin"
	}
	bootstrapPass := strings.TrimSpace(os.Getenv("HONDAGO_BOOTSTRAP_ADMIN_PASSWORD"))
	if bootstrapPass == "" {
		return fmt.Errorf("nenhum usuÃ¡rio de aplicaÃ§Ã£o encontrado; defina HONDAGO_BOOTSTRAP_ADMIN_PASSWORD para bootstrap inicial")
	}
	if err := validateStrongPassword(bootstrapPass); err != nil {
		return fmt.Errorf("senha bootstrap invÃ¡lida: %w", err)
	}

	hash, err := HashPassword(bootstrapPass)
	if err != nil {
		return err
	}
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	_, err = s.DB.ExecContext(
		ctx,
		s.bind(`INSERT INTO users (username, password_hash, full_name, role, is_active, failed_login_attempts, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, 0, ?, ?)`,
		),
		bootstrapUser, hash, "Administrador", "admin", now, now,
	)
	return err
}

func validateStrongPassword(raw string) error {
	if len(raw) < 12 {
		return fmt.Errorf("mÃ­nimo de 12 caracteres")
	}
	var hasUpper, hasLower, hasDigit, hasSymbol bool
	for _, r := range raw {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()-_=+[]{};:,.?/\\|", r):
			hasSymbol = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSymbol {
		return fmt.Errorf("use maiÃºscula, minÃºscula, nÃºmero e sÃ­mbolo")
	}
	return nil
}

func (s *Store) SearchAppUsers(ctx context.Context, query string, limit int) ([]AppUserRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	cols := `id, username, password_hash, full_name, cpf, branch, email, phone, mfa_enabled, mfa_secret, role, is_active, failed_login_attempts, locked_until, last_login_at, updated_at, created_at`
	q := strings.TrimSpace(query)

	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = s.DB.QueryContext(ctx, s.bind("SELECT "+cols+" FROM users ORDER BY id ASC LIMIT ?"), limit)
	} else {
		likeOp := "LIKE"
		if s.driver == "pgx" {
			likeOp = "ILIKE"
		}
		like := "%" + q + "%"
		rows, err = s.DB.QueryContext(
			ctx,
			s.bind(`SELECT `+cols+` FROM users
			 WHERE CAST(id AS TEXT)=?
			    OR CAST(COALESCE(username,'') AS TEXT) `+likeOp+` ?
			    OR CAST(COALESCE(full_name,'') AS TEXT) `+likeOp+` ?
			    OR CAST(COALESCE(cpf,'') AS TEXT) `+likeOp+` ?
			    OR CAST(COALESCE(branch,'') AS TEXT) `+likeOp+` ?
			    OR CAST(COALESCE(email,'') AS TEXT) `+likeOp+` ?
			    OR CAST(COALESCE(role,'') AS TEXT) `+likeOp+` ?
			 ORDER BY id ASC
			 LIMIT ?`),
			q, like, like, like, like, like, like, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AppUserRecord, 0, limit)
	for rows.Next() {
		var r AppUserRecord
		if err := rows.Scan(
			&r.ID, &r.Username, &r.PasswordHash, &r.DisplayName, &r.CPF, &r.Filial, &r.Email, &r.Phone, &r.MFAEnabled, &r.MFASecret, &r.Role, &r.IsActive,
			&r.FailedLoginAttempts, &r.LockedUntil, &r.LastLoginAt, &r.UpdatedAt, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetAppUserByID(ctx context.Context, id int64) (*AppUserRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id invalido")
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT id, username, password_hash, full_name, cpf, branch, email, phone, mfa_enabled, mfa_secret, role, is_active, failed_login_attempts, locked_until, last_login_at, updated_at, created_at
		 FROM users
		 WHERE id=?`),
		id,
	)
	var r AppUserRecord
	if err := row.Scan(
		&r.ID, &r.Username, &r.PasswordHash, &r.DisplayName, &r.CPF, &r.Filial, &r.Email, &r.Phone, &r.MFAEnabled, &r.MFASecret, &r.Role, &r.IsActive,
		&r.FailedLoginAttempts, &r.LockedUntil, &r.LastLoginAt, &r.UpdatedAt, &r.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) FindAppUserByUsername(ctx context.Context, username string) (*AppUserRecord, error) {
	u := strings.TrimSpace(username)
	if u == "" {
		return nil, fmt.Errorf("username vazio")
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT id, username, password_hash, full_name, cpf, branch, email, phone, mfa_enabled, mfa_secret, role, is_active, failed_login_attempts, locked_until, last_login_at, updated_at, created_at
		 FROM users
		 WHERE username=?
		 LIMIT 1`),
		u,
	)
	var r AppUserRecord
	if err := row.Scan(
		&r.ID, &r.Username, &r.PasswordHash, &r.DisplayName, &r.CPF, &r.Filial, &r.Email, &r.Phone, &r.MFAEnabled, &r.MFASecret, &r.Role, &r.IsActive,
		&r.FailedLoginAttempts, &r.LockedUntil, &r.LastLoginAt, &r.UpdatedAt, &r.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) GetAppUserByCPF(ctx context.Context, cpf string) (*AppUserRecord, error) {
	c := strings.TrimSpace(cpf)
	if c == "" {
		return nil, fmt.Errorf("cpf vazio")
	}
	// Busca por CPF exato ou limpando formataÃ§Ã£o comum
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT id, username, password_hash, full_name, cpf, branch, email, phone, mfa_enabled, mfa_secret, role, is_active, failed_login_attempts, locked_until, last_login_at, updated_at, created_at
		 FROM users
		 WHERE cpf = ? OR REPLACE(REPLACE(cpf, '.', ''), '-', '') = ?
		 LIMIT 1`),
		c, c,
	)
	var r AppUserRecord
	if err := row.Scan(
		&r.ID, &r.Username, &r.PasswordHash, &r.DisplayName, &r.CPF, &r.Filial, &r.Email, &r.Phone, &r.MFAEnabled, &r.MFASecret, &r.Role, &r.IsActive,
		&r.FailedLoginAttempts, &r.LockedUntil, &r.LastLoginAt, &r.UpdatedAt, &r.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) SaveAppUser(ctx context.Context, r AppUserRecord, plainPassword string) (int64, error) {
	username := strings.TrimSpace(r.Username)
	if username == "" {
		return 0, fmt.Errorf("username obrigatorio")
	}
	var existingID int64
	err := s.DB.QueryRowContext(ctx, s.bind("SELECT id FROM users WHERE username=? LIMIT 1"), username).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	if existingID > 0 && existingID != r.ID {
		return 0, fmt.Errorf("username ja cadastrado")
	}
	displayName := strings.TrimSpace(r.DisplayName)
	if displayName == "" {
		displayName = username
	}
	role := strings.TrimSpace(r.Role)
	if role == "" {
		role = "operador"
	}
	isActive := int64(0)
	if r.IsActive != 0 {
		isActive = 1
	}

	if r.ID > 0 {
		if strings.TrimSpace(plainPassword) != "" {
			hash, err := HashPassword(plainPassword)
			if err != nil {
				return 0, err
			}
			now := formatDateTimeUTCMinus3(nowUTCMinus3())
			_, err = s.DB.ExecContext(
				ctx,
			s.bind(`UPDATE users
				 SET username=?,
				     password_hash=?,
				     full_name=?,
				     cpf=?,
				     branch=?,
				     email=?,
				     phone=?,
				     mfa_enabled=?,
				     mfa_secret=?,
				     role=?,
				     is_active=?,
				     updated_at=?
				 WHERE id=?`),
				username, hash, displayName, nullableTrimmed(r.CPF), nullableTrimmed(r.Filial), nullableTrimmed(r.Email), nullableTrimmed(r.Phone), r.MFAEnabled, nullableTrimmed(r.MFASecret), role, isActive, now, r.ID,
			)
			if err != nil {
				return 0, err
			}
			return r.ID, nil
		}

		now := formatDateTimeUTCMinus3(nowUTCMinus3())
		_, err = s.DB.ExecContext(
			ctx,
		s.bind(`UPDATE users
			 SET username=?,
			     full_name=?,
			     cpf=?,
			     branch=?,
			     email=?,
			     phone=?,
			     mfa_enabled=?,
			     mfa_secret=?,
			     role=?,
			     is_active=?,
			     updated_at=?
			 WHERE id=?`),
			username, displayName, nullableTrimmed(r.CPF), nullableTrimmed(r.Filial), nullableTrimmed(r.Email), nullableTrimmed(r.Phone), r.MFAEnabled, nullableTrimmed(r.MFASecret), role, isActive, now, r.ID,
		)
		if err != nil {
			return 0, err
		}
		return r.ID, nil
	}

	if strings.TrimSpace(plainPassword) == "" {
		return 0, fmt.Errorf("account_password obrigatoria para novo usuario")
	}
	hash, err := HashPassword(plainPassword)
	if err != nil {
		return 0, err
	}
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	if s.driver == "pgx" {
		var id int64
		err = s.DB.QueryRowContext(
			ctx,
			s.bind(`INSERT INTO users (username, password_hash, full_name, cpf, branch, email, phone, mfa_enabled, mfa_secret, role, is_active, failed_login_attempts, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?) RETURNING id`),
			username, hash, displayName, nullableTrimmed(r.CPF), nullableTrimmed(r.Filial), nullableTrimmed(r.Email), nullableTrimmed(r.Phone), r.MFAEnabled, nullableTrimmed(r.MFASecret), role, isActive, now, now,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		s.bind(`INSERT INTO users (username, password_hash, full_name, cpf, branch, email, phone, mfa_enabled, mfa_secret, role, is_active, failed_login_attempts, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`),
		username, hash, displayName, nullableTrimmed(r.CPF), nullableTrimmed(r.Filial), nullableTrimmed(r.Email), nullableTrimmed(r.Phone), r.MFAEnabled, nullableTrimmed(r.MFASecret), role, isActive, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteAppUser(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM users WHERE id=?"), id)
	return err
}

func (s *Store) RegisterAppUserLoginFailure(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	now := nowUTCMinus3()
	updatedAt := formatDateTimeUTCMinus3(now)
	lockedUntil := formatDateTimeUTCMinus3(now.Add(15 * time.Minute))
	_, err := s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE users
		 SET failed_login_attempts=COALESCE(failed_login_attempts,0)+1,
		     locked_until=CASE WHEN COALESCE(failed_login_attempts,0)+1 >= 5 THEN ? ELSE locked_until END,
		     updated_at=?
		 WHERE id=?`),
		lockedUntil, updatedAt, id,
	)
	return err
}

func (s *Store) RegisterAppUserLoginSuccess(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	_, err := s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE users
		 SET failed_login_attempts=0,
		     locked_until=NULL,
		     last_login_at=?,
		     updated_at=?
		 WHERE id=?`),
		now, now, id,
	)
	return err
}

func nullableTrimmed(in sql.NullString) any {
	if !in.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(in.String)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func onlyDigits(raw string) string {
	if raw == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *Store) EnsureAuthControlColumns(ctx context.Context) error {
	var authExists int
	if err := s.DB.QueryRowContext(
		ctx,
		"SELECT 1 FROM sqlite_master WHERE type='table' AND name='api_accounts' LIMIT 1",
	).Scan(&authExists); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("tabela 'api_accounts' nÃƒÆ’Ã‚Â£o encontrada no banco selecionado")
		}
		return fmt.Errorf("check api_accounts table: %w", err)
	}

	cols := map[string]string{
		"cooldown_until":  "DATETIME",
		"in_flight":       "INTEGER DEFAULT 0",
		"error_401_count": "INTEGER DEFAULT 0",
		"error_429_count": "INTEGER DEFAULT 0",
		"blocked_until":   "DATETIME",
		"priority_score":  "REAL DEFAULT 0",
	}

	exists := map[string]bool{}
	rows, err := s.DB.QueryContext(ctx, "PRAGMA table_info(api_accounts)")
	if err != nil {
		return fmt.Errorf("pragma api_accounts table: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		exists[name] = true
	}
	for col, ddl := range cols {
		if exists[col] {
			continue
		}
		query := fmt.Sprintf("ALTER TABLE api_accounts ADD COLUMN %s %s", col, ddl)
		if _, err := s.DB.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("add column %s: %w", col, err)
		}
	}
	return nil
}

func (s *Store) LoadUsers(ctx context.Context, cpfFilter, codEmpreFilter []string) ([]User, error) {
	base := `
SELECT
  id, cpf, company_code, account_user, account_password, COALESCE(token, '') AS token, last_request_at,
  cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score
FROM api_accounts
WHERE token IS NOT NULL AND token <> ''
`
	args := []any{}
	if len(cpfFilter) > 0 {
		base += " AND cpf IN (" + placeholders(len(cpfFilter)) + ")"
		for _, v := range cpfFilter {
			args = append(args, v)
		}
	}
	if len(codEmpreFilter) > 0 {
		base += " AND company_code IN (" + placeholders(len(codEmpreFilter)) + ")"
		for _, v := range codEmpreFilter {
			args = append(args, v)
		}
	}
	base += " ORDER BY COALESCE(priority_score,0) DESC, id ASC"

	bound := s.bind(base)
	rows, err := s.DB.QueryContext(ctx, bound, args...)
	if err != nil {
		return nil, fmt.Errorf("load users query failed: %w | sql=%s | args=%v", err, bound, args)
	}
	defer rows.Close()

	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.CPF, &u.CodEmpresa, &u.CodUsuario, &u.Senha, &u.Token, &u.LastRequest,
			&u.CooldownUntil, &u.InFlight, &u.Error401Count, &u.Error429Count, &u.BlockedUntil, &u.PriorityScore,
		); err != nil {
			return nil, err
		}
		if err := revealUserPassword(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) LoadUsersForAuth(ctx context.Context, cpfFilter, codEmpreFilter []string) ([]User, error) {
	base := `
SELECT
  id, cpf, company_code, account_user, account_password, COALESCE(token, '') AS token, last_request_at,
  cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score
FROM api_accounts
WHERE account_user IS NOT NULL AND account_user <> ''
  AND company_code IS NOT NULL AND company_code <> ''
  AND account_password IS NOT NULL AND account_password <> ''
`
	args := []any{}
	if len(cpfFilter) > 0 {
		base += " AND cpf IN (" + placeholders(len(cpfFilter)) + ")"
		for _, v := range cpfFilter {
			args = append(args, v)
		}
	}
	if len(codEmpreFilter) > 0 {
		base += " AND company_code IN (" + placeholders(len(codEmpreFilter)) + ")"
		for _, v := range codEmpreFilter {
			args = append(args, v)
		}
	}
	base += " ORDER BY id ASC"

	bound := s.bind(base)
	rows, err := s.DB.QueryContext(ctx, bound, args...)
	if err != nil {
		return nil, fmt.Errorf("load users for auth query failed: %w | sql=%s | args=%v", err, bound, args)
	}
	defer rows.Close()

	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.ID, &u.CPF, &u.CodEmpresa, &u.CodUsuario, &u.Senha, &u.Token, &u.LastRequest,
			&u.CooldownUntil, &u.InFlight, &u.Error401Count, &u.Error429Count, &u.BlockedUntil, &u.PriorityScore,
		); err != nil {
			return nil, err
		}
		if err := revealUserPassword(&u); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) LoadCotas(ctx context.Context, f CotaFilter) ([]Cota, error) {
	base := `
SELECT
  CAST(id AS INTEGER) AS id,
  CAST(group_api_id AS INTEGER) AS group_api_id,
  CAST(products AS TEXT) AS products,
  CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER) AS due_day,
  CAST(group_kind AS TEXT) AS group_kind,
  CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) AS group_code,
  CAST(COALESCE(NULLIF(CAST(quota AS TEXT), ''), '0') AS INTEGER) AS quota,
  CAST(COALESCE(NULLIF(CAST(r AS TEXT), ''), '0') AS INTEGER) AS r,
  CAST(COALESCE(NULLIF(CAST(d AS TEXT), ''), '0') AS INTEGER) AS d,
  CAST(COALESCE(NULLIF(CAST(participants AS TEXT), ''), '0') AS INTEGER) AS participants
FROM available_group_ids
WHERE booked = 0 AND failed = 0
`
	args := []any{}
	if len(f.Produto) > 0 {
		base += " AND CAST(COALESCE(NULLIF(CAST(products AS TEXT), ''), '0') AS INTEGER) IN (" + placeholders(len(f.Produto)) + ")"
		for _, v := range f.Produto {
			args = append(args, v)
		}
	}
	if len(f.Vencimento) > 0 {
		base += " AND CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER) IN (" + placeholders(len(f.Vencimento)) + ")"
		for _, v := range f.Vencimento {
			args = append(args, v)
		}
	}
	if len(f.Grupo) > 0 {
		base += " AND CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) IN (" + placeholders(len(f.Grupo)) + ")"
		for _, v := range f.Grupo {
			args = append(args, v)
		}
	}
	if len(f.IDCota) > 0 {
		ph := placeholders(len(f.IDCota))
		base += " AND (" +
			"CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER) IN (" + ph + ")" +
			" OR " +
			"CAST(COALESCE(NULLIF(CAST(group_api_id AS TEXT), ''), '0') AS INTEGER) IN (" + ph + ")" +
			" OR CAST(COALESCE(NULLIF(CAST(quota AS TEXT), ''), '0') AS INTEGER) IN (" + ph + ")" +
			" OR CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) IN (" + ph + ")" +
			")"
		for _, v := range f.IDCota {
			args = append(args, v)
		}
		for _, v := range f.IDCota {
			args = append(args, v)
		}
		for _, v := range f.IDCota {
			args = append(args, v)
		}
		for _, v := range f.IDCota {
			args = append(args, v)
		}
	}
	if strings.TrimSpace(f.TipoGrupo) != "" {
		base += " AND group_kind = ?"
		args = append(args, strings.TrimSpace(f.TipoGrupo))
	}
	base += " ORDER BY id ASC"
	if f.Limit > 0 {
		base += " LIMIT ?"
		args = append(args, f.Limit)
	}

	bound := s.bind(base)
	rows, err := s.DB.QueryContext(ctx, bound, args...)
	if err != nil {
		return nil, fmt.Errorf("load cotas query failed: %w | sql=%s | args=%v", err, bound, args)
	}
	defer rows.Close()

	out := []Cota{}
	for rows.Next() {
		var c Cota
		var id, group_api_id, due_day, group_code, quota, rVal, dVal, participants int64
		if err := rows.Scan(
			&id, &group_api_id, &c.Produto, &due_day, &c.Tipo, &group_code, &quota, &rVal, &dVal, &participants,
		); err != nil {
			return nil, err
		}
		c.ID = id
		c.IDGrupo = group_api_id
		c.Vencimento = int(due_day)
		c.Grupo = int(group_code)
		c.Cota = int(quota)
		c.R = int(rVal)
		c.D = int(dVal)
		c.Participantes = int(participants)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) MarkCotaBooked(ctx context.Context, rowID int64) error {
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	_, err := s.DB.ExecContext(
		ctx,
		s.bind("UPDATE available_group_ids SET booked=1, created_at=? WHERE id=?"),
		now, rowID,
	)
	return err
}

func (s *Store) MarkCotaFailed(ctx context.Context, rowID int64) error {
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	_, err := s.DB.ExecContext(
		ctx,
		s.bind("UPDATE available_group_ids SET failed=1, created_at=? WHERE id=?"),
		now, rowID,
	)
	return err
}

func (s *Store) SaveCotasReservadas(ctx context.Context, rows []ReservedCota) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	inserted := 0
	for _, row := range rows {
		group_code := strings.TrimSpace(row.CodGrupo)
		quota_rd := strings.TrimSpace(row.CotaRD)
		if group_code == "" || quota_rd == "" {
			continue
		}

		var one int
		err := tx.QueryRowContext(
			ctx,
			s.bind("SELECT 1 FROM reservations WHERE group_code=? AND quota_rd=? LIMIT 1"),
			group_code,
			quota_rd,
		).Scan(&one)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return inserted, err
		}

		doc := onlyDigits(strings.TrimSpace(row.NumDocumentoPessoa))
		now := formatDateTimeUTCMinus3(nowUTCMinus3())
		var requestID any = nil
		if row.RequestID > 0 {
			requestID = row.RequestID
		}
		_, err = tx.ExecContext(
			ctx,
			s.bind(`INSERT INTO reservations
			 (request_id, reserved_by, person_document, group_code, quota_rd, model_name, replacement_quota_id, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
			requestID,
			strings.TrimSpace(row.UsuarioReserva),
			doc,
			group_code,
			quota_rd,
			strings.TrimSpace(row.CodModelo),
			strings.TrimSpace(row.IDCotaReposicao),
			now,
		)
		if err != nil {
			return inserted, err
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

func (s *Store) SearchReservedCotas(ctx context.Context, query, fromDate, toDate string, limit int) ([]ReservedCotaRecord, error) {
	if limit <= 0 {
		limit = 500
	}
	q := strings.TrimSpace(query)
	from := strings.TrimSpace(fromDate)
	to := strings.TrimSpace(toDate)
	const cols = `id, reserved_by, person_document, group_code, quota_rd, model_name, replacement_quota_id, created_at`
	baseDateExpr := `SUBSTR(TRIM(CAST(created_at AS TEXT)), 1, 10)`
	whereParts := make([]string, 0, 3)
	args := make([]any, 0, 10)
	if from != "" {
		whereParts = append(whereParts, baseDateExpr+` >= ?`)
		args = append(args, from)
	}
	if to != "" {
		whereParts = append(whereParts, baseDateExpr+` <= ?`)
		args = append(args, to)
	}

	var rows *sql.Rows
	var err error
	if q == "" {
		querySQL := "SELECT " + cols + " FROM reservations"
		if len(whereParts) > 0 {
			querySQL += " WHERE " + strings.Join(whereParts, " AND ")
		}
		querySQL += " ORDER BY id DESC LIMIT ?"
		args = append(args, limit)
		rows, err = s.DB.QueryContext(ctx, s.bind(querySQL), args...)
	} else {
		textLike := "LIKE"
		if s.driver == "pgx" {
			textLike = "ILIKE"
		}
		like := "%" + q + "%"
		whereParts = append(whereParts, `(CAST(id AS TEXT)=?
   OR CAST(COALESCE(reserved_by,'') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(person_document,'') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(group_code,'') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(quota_rd,'') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(model_name,'') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(replacement_quota_id,'') AS TEXT) `+textLike+` ?)`)
		args = append(args, q, like, like, like, like, like, like)
		querySQL := "SELECT " + cols + " FROM reservations"
		if len(whereParts) > 0 {
			querySQL += " WHERE " + strings.Join(whereParts, " AND ")
		}
		querySQL += " ORDER BY id DESC LIMIT ?"
		args = append(args, limit)
		rows, err = s.DB.QueryContext(ctx, s.bind(querySQL), args...)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ReservedCotaRecord, 0, limit)
	for rows.Next() {
		var r ReservedCotaRecord
		if err := rows.Scan(
			&r.ID,
			&r.UsuarioReserva,
			&r.NumDocumentoPessoa,
			&r.CodGrupo,
			&r.CotaRD,
			&r.CodModelo,
			&r.IDCotaReposicao,
			&r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetLatestReservedCotaByIDCotaReposicao(ctx context.Context, replacement_quota_id, cpf string) (*ReservedCotaRecord, error) {
	replacement_quota_id = strings.TrimSpace(replacement_quota_id)
	cpf = strings.TrimSpace(cpf)
	if replacement_quota_id == "" {
		return nil, fmt.Errorf("replacement_quota_id vazio")
	}

	const cols = `id, request_id, reserved_by, person_document, group_code, quota_rd, model_name, replacement_quota_id`
	row := &ReservedCotaRecord{}
	var err error
	if cpf != "" {
		err = s.DB.QueryRowContext(
			ctx,
			s.bind(`SELECT `+cols+` FROM reservations
WHERE replacement_quota_id=? AND person_document=?
ORDER BY id DESC
LIMIT 1`),
			replacement_quota_id, cpf,
		).Scan(
			&row.ID, &row.RequestID, &row.UsuarioReserva, &row.NumDocumentoPessoa,
			&row.CodGrupo, &row.CotaRD, &row.CodModelo, &row.IDCotaReposicao,
		)
		if err == nil {
			return row, nil
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
	}

	err = s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT `+cols+` FROM reservations
WHERE replacement_quota_id=?
ORDER BY id DESC
LIMIT 1`),
		replacement_quota_id,
	).Scan(
		&row.ID, &row.RequestID, &row.UsuarioReserva, &row.NumDocumentoPessoa,
		&row.CodGrupo, &row.CotaRD, &row.CodModelo, &row.IDCotaReposicao,
	)
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (s *Store) SearchSolicitacoes(ctx context.Context, query, column, statusFilter, fromDate, toDate string, limit int) ([]SolicitacaoRecord, error) {
	if limit <= 0 {
		limit = 500
	}
	q := strings.TrimSpace(query)
	col := strings.TrimSpace(column)
	status := strings.ToLower(strings.TrimSpace(statusFilter))
	from := strings.TrimSpace(fromDate)
	to := strings.TrimSpace(toDate)
	const cols = `CAST(COALESCE(id, 0) AS INTEGER) AS id,
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
CAST(COALESCE(contemplation_bid, '') AS TEXT) AS contemplation_bid`

	whereParts := make([]string, 0, 2)
	args := make([]any, 0, 20)

	pendingCond := `( (status IS NULL OR TRIM(CAST(status AS TEXT)) = '' OR LOWER(TRIM(CAST(status AS TEXT))) NOT LIKE 'atendid%')
  AND (served_at IS NULL OR TRIM(CAST(served_at AS TEXT)) = '')
  AND (served_date IS NULL OR TRIM(CAST(served_date AS TEXT)) = '') )`
	attendedCond := `( (status IS NOT NULL AND LOWER(TRIM(CAST(status AS TEXT))) LIKE 'atendid%')
  OR (served_at IS NOT NULL AND TRIM(CAST(served_at AS TEXT)) <> '')
  OR (served_date IS NOT NULL AND TRIM(CAST(served_date AS TEXT)) <> '') )`

	switch status {
	case "", "pending", "pendente", "pendentes":
		whereParts = append(whereParts, pendingCond)
	case "attended", "atendida", "atendidas", "atendido", "atendidos":
		whereParts = append(whereParts, attendedCond)
	case "all", "todas", "todos":
		// sem filtro
	default:
		whereParts = append(whereParts, pendingCond)
	}

	baseDateExpr := `COALESCE(NULLIF(SUBSTR(TRIM(CAST(requested_date AS TEXT)), 1, 10), ''), SUBSTR(TRIM(CAST(requested_at AS TEXT)), 1, 10))`
	if from != "" {
		whereParts = append(whereParts, baseDateExpr+` >= ?`)
		args = append(args, from)
	}
	if to != "" {
		whereParts = append(whereParts, baseDateExpr+` <= ?`)
		args = append(args, to)
	}

	if q != "" {
		textLike := "LIKE"
		if s.driver == "pgx" {
			textLike = "ILIKE"
		}
		if col == "" {
			like := "%" + q + "%"
			whereParts = append(whereParts, `(CAST(id AS TEXT)=?
   OR CAST(COALESCE(branch, '') AS TEXT) ` + textLike + ` ?
   OR CAST(COALESCE(seller_name, '') AS TEXT) ` + textLike + ` ?
   OR CAST(COALESCE(cpf, '') AS TEXT) ` + textLike + ` ?
   OR CAST(COALESCE(model_name, '') AS TEXT) ` + textLike + ` ?
   OR CAST(COALESCE(licensed, '') AS TEXT) ` + textLike + ` ?
   OR CAST(installments AS TEXT) LIKE ?
   OR CAST(bid_percent AS TEXT) LIKE ?
   OR CAST(COALESCE(with_restriction, '') AS TEXT) ` + textLike + ` ?
   OR CAST(group_code AS TEXT) LIKE ?
   OR CAST(COALESCE(notes, '') AS TEXT) ` + textLike + ` ?
   OR CAST(served_group AS TEXT) LIKE ?
   OR CAST(requested_quota_id AS TEXT) LIKE ?
   OR CAST(COALESCE(quota_rd, '') AS TEXT) ` + textLike + ` ?
   OR CAST(COALESCE(status, '') AS TEXT) ` + textLike + ` ?)`)
			args = append(args, q, like, like, like, like, like, like, like, like, like, like, like, like, like, like)
		} else {
			numericCols := map[string]string{
				"id":                 "id",
				"installments":       "installments",
				"bid_percent":        "bid_percent",
				"group_code":         "group_code",
				"served_group":       "served_group",
				"requested_quota_id": "requested_quota_id",
			}
			textCols := map[string]string{
				"branch":           "branch",
				"seller_name":      "seller_name",
				"cpf":              "cpf",
				"model_name":       "model_name",
				"plan":             "licensed",
				"with_restriction": "with_restriction",
				"notes":            "notes",
				"quota_rd":         "quota_rd",
				"status":           "status",
			}
			if rawCol, ok := numericCols[col]; ok {
				n, convErr := parseInt64Safe(q)
				if convErr != nil {
					return []SolicitacaoRecord{}, nil
				}
				whereParts = append(whereParts, "CAST(COALESCE(NULLIF("+rawCol+", ''), '0') AS INTEGER)=?")
				args = append(args, n)
			} else if rawCol, ok := textCols[col]; ok {
				whereParts = append(whereParts, "CAST(COALESCE("+rawCol+", '') AS TEXT) "+textLike+" ?")
				args = append(args, "%"+q+"%")
			} else {
				return nil, fmt.Errorf("coluna de busca invalida")
			}
		}
	}

	querySQL := "SELECT " + cols + " FROM requests"
	if len(whereParts) > 0 {
		querySQL += " WHERE " + strings.Join(whereParts, " AND ")
	}
	querySQL += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.DB.QueryContext(ctx, s.bind(querySQL), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]SolicitacaoRecord, 0, limit)
	for rows.Next() {
		var r SolicitacaoRecord
		if err := rows.Scan(
			&r.ID, &r.DataHoraSolicitacao, &r.DataSolicitacao, &r.HoraSolicitacao, &r.Filial, &r.Vendedor, &r.CPF, &r.Modelo, &r.Plano, &r.QtdeParcelas,
			&r.PercLance, &r.ComRestricao, &r.Grupo, &r.Notes, &r.IDCota, &r.GrupoAtendido, &r.CotaRD,
			&r.DataHoraAtendimento, &r.DataAtendimento, &r.HoraAtendimento, &r.Situacao, &r.LanceContemplacao,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CountSolicitacoesByGrupoInPeriodo(ctx context.Context, fromDate, toDate string) (map[int64]int64, error) {
	ref := strings.TrimSpace(toDate)
	if ref == "" {
		ref = strings.TrimSpace(fromDate)
	}
	if ref == "" {
		ref = nowUTCMinus3().Format("2006-01-02")
	}
	refDate, err := time.Parse("2006-01-02", ref)
	if err != nil {
		return nil, fmt.Errorf("data de referencia invalida: %w", err)
	}
	refDate = refDate.In(utcMinus3Loc)

	// Regra da tela: base sempre no mes da Data Final.
	// Janela vai do vencimento do mes anterior ate o vencimento do mes da Data Final.
	refMonthStart := time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, utcMinus3Loc)
	prevMonthStart := refMonthStart.AddDate(0, -1, 0)
	refMonthEnd := refMonthStart.AddDate(0, 1, -1)
	broadFrom := prevMonthStart.Format("2006-01-02")
	broadTo := refMonthEnd.Format("2006-01-02")

	dueByGroup, err := s.loadActiveGroupDueDayMap(ctx)
	if err != nil {
		return nil, err
	}
	if len(dueByGroup) == 0 {
		return map[int64]int64{}, nil
	}

	baseDateExpr := `COALESCE(NULLIF(SUBSTR(TRIM(CAST(requested_date AS TEXT)), 1, 10), ''), SUBSTR(TRIM(CAST(requested_at AS TEXT)), 1, 10))`
	query := `SELECT CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) AS grupo_num, ` + baseDateExpr + ` AS dt
FROM requests
WHERE ` + baseDateExpr + ` >= ? AND ` + baseDateExpr + ` <= ?
  AND CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) > 0`

	rows, err := s.DB.QueryContext(ctx, s.bind(query), broadFrom, broadTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64]int64, 256)
	for rows.Next() {
		var g int64
		var dtRaw string
		if err := rows.Scan(&g, &dtRaw); err != nil {
			return nil, err
		}
		dueDay, ok := dueByGroup[g]
		if !ok || dueDay <= 0 {
			continue
		}
		dtRaw = strings.TrimSpace(dtRaw)
		if dtRaw == "" {
			continue
		}
		dt, err := time.Parse("2006-01-02", dtRaw)
		if err != nil {
			continue
		}
		dt = dt.In(utcMinus3Loc)
		start := dueDateForGroupMonth(refDate.Year(), refDate.Month()-1, dueDay)
		end := dueDateForGroupMonth(refDate.Year(), refDate.Month(), dueDay)
		if (dt.Equal(start) || dt.After(start)) && (dt.Equal(end) || dt.Before(end)) {
			out[g]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func dueDateForGroupMonth(year int, month time.Month, dueDay int64) time.Time {
	m := month
	y := year
	if m < time.January {
		m = time.December
		y--
	}
	if m > time.December {
		m = time.January
		y++
	}
	day := int(dueDay)
	if day < 1 {
		day = 1
	}
	lastDay := time.Date(y, m+1, 0, 0, 0, 0, 0, utcMinus3Loc).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(y, m, day, 0, 0, 0, 0, utcMinus3Loc)
}

func (s *Store) loadActiveGroupDueDayMap(ctx context.Context) (map[int64]int64, error) {
	query := `SELECT
CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) AS group_code,
CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER) AS due_day
FROM active_groups
WHERE CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) > 0
ORDER BY id DESC`
	rows, err := s.DB.QueryContext(ctx, s.bind(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]int64, 512)
	for rows.Next() {
		var g int64
		var d int64
		if err := rows.Scan(&g, &d); err != nil {
			return nil, err
		}
		if g <= 0 || d <= 0 {
			continue
		}
		if _, exists := out[g]; !exists {
			out[g] = d
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fallback: quando um group_code nao existir em active_groups,
	// tenta obter due_day da base operacional de cotas disponiveis.
	fallbackQ := `SELECT
CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) AS group_code,
MAX(CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER)) AS due_day
FROM available_group_ids
WHERE CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) > 0
GROUP BY CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER)`
	fRows, err := s.DB.QueryContext(ctx, s.bind(fallbackQ))
	if err != nil {
		return nil, err
	}
	defer fRows.Close()
	for fRows.Next() {
		var g int64
		var d int64
		if err := fRows.Scan(&g, &d); err != nil {
			return nil, err
		}
		if g <= 0 || d <= 0 {
			continue
		}
		if _, exists := out[g]; !exists {
			out[g] = d
		}
	}
	return out, fRows.Err()
}

func (s *Store) GetLastSolicitacaoByCPF(ctx context.Context, cpf string) (*SolicitacaoRecord, error) {
	cpf = strings.TrimSpace(cpf)
	if cpf == "" {
		return nil, fmt.Errorf("cpf vazio")
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT CAST(COALESCE(id, 0) AS INTEGER) AS id,
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
WHERE REPLACE(REPLACE(cpf, '.', ''), '-', '') = ?
ORDER BY id DESC
LIMIT 1`),
		cpf,
	)
	var r SolicitacaoRecord
	if err := row.Scan(
		&r.ID, &r.DataHoraSolicitacao, &r.DataSolicitacao, &r.HoraSolicitacao, &r.Filial, &r.Vendedor, &r.CPF, &r.Modelo, &r.Plano, &r.QtdeParcelas,
		&r.PercLance, &r.ComRestricao, &r.Grupo, &r.Notes, &r.IDCota, &r.GrupoAtendido, &r.CotaRD,
		&r.DataHoraAtendimento, &r.DataAtendimento, &r.HoraAtendimento, &r.Situacao, &r.LanceContemplacao,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) GetAvailableGroupInfoByCode(ctx context.Context, groupCode int) (*IDsGrupoDisponivelRecord, error) {
	const cols = `
CAST(COALESCE(id, 0) AS INTEGER) AS id,
CAST(COALESCE(group_api_id, 0) AS INTEGER) AS group_api_id,
CAST(products AS TEXT) AS products,
CAST(COALESCE(due_day, 0) AS INTEGER) AS due_day,
CAST(COALESCE(term_months, 0) AS INTEGER) AS term_months,
CAST(group_kind AS TEXT) AS group_kind,
CAST(COALESCE(group_code, 0) AS INTEGER) AS group_code,
CAST(COALESCE(quota, 0) AS INTEGER) AS quota,
CAST(COALESCE(r, 0) AS INTEGER) AS r,
CAST(COALESCE(d, 0) AS INTEGER) AS d,
CAST(COALESCE(booked, 0) AS INTEGER) AS booked,
CAST(created_at AS TEXT) AS created_at,
CAST(COALESCE(participants, 0) AS INTEGER) AS participants,
CAST(COALESCE(failed, 0) AS INTEGER) AS failed`

	query := `SELECT ` + cols + ` FROM available_group_ids 
              WHERE group_code = ? 
              AND (booked = 0 OR booked IS NULL) 
              AND (failed = 0 OR failed IS NULL)
              ORDER BY id DESC LIMIT 1`

	var rec IDsGrupoDisponivelRecord
	err := s.DB.QueryRowContext(ctx, s.bind(query), groupCode).Scan(
		&rec.ID, &rec.IDGrupo, &rec.Produto, &rec.Vencimento, &rec.Prazo,
		&rec.Tipo, &rec.Grupo, &rec.Cota, &rec.R, &rec.D,
		&rec.Booked, &rec.CreatedAt, &rec.Participantes, &rec.Failed,
	)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Store) GetSolicitacaoByID(ctx context.Context, id int64) (*SolicitacaoRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id invalido")
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT CAST(COALESCE(id, 0) AS INTEGER) AS id,
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
WHERE id=?`),
		id,
	)
	var r SolicitacaoRecord
	if err := row.Scan(
		&r.ID, &r.DataHoraSolicitacao, &r.DataSolicitacao, &r.HoraSolicitacao, &r.Filial, &r.Vendedor, &r.CPF, &r.Modelo, &r.Plano, &r.QtdeParcelas,
		&r.PercLance, &r.ComRestricao, &r.Grupo, &r.Notes, &r.IDCota, &r.GrupoAtendido, &r.CotaRD,
		&r.DataHoraAtendimento, &r.DataAtendimento, &r.HoraAtendimento, &r.Situacao, &r.LanceContemplacao,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) SaveSolicitacao(ctx context.Context, r SolicitacaoRecord) (int64, error) {
	relationIDs, err := s.resolveSolicitacaoRelationIDs(ctx, r)
	if err != nil {
		return 0, err
	}

	dataSolicitacao := r.DataSolicitacao
	horaSolicitacao := r.HoraSolicitacao
	if !dataSolicitacao.Valid || strings.TrimSpace(dataSolicitacao.String) == "" {
		d, h := splitDateTimeParts(r.DataHoraSolicitacao)
		dataSolicitacao = d
		if !horaSolicitacao.Valid || strings.TrimSpace(horaSolicitacao.String) == "" {
			horaSolicitacao = h
		}
	}

	dataAtendimento := r.DataAtendimento
	horaAtendimento := r.HoraAtendimento
	if !dataAtendimento.Valid || strings.TrimSpace(dataAtendimento.String) == "" {
		d, h := splitDateTimeParts(r.DataHoraAtendimento)
		dataAtendimento = d
		if !horaAtendimento.Valid || strings.TrimSpace(horaAtendimento.String) == "" {
			horaAtendimento = h
		}
	}

	if r.ID > 0 {
		_, err = s.DB.ExecContext(
			ctx,
			s.bind(`UPDATE requests
SET requested_at=?,
    requester_user_id=?,
    vendor_identity_id=?,
    api_account_id=?,
    requested_date=?,
    requested_time=?,
    branch=?,
    seller_name=?,
    cpf=?,
    model_name=?,
    licensed=?,
    installments=?,
    bid_percent=?,
    with_restriction=?,
    group_code=?,
    notes=?,
    requested_quota_id=?,
    served_group=?,
    quota_rd=?,
    served_at=?,
    served_date=?,
    served_time=?,
    status=?,
    contemplation_bid=?
WHERE id=?`),
			nullStringArg(r.DataHoraSolicitacao),
			nullIntArg(relationIDs.RequesterUserID),
			nullIntArg(relationIDs.VendorIdentityID),
			nullIntArg(relationIDs.APIAccountID),
			nullStringArg(dataSolicitacao),
			nullStringArg(horaSolicitacao),
			strings.TrimSpace(r.Filial),
			strings.TrimSpace(r.Vendedor),
			strings.TrimSpace(r.CPF),
			strings.TrimSpace(r.Modelo),
			strings.TrimSpace(r.Plano),
			nullIntArg(r.QtdeParcelas),
			nullFloatArg(r.PercLance),
			strings.TrimSpace(r.ComRestricao),
			nullIntArg(r.Grupo),
			strings.TrimSpace(r.Notes),
			nullIntArg(r.IDCota),
			nullIntArg(r.GrupoAtendido),
			strings.TrimSpace(r.CotaRD),
			nullStringArg(r.DataHoraAtendimento),
			nullStringArg(dataAtendimento),
			nullStringArg(horaAtendimento),
			strings.TrimSpace(r.Situacao),
			strings.TrimSpace(r.LanceContemplacao),
			r.ID,
		)
		if err != nil {
			return 0, err
		}
		return r.ID, nil
	}

	insertArgs := []any{
		nullStringArg(r.DataHoraSolicitacao),
		nullIntArg(relationIDs.RequesterUserID),
		nullIntArg(relationIDs.VendorIdentityID),
		nullIntArg(relationIDs.APIAccountID),
		nullStringArg(dataSolicitacao),
		nullStringArg(horaSolicitacao),
		strings.TrimSpace(r.Filial),
		strings.TrimSpace(r.Vendedor),
		strings.TrimSpace(r.CPF),
		strings.TrimSpace(r.Modelo),
		strings.TrimSpace(r.Plano),
		nullIntArg(r.QtdeParcelas),
		nullFloatArg(r.PercLance),
		strings.TrimSpace(r.ComRestricao),
		nullIntArg(r.Grupo),
		strings.TrimSpace(r.Notes),
		nullIntArg(r.IDCota),
		nullIntArg(r.GrupoAtendido),
		strings.TrimSpace(r.CotaRD),
		nullStringArg(r.DataHoraAtendimento),
		nullStringArg(dataAtendimento),
		nullStringArg(horaAtendimento),
		strings.TrimSpace(r.Situacao),
		strings.TrimSpace(r.LanceContemplacao),
	}
	if s.driver == "pgx" {
		var id int64
		err = s.DB.QueryRowContext(
			ctx,
			s.bind(`INSERT INTO requests
 (requested_at, requester_user_id, vendor_identity_id, api_account_id, requested_date, requested_time, branch, seller_name, cpf, model_name, licensed, installments, bid_percent, with_restriction, group_code, notes, requested_quota_id, served_group, quota_rd, served_at, served_date, served_time, status, contemplation_bid)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
			insertArgs...,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		`INSERT INTO requests
 (requested_at, requester_user_id, vendor_identity_id, api_account_id, requested_date, requested_time, branch, seller_name, cpf, model_name, licensed, installments, bid_percent, with_restriction, group_code, notes, requested_quota_id, served_group, quota_rd, served_at, served_date, served_time, status, contemplation_bid)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		insertArgs...,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) getCurrentSolicitacaoRelationIDs(ctx context.Context, requestID int64) (requestRelationIDs, error) {
	var out requestRelationIDs
	if requestID <= 0 {
		return out, nil
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT requester_user_id, vendor_identity_id, api_account_id
		   FROM requests
		  WHERE id=?`),
		requestID,
	)
	if err := row.Scan(&out.RequesterUserID, &out.VendorIdentityID, &out.APIAccountID); err != nil {
		if err == sql.ErrNoRows {
			return requestRelationIDs{}, nil
		}
		return requestRelationIDs{}, err
	}
	return out, nil
}

func (s *Store) resolveSolicitacaoRelationIDs(ctx context.Context, r SolicitacaoRecord) (requestRelationIDs, error) {
	var out requestRelationIDs
	cpf := strings.TrimSpace(r.CPF)
	branch := strings.TrimSpace(r.Filial)
	seller := strings.TrimSpace(r.Vendedor)

	userID, err := s.resolveRequesterUserID(ctx, cpf, branch)
	if err != nil {
		return out, err
	}
	out.RequesterUserID = userID

	apiID, err := s.resolveAPIAccountID(ctx, cpf)
	if err != nil {
		return out, err
	}
	out.APIAccountID = apiID

	vendorID, err := s.resolveVendorIdentityID(ctx, userID, cpf, seller, branch)
	if err != nil {
		return out, err
	}
	out.VendorIdentityID = vendorID

	return out, nil
}

func (s *Store) resolveRequesterUserID(ctx context.Context, cpf, branch string) (sql.NullInt64, error) {
	cpf = strings.TrimSpace(cpf)
	branch = strings.TrimSpace(branch)
	if cpf == "" {
		return sql.NullInt64{}, nil
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT id
		   FROM users
		  WHERE is_active=1
		    AND TRIM(COALESCE(cpf,''))=?
		  ORDER BY
		    CASE WHEN LOWER(TRIM(COALESCE(role,''))) IN ('vendedor','seller_name') THEN 0 ELSE 1 END,
		    CASE WHEN ? <> '' AND LOWER(TRIM(COALESCE(branch,''))) = LOWER(TRIM(?)) THEN 0 ELSE 1 END,
		    id ASC
		  LIMIT 1`),
		cpf, branch, branch,
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return sql.NullInt64{}, nil
		}
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: id, Valid: true}, nil
}

func (s *Store) resolveAPIAccountID(ctx context.Context, cpf string) (sql.NullInt64, error) {
	cpf = strings.TrimSpace(cpf)
	if cpf == "" {
		return sql.NullInt64{}, nil
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT id
		   FROM api_accounts
		  WHERE TRIM(COALESCE(cpf,''))=?
		  ORDER BY CASE WHEN token IS NOT NULL AND TRIM(token)<>'' THEN 0 ELSE 1 END, id ASC
		  LIMIT 1`),
		cpf,
	)
	var id int64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return sql.NullInt64{}, nil
		}
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: id, Valid: true}, nil
}

func (s *Store) resolveVendorIdentityID(ctx context.Context, requesterUserID sql.NullInt64, cpf, seller, branch string) (sql.NullInt64, error) {
	seller = strings.TrimSpace(seller)
	branch = strings.TrimSpace(branch)
	cpf = strings.TrimSpace(cpf)
	if seller == "" {
		return sql.NullInt64{}, nil
	}

	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT id, user_id
		   FROM vendor_identity_map
		  WHERE LOWER(TRIM(seller_name)) = LOWER(TRIM(?))
		    AND COALESCE(LOWER(TRIM(branch)), '') = COALESCE(LOWER(TRIM(?)), '')
		    AND COALESCE(TRIM(cpf), '') = COALESCE(TRIM(?), '')
		  ORDER BY is_active DESC, id ASC
		  LIMIT 1`),
		seller, branch, cpf,
	)
	var id int64
	var currentUserID sql.NullInt64
	if err := row.Scan(&id, &currentUserID); err == nil {
		if requesterUserID.Valid && (!currentUserID.Valid || currentUserID.Int64 != requesterUserID.Int64) {
			_, _ = s.DB.ExecContext(
				ctx,
				s.bind(`UPDATE vendor_identity_map
				    SET user_id=?, updated_at=?
				  WHERE id=?`),
				requesterUserID.Int64, formatDateTimeUTCMinus3(nowUTCMinus3()), id,
			)
		}
		return sql.NullInt64{Int64: id, Valid: true}, nil
	} else if err != sql.ErrNoRows {
		return sql.NullInt64{}, err
	}

	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	if s.driver == "pgx" {
		var newID int64
		err := s.DB.QueryRowContext(
			ctx,
			s.bind(`INSERT INTO vendor_identity_map (user_id, cpf, seller_name, branch, source, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'requests', 1, ?, ?) RETURNING id`),
			nullIntArg(requesterUserID),
			nullStringArg(sql.NullString{String: cpf, Valid: cpf != ""}),
			seller,
			nullStringArg(sql.NullString{String: branch, Valid: branch != ""}),
			now,
			now,
		).Scan(&newID)
		if err != nil {
			return sql.NullInt64{}, err
		}
		return sql.NullInt64{Int64: newID, Valid: true}, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		`INSERT INTO vendor_identity_map (user_id, cpf, seller_name, branch, source, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'requests', 1, ?, ?)`,
		nullIntArg(requesterUserID),
		nullStringArg(sql.NullString{String: cpf, Valid: cpf != ""}),
		seller,
		nullStringArg(sql.NullString{String: branch, Valid: branch != ""}),
		now,
		now,
	)
	if err != nil {
		return sql.NullInt64{}, err
	}
	newID, err := res.LastInsertId()
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: newID, Valid: true}, nil
}

func (s *Store) DeleteSolicitacao(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, s.bind("DELETE FROM reservations WHERE request_id=?"), id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, s.bind("DELETE FROM manual_notifications WHERE request_id=?"), id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, s.bind("DELETE FROM requests WHERE id=?"), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) FindBookingUserByCPF(ctx context.Context, cpf string) (*User, error) {
	cleanCPF := strings.TrimSpace(cpf)
	if cleanCPF == "" {
		return nil, sql.ErrNoRows
	}
	const cols = `id, cpf, company_code, account_user, account_password, COALESCE(token, '') AS token, last_request_at, cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score`

	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT `+cols+` FROM api_accounts
WHERE cpf=? AND token IS NOT NULL AND token <> ''
ORDER BY id ASC
LIMIT 1`),
		cleanCPF,
	)
	var u User
	if err := row.Scan(
		&u.ID, &u.CPF, &u.CodEmpresa, &u.CodUsuario, &u.Senha, &u.Token, &u.LastRequest,
		&u.CooldownUntil, &u.InFlight, &u.Error401Count, &u.Error429Count, &u.BlockedUntil, &u.PriorityScore,
	); err != nil {
		return nil, err
	}
	if err := revealUserPassword(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) FindAvailableCotaForSolicitacao(ctx context.Context, group_code, idCota, quota, r, d string) (*Cota, error) {
	group_code = strings.TrimSpace(group_code)
	idCota = strings.TrimSpace(idCota)
	quota = strings.TrimSpace(quota)
	r = strings.TrimSpace(r)
	d = strings.TrimSpace(d)

	base := `
SELECT
  CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER) AS id,
  CAST(COALESCE(NULLIF(CAST(group_api_id AS TEXT), ''), '0') AS INTEGER) AS group_api_id,
  CAST(products AS TEXT) AS products,
  CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER) AS due_day,
  CAST(group_kind AS TEXT) AS group_kind,
  CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) AS group_code,
  CAST(COALESCE(NULLIF(CAST(quota AS TEXT), ''), '0') AS INTEGER) AS quota,
  CAST(COALESCE(NULLIF(CAST(r AS TEXT), ''), '0') AS INTEGER) AS r,
  CAST(COALESCE(NULLIF(CAST(d AS TEXT), ''), '0') AS INTEGER) AS d,
  CAST(COALESCE(NULLIF(CAST(participants AS TEXT), ''), '0') AS INTEGER) AS participants
FROM available_group_ids
WHERE booked = 0 AND failed = 0
`
	args := []any{}
	if group_code != "" {
		base += " AND CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS TEXT) = ?"
		args = append(args, group_code)
	}
	if idCota != "" {
		base += " AND (CAST(COALESCE(NULLIF(CAST(group_api_id AS TEXT), ''), '0') AS TEXT) = ? OR CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS TEXT) = ?)"
		args = append(args, idCota, idCota)
	}
	if quota != "" {
		base += " AND CAST(COALESCE(NULLIF(CAST(quota AS TEXT), ''), '0') AS TEXT) = ?"
		args = append(args, quota)
	}
	if r != "" {
		base += " AND CAST(COALESCE(NULLIF(CAST(r AS TEXT), ''), '0') AS TEXT) = ?"
		args = append(args, r)
	}
	if d != "" {
		base += " AND CAST(COALESCE(NULLIF(CAST(d AS TEXT), ''), '0') AS TEXT) = ?"
		args = append(args, d)
	}
	if idCota == "" {
		// Sem ID Cota informado: prioriza o maior group_api_id disponÃƒÆ’Ã‚Â­vel (mais recente).
		base += " ORDER BY CAST(COALESCE(NULLIF(CAST(group_api_id AS TEXT), ''), '0') AS INTEGER) DESC, id DESC LIMIT 1"
	} else {
		// Com ID Cota informado: mantÃƒÆ’Ã‚Â©m a seleÃƒÆ’Ã‚Â§ÃƒÆ’Ã‚Â£o estÃƒÆ’Ã‚Â¡vel como jÃƒÆ’Ã‚Â¡ estava.
		base += " ORDER BY id ASC LIMIT 1"
	}

	row := s.DB.QueryRowContext(ctx, s.bind(base), args...)
	var c Cota
	if err := row.Scan(
		&c.ID, &c.IDGrupo, &c.Produto, &c.Vencimento, &c.Tipo, &c.Grupo, &c.Cota, &c.R, &c.D, &c.Participantes,
	); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) MarkSolicitacaoAtendidaByID(
	ctx context.Context,
	id int64,
	idCota sql.NullInt64,
	grupoAtendido sql.NullInt64,
	quota_rd string,
	dataHoraAtendimento string,
	status string,
) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	d, h := splitDateTimeParts(sql.NullString{String: strings.TrimSpace(dataHoraAtendimento), Valid: strings.TrimSpace(dataHoraAtendimento) != ""})
	res, err := s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE requests
SET requested_quota_id=?,
    served_group=?,
    quota_rd=?,
    served_at=?,
    served_date=?,
    served_time=?,
    status=?
WHERE id=?`),
		nullIntArg(idCota),
		nullIntArg(grupoAtendido),
		strings.TrimSpace(quota_rd),
		nullStringArg(sql.NullString{String: strings.TrimSpace(dataHoraAtendimento), Valid: strings.TrimSpace(dataHoraAtendimento) != ""}),
		nullStringArg(d),
		nullStringArg(h),
		strings.TrimSpace(status),
		id,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) FindIDModeloByNome(ctx context.Context, nomeModelo string) (int64, error) {
	model_name := strings.TrimSpace(nomeModelo)
	if model_name == "" {
		return 0, sql.ErrNoRows
	}

	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT CAST(COALESCE(NULLIF(CAST(model_api_id AS TEXT), ''), '0') AS INTEGER)
FROM models
WHERE LOWER(TRIM(COALESCE(model_name, ''))) = LOWER(TRIM(?))
ORDER BY
  CASE WHEN LOWER(TRIM(COALESCE(status, ''))) = 'is_active' THEN 0 ELSE 1 END,
  id DESC
LIMIT 1`),
		model_name,
	)

	var model_api_id int64
	if err := row.Scan(&model_api_id); err != nil {
		return 0, err
	}
	if model_api_id <= 0 {
		return 0, sql.ErrNoRows
	}
	return model_api_id, nil
}

func (s *Store) FindModeloNomeByIDModelo(ctx context.Context, model_api_id int64) (string, error) {
	if model_api_id <= 0 {
		return "", sql.ErrNoRows
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT CAST(COALESCE(model_name, '') AS TEXT)
FROM models
WHERE CAST(COALESCE(NULLIF(CAST(model_api_id AS TEXT), ''), '0') AS INTEGER) = ?
ORDER BY
  CASE WHEN LOWER(TRIM(COALESCE(status, ''))) = 'is_active' THEN 0 ELSE 1 END,
  id DESC
LIMIT 1`),
		model_api_id,
	)
	var nome string
	if err := row.Scan(&nome); err != nil {
		return "", err
	}
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", sql.ErrNoRows
	}
	return nome, nil
}

func (s *Store) DeleteReservedCota(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM reservations WHERE id=?"), id)
	return err
}

func (s *Store) DeleteReservedCotasByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			args = append(args, id)
		}
	}
	if len(args) == 0 {
		return 0, nil
	}

	res, err := s.DB.ExecContext(
		ctx,
		s.bind("DELETE FROM reservations WHERE id IN ("+placeholders(len(args))+")"),
		args...,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) InsertManualNotifications(ctx context.Context, rows []ManualNotificationRecord) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	inserted := int64(0)
	for _, row := range rows {
		msg := strings.TrimSpace(row.Mensagem)
		if msg == "" {
			continue
		}
		channel := strings.TrimSpace(row.Canal)
		if channel == "" {
			channel = "whatsapp"
		}
		status := strings.TrimSpace(row.Status)
		if status == "" {
			status = "pendente"
		}
		var sid any = nil
		if row.SolicitacaoID.Valid && row.SolicitacaoID.Int64 > 0 {
			sid = row.SolicitacaoID.Int64
		}
		_, err := tx.ExecContext(
			ctx,
			s.bind(`INSERT INTO manual_notifications
 (request_id, cpf, seller_name, branch, channel, message, status, action_user, created_at, updated_at)
 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
			sid,
			strings.TrimSpace(row.CPF),
			strings.TrimSpace(row.Vendedor),
			strings.TrimSpace(row.Filial),
			channel,
			msg,
			status,
			strings.TrimSpace(row.UsuarioAcao),
			now,
			now,
		)
		if err != nil {
			return inserted, err
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

func (s *Store) SearchManualNotifications(ctx context.Context, query, status, fromDate, toDate string, solicitacaoID int64, cpfOnly string, limit int) ([]ManualNotificationRecord, error) {
	if limit <= 0 {
		limit = 500
	}
	q := strings.TrimSpace(query)
	st := strings.ToLower(strings.TrimSpace(status))
	from := strings.TrimSpace(fromDate)
	to := strings.TrimSpace(toDate)
	cpf := strings.TrimSpace(cpfOnly)

	const cols = `id, request_id, cpf, seller_name, branch, channel, message, status, copied_at, sent_at, CAST(COALESCE(action_user, '') AS TEXT) AS action_user, created_at, updated_at`
	whereParts := make([]string, 0, 8)
	args := make([]any, 0, 16)

	baseDateExpr := `SUBSTR(TRIM(CAST(created_at AS TEXT)), 1, 10)`
	if from != "" {
		whereParts = append(whereParts, baseDateExpr+` >= ?`)
		args = append(args, from)
	}
	if to != "" {
		whereParts = append(whereParts, baseDateExpr+` <= ?`)
		args = append(args, to)
	}
	if solicitacaoID > 0 {
		whereParts = append(whereParts, `CAST(COALESCE(request_id, 0) AS INTEGER)=?`)
		args = append(args, solicitacaoID)
	}
	if cpf != "" {
		whereParts = append(whereParts, `CAST(COALESCE(cpf, '') AS TEXT)=?`)
		args = append(args, cpf)
	}
	switch st {
	case "", "all", "todas", "todos":
		// sem filtro
	default:
		whereParts = append(whereParts, `LOWER(TRIM(CAST(COALESCE(status,'') AS TEXT)))=?`)
		args = append(args, st)
	}

	if q != "" {
		like := "%" + q + "%"
		likeOp := "LIKE"
		if s.driver == "pgx" {
			likeOp = "ILIKE"
		}
		whereParts = append(whereParts, `(CAST(id AS TEXT)=?
 OR CAST(COALESCE(request_id,'') AS TEXT)=?
 OR CAST(COALESCE(cpf,'') AS TEXT) `+likeOp+` ?
 OR CAST(COALESCE(seller_name,'') AS TEXT) `+likeOp+` ?
 OR CAST(COALESCE(branch,'') AS TEXT) `+likeOp+` ?
 OR CAST(COALESCE(channel,'') AS TEXT) `+likeOp+` ?
 OR CAST(COALESCE(status,'') AS TEXT) `+likeOp+` ?
 OR CAST(COALESCE(message,'') AS TEXT) `+likeOp+` ?)`)
		args = append(args, q, q, like, like, like, like, like, like)
	}

	querySQL := "SELECT " + cols + " FROM manual_notifications"
	if len(whereParts) > 0 {
		querySQL += " WHERE " + strings.Join(whereParts, " AND ")
	}
	querySQL += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.DB.QueryContext(ctx, s.bind(querySQL), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ManualNotificationRecord, 0, limit)
	for rows.Next() {
		var r ManualNotificationRecord
		if err := rows.Scan(
			&r.ID, &r.SolicitacaoID, &r.CPF, &r.Vendedor, &r.Filial, &r.Canal, &r.Mensagem, &r.Status,
			&r.CopiadaEm, &r.EnviadaEm, &r.UsuarioAcao, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) MarkManualNotificationsStatus(ctx context.Context, ids []int64, status, usuarioAcao string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	st := strings.ToLower(strings.TrimSpace(status))
	if st == "" {
		return 0, fmt.Errorf("status obrigatorio")
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			args = append(args, id)
		}
	}
	if len(args) == 0 {
		return 0, nil
	}
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	setCopied := st == "copiada"
	setSent := st == "enviada_manual"

	sqlUpdate := `UPDATE manual_notifications
SET status=?,
    action_user=?,
    copied_at=CASE WHEN ? THEN ? ELSE copied_at END,
    sent_at=CASE WHEN ? THEN ? ELSE sent_at END,
    updated_at=?
WHERE id IN (` + placeholders(len(args)) + `)`
	params := []any{st, strings.TrimSpace(usuarioAcao), setCopied, now, setSent, now, now}
	params = append(params, args...)
	res, err := s.DB.ExecContext(ctx, s.bind(sqlUpdate), params...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) SearchIDsGruposDisponiveis(ctx context.Context, query, column string, offset, limit int) ([]IDsGrupoDisponivelRecord, int64, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	q := strings.TrimSpace(query)
	col := strings.TrimSpace(column)
	const cols = `
CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER) AS id,
CAST(COALESCE(NULLIF(CAST(group_api_id AS TEXT), ''), '0') AS INTEGER) AS group_api_id,
CAST(products AS TEXT) AS products,
CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER) AS due_day,
CAST(COALESCE(NULLIF(CAST(term_months AS TEXT), ''), '0') AS INTEGER) AS term_months,
CAST(group_kind AS TEXT) AS group_kind,
CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) AS group_code,
CAST(COALESCE(NULLIF(CAST(quota AS TEXT), ''), '0') AS INTEGER) AS quota,
CAST(COALESCE(NULLIF(CAST(r AS TEXT), ''), '0') AS INTEGER) AS r,
CAST(COALESCE(NULLIF(CAST(d AS TEXT), ''), '0') AS INTEGER) AS d,
CAST(COALESCE(NULLIF(CAST(booked AS TEXT), ''), '0') AS INTEGER) AS booked,
CAST(created_at AS TEXT) AS created_at,
CAST(COALESCE(NULLIF(CAST(participants AS TEXT), ''), '0') AS INTEGER) AS participants,
CAST(COALESCE(NULLIF(CAST(failed AS TEXT), ''), '0') AS INTEGER) AS failed`

	whereSQL := ""
	whereArgs := []any{}
	if q != "" {
		textLike := "LIKE"
		if s.driver == "pgx" {
			textLike = "ILIKE"
		}
		if col == "" {
			like := "%" + q + "%"
			whereSQL = `
WHERE CAST(id AS TEXT)=?
   OR CAST(group_api_id AS TEXT) LIKE ?
   OR CAST(products AS TEXT) ` + textLike + ` ?
   OR CAST(due_day AS TEXT) LIKE ?
   OR CAST(term_months AS TEXT) LIKE ?
   OR CAST(group_kind AS TEXT) ` + textLike + ` ?
   OR CAST(group_code AS TEXT) LIKE ?
   OR CAST(quota AS TEXT) LIKE ?
   OR CAST(r AS TEXT) LIKE ?
   OR CAST(d AS TEXT) LIKE ?
   OR CAST(booked AS TEXT) LIKE ?
   OR CAST(created_at AS TEXT) LIKE ?
   OR CAST(participants AS TEXT) LIKE ?
   OR CAST(failed AS TEXT) LIKE ?`
			whereArgs = []any{q, like, like, like, like, like, like, like, like, like, like, like, like, like}
		} else {
			numericCols := map[string]string{
				"id":           "id",
				"id_grupo":     "group_api_id",
				"due_day":      "due_day",
				"term_months":  "term_months",
				"group_code":   "group_code",
				"quota":        "quota",
				"r":            "r",
				"d":            "d",
				"booked":       "booked",
				"participants": "participants",
				"failed":       "failed",
			}
			textCols := map[string]string{
				"products":   "products",
				"group_kind": "group_kind",
				"created_at": "created_at",
			}
			if rawCol, ok := numericCols[col]; ok {
				n, convErr := parseInt64Safe(q)
				if convErr != nil {
					return []IDsGrupoDisponivelRecord{}, 0, nil
				}
				whereSQL = "WHERE CAST(COALESCE(" + rawCol + ", 0) AS INTEGER)=?"
				whereArgs = []any{n}
			} else if rawCol, ok := textCols[col]; ok {
				whereSQL = "WHERE CAST(COALESCE(" + rawCol + ", '') AS TEXT) " + textLike + " ?"
				whereArgs = []any{"%" + q + "%"}
			} else {
				return nil, 0, fmt.Errorf("coluna de busca invÃƒÆ’Ã‚Â¡lida")
			}
		}
	}

	var total int64
	countSQL := "SELECT COUNT(1) FROM available_group_ids " + whereSQL
	if err := s.DB.QueryRowContext(ctx, s.bind(countSQL), whereArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	querySQL := "SELECT " + cols + " FROM available_group_ids " + whereSQL + " ORDER BY id DESC LIMIT ? OFFSET ?"
	queryArgs := append([]any{}, whereArgs...)
	queryArgs = append(queryArgs, limit, offset)

	rows, err := s.DB.QueryContext(ctx, s.bind(querySQL), queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]IDsGrupoDisponivelRecord, 0, limit)
	for rows.Next() {
		var r IDsGrupoDisponivelRecord
		if err := rows.Scan(
			&r.ID,
			&r.IDGrupo,
			&r.Produto,
			&r.Vencimento,
			&r.Prazo,
			&r.Tipo,
			&r.Grupo,
			&r.Cota,
			&r.R,
			&r.D,
			&r.Booked,
			&r.CreatedAt,
			&r.Participantes,
			&r.Failed,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func parseInt64Safe(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	sign := int64(1)
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		sign = -1
		s = s[1:]
	}
	if s == "" {
		return 0, fmt.Errorf("invalid")
	}
	var n int64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid")
		}
		n = n*10 + int64(ch-'0')
	}
	return sign * n, nil
}

func nullStringOrNil(v sql.NullString) any {
	if v.Valid && strings.TrimSpace(v.String) != "" {
		return strings.TrimSpace(v.String)
	}
	return nil
}

func nullInt64OrNil(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}

func nullFloat64OrNil(v sql.NullFloat64) any {
	if v.Valid {
		return v.Float64
	}
	return nil
}

func (s *Store) GetIDsGrupoDisponivelByID(ctx context.Context, id int64) (*IDsGrupoDisponivelRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id invalido")
	}
	const cols = `
CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER) AS id,
CAST(COALESCE(NULLIF(CAST(group_api_id AS TEXT), ''), '0') AS INTEGER) AS group_api_id,
CAST(products AS TEXT) AS products,
CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER) AS due_day,
CAST(COALESCE(NULLIF(CAST(term_months AS TEXT), ''), '0') AS INTEGER) AS term_months,
CAST(group_kind AS TEXT) AS group_kind,
CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) AS group_code,
CAST(COALESCE(NULLIF(CAST(quota AS TEXT), ''), '0') AS INTEGER) AS quota,
CAST(COALESCE(NULLIF(CAST(r AS TEXT), ''), '0') AS INTEGER) AS r,
CAST(COALESCE(NULLIF(CAST(d AS TEXT), ''), '0') AS INTEGER) AS d,
CAST(COALESCE(NULLIF(CAST(booked AS TEXT), ''), '0') AS INTEGER) AS booked,
CAST(created_at AS TEXT) AS created_at,
CAST(COALESCE(NULLIF(CAST(participants AS TEXT), ''), '0') AS INTEGER) AS participants,
CAST(COALESCE(NULLIF(CAST(failed AS TEXT), ''), '0') AS INTEGER) AS failed`
	row := s.DB.QueryRowContext(ctx, s.bind("SELECT "+cols+" FROM available_group_ids WHERE id=?"), id)
	var r IDsGrupoDisponivelRecord
	if err := row.Scan(
		&r.ID,
		&r.IDGrupo,
		&r.Produto,
		&r.Vencimento,
		&r.Prazo,
		&r.Tipo,
		&r.Grupo,
		&r.Cota,
		&r.R,
		&r.D,
		&r.Booked,
		&r.CreatedAt,
		&r.Participantes,
		&r.Failed,
	); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) UpdateIDsGrupoDisponivel(ctx context.Context, r IDsGrupoDisponivelRecord) error {
	if r.ID <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE available_group_ids
		 SET group_api_id=?,
		     products=?,
		     due_day=?,
		     term_months=?,
		     group_kind=?,
		     group_code=?,
		     quota=?,
		     r=?,
		     d=?,
		     booked=?,
		     created_at=?,
		     participants=?,
		     failed=?
		 WHERE id=?`),
		r.IDGrupo,
		strings.TrimSpace(r.Produto),
		r.Vencimento,
		r.Prazo,
		strings.TrimSpace(r.Tipo),
		r.Grupo,
		r.Cota,
		r.R,
		r.D,
		r.Booked,
		r.CreatedAt.String,
		r.Participantes,
		r.Failed,
		r.ID,
	)
	return err
}

func (s *Store) InsertIDsGrupoDisponivel(ctx context.Context, r IDsGrupoDisponivelRecord) (int64, error) {
	if s.driver == "pgx" {
		var id int64
		err := s.DB.QueryRowContext(
			ctx,
			s.bind(`INSERT INTO available_group_ids
		 (group_api_id, products, due_day, term_months, group_kind, group_code, quota, r, d, booked, created_at, participants, failed)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
			r.IDGrupo,
			strings.TrimSpace(r.Produto),
			r.Vencimento,
			r.Prazo,
			strings.TrimSpace(r.Tipo),
			r.Grupo,
			r.Cota,
			r.R,
			r.D,
			r.Booked,
			r.CreatedAt.String,
			r.Participantes,
			r.Failed,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		s.bind(`INSERT INTO available_group_ids
		 (group_api_id, products, due_day, term_months, group_kind, group_code, quota, r, d, booked, created_at, participants, failed)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		r.IDGrupo,
		strings.TrimSpace(r.Produto),
		r.Vencimento,
		r.Prazo,
		strings.TrimSpace(r.Tipo),
		r.Grupo,
		r.Cota,
		r.R,
		r.D,
		r.Booked,
		r.CreatedAt.String,
		r.Participantes,
		r.Failed,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteIDsGrupoDisponivel(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM available_group_ids WHERE id=?"), id)
	return err
}

func (s *Store) DeleteIDsGruposDisponiveisByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			args = append(args, id)
		}
	}
	if len(args) == 0 {
		return 0, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		s.bind("DELETE FROM available_group_ids WHERE id IN ("+placeholders(len(args))+")"),
		args...,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) SearchModelos(ctx context.Context, query string, limit int) ([]ModeloRecord, error) {
	if limit <= 0 {
		limit = 500
	}
	q := strings.TrimSpace(query)
	const cols = `CAST(COALESCE(id, 0) AS INTEGER) AS id,
CAST(COALESCE(model_api_id, 0) AS INTEGER) AS model_api_id,
CAST(COALESCE(model_name, '') AS TEXT) AS model_name,
CAST(COALESCE(status, '') AS TEXT) AS status`

	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = s.DB.QueryContext(ctx, s.bind("SELECT "+cols+" FROM models ORDER BY id DESC LIMIT ?"), limit)
	} else {
		like := "%" + q + "%"
		likeOp := "LIKE"
		if s.driver == "pgx" {
			likeOp = "ILIKE"
		}
		rows, err = s.DB.QueryContext(
			ctx,
			s.bind(`SELECT `+cols+` FROM models
WHERE CAST(id AS TEXT)=?
   OR CAST(model_api_id AS TEXT) LIKE ?
   OR CAST(COALESCE(model_name, '') AS TEXT) `+likeOp+` ?
   OR CAST(COALESCE(status, '') AS TEXT) `+likeOp+` ?
ORDER BY id DESC
LIMIT ?`),
			q, like, like, like, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ModeloRecord, 0, limit)
	for rows.Next() {
		var r ModeloRecord
		if err := rows.Scan(&r.ID, &r.IDModelo, &r.Modelo, &r.Status); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetModeloByID(ctx context.Context, id int64) (*ModeloRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id invalido")
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT CAST(COALESCE(id, 0) AS INTEGER),
        CAST(COALESCE(model_api_id, 0) AS INTEGER),
        CAST(COALESCE(model_name, '') AS TEXT),
        CAST(COALESCE(status, '') AS TEXT)
   FROM models
  WHERE id=?`),
		id,
	)
	var r ModeloRecord
	if err := row.Scan(&r.ID, &r.IDModelo, &r.Modelo, &r.Status); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) SaveModelo(ctx context.Context, r ModeloRecord) (int64, error) {
	model_name := strings.TrimSpace(r.Modelo)
	if r.IDModelo <= 0 || model_name == "" {
		return 0, fmt.Errorf("model_api_id e model_name obrigatorios")
	}
	status := strings.TrimSpace(r.Status)
	if status == "" {
		status = "is_active"
	}

	if r.ID > 0 {
		_, err := s.DB.ExecContext(
			ctx,
			s.bind(`UPDATE models
SET model_api_id=?, model_name=?, status=?
WHERE id=?`),
			r.IDModelo, model_name, status, r.ID,
		)
		if err != nil {
			return 0, err
		}
		return r.ID, nil
	}

	if s.driver == "pgx" {
		var id int64
		err := s.DB.QueryRowContext(
			ctx,
			s.bind(`INSERT INTO models (model_api_id, model_name, status)
VALUES (?, ?, ?) RETURNING id`),
			r.IDModelo, model_name, status,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		s.bind(`INSERT INTO models (model_api_id, model_name, status)
VALUES (?, ?, ?)`),
		r.IDModelo, model_name, status,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteModelo(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM models WHERE id=?"), id)
	return err
}

func (s *Store) SearchProdutos(ctx context.Context, query string, limit int) ([]ProdutoRecord, error) {
	if limit <= 0 {
		limit = 500
	}
	q := strings.TrimSpace(query)
	const cols = `CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER) AS id,
CAST(COALESCE(NULLIF(CAST(product_api_id AS TEXT), ''), '0') AS INTEGER) AS product_api_id,
CAST(COALESCE(products, '') AS TEXT) AS products,
CAST(COALESCE(status, '') AS TEXT) AS status`

	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = s.DB.QueryContext(ctx, s.bind("SELECT "+cols+" FROM products ORDER BY id DESC LIMIT ?"), limit)
	} else {
		textLike := "LIKE"
		if s.driver == "pgx" {
			textLike = "ILIKE"
		}
		like := "%" + q + "%"
		rows, err = s.DB.QueryContext(
			ctx,
			s.bind(`SELECT `+cols+` FROM products
WHERE CAST(id AS TEXT)=?
   OR CAST(product_api_id AS TEXT) LIKE ?
   OR CAST(COALESCE(products, '') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(status, '') AS TEXT) `+textLike+` ?
ORDER BY id DESC
LIMIT ?`),
			q, like, like, like, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ProdutoRecord, 0, limit)
	for rows.Next() {
		var r ProdutoRecord
		if err := rows.Scan(&r.ID, &r.IDProduto, &r.Produto, &r.Status); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetProdutoByID(ctx context.Context, id int64) (*ProdutoRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id invalido")
	}
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER),
        CAST(COALESCE(NULLIF(CAST(product_api_id AS TEXT), ''), '0') AS INTEGER),
        CAST(COALESCE(products, '') AS TEXT),
        CAST(COALESCE(status, '') AS TEXT)
   FROM products
  WHERE id=?`),
		id,
	)
	var r ProdutoRecord
	if err := row.Scan(&r.ID, &r.IDProduto, &r.Produto, &r.Status); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) SaveProduto(ctx context.Context, r ProdutoRecord) (int64, error) {
	products := strings.TrimSpace(r.Produto)
	if r.IDProduto <= 0 || products == "" {
		return 0, fmt.Errorf("product_api_id e products obrigatorios")
	}
	status := strings.TrimSpace(r.Status)
	if status == "" {
		status = "is_active"
	}

	if r.ID > 0 {
		_, err := s.DB.ExecContext(
			ctx,
			s.bind(`UPDATE products
SET product_api_id=?, products=?, status=?
WHERE id=?`),
			r.IDProduto, products, status, r.ID,
		)
		if err != nil {
			return 0, err
		}
		return r.ID, nil
	}

	if s.driver == "pgx" {
		var id int64
		err := s.DB.QueryRowContext(
			ctx,
			s.bind(`INSERT INTO products (product_api_id, products, status)
VALUES (?, ?, ?) RETURNING id`),
			r.IDProduto, products, status,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		s.bind(`INSERT INTO products (product_api_id, products, status)
VALUES (?, ?, ?)`),
		r.IDProduto, products, status,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteProduto(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM products WHERE id=?"), id)
	return err
}

func (s *Store) SearchAssembleias(ctx context.Context, query string, limit int) ([]AssembleiaRecord, error) {
	if limit <= 0 {
		limit = 1000
	}
	q := strings.TrimSpace(query)
	const cols = `CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER) AS id,
CAST(COALESCE(quota_rd, '') AS TEXT) AS quota_rd,
CAST(COALESCE(CAST(contemplation_date AS TEXT), '') AS TEXT) AS contemplation_date,
CAST(COALESCE(contemplation_type, '') AS TEXT) AS contemplation_type,
CAST(COALESCE(CAST(disqualification_date AS TEXT), '') AS TEXT) AS disqualification_date,
CAST(COALESCE(client_name, '') AS TEXT) AS client_name,
CASE WHEN TRIM(COALESCE(CAST(bid_percent AS TEXT), '')) = '' THEN NULL ELSE CAST(REPLACE(CAST(bid_percent AS TEXT), ',', '.') AS REAL) END AS bid_percent,
CAST(COALESCE(seller_name, '') AS TEXT) AS seller_name,
CASE WHEN TRIM(COALESCE(CAST(group_code AS TEXT), '')) = '' THEN NULL ELSE CAST(CAST(group_code AS TEXT) AS INTEGER) END AS group_code,
CASE WHEN TRIM(COALESCE(CAST(federal_lottery AS TEXT), '')) = '' THEN NULL ELSE CAST(CAST(federal_lottery AS TEXT) AS INTEGER) END AS federal_lottery,
CAST(COALESCE(group_quota_rd, '') AS TEXT) AS group_quota_rd`

	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = s.DB.QueryContext(ctx, s.bind("SELECT "+cols+" FROM assemblies ORDER BY id DESC LIMIT ?"), limit)
	} else {
		textLike := "LIKE"
		if s.driver == "pgx" {
			textLike = "ILIKE"
		}
		like := "%" + q + "%"
		rows, err = s.DB.QueryContext(
			ctx,
			s.bind(`SELECT `+cols+` FROM assemblies
WHERE CAST(id AS TEXT)=?
   OR CAST(COALESCE(quota_rd, '') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(CAST(contemplation_date AS TEXT), '') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(contemplation_type, '') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(CAST(disqualification_date AS TEXT), '') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(client_name, '') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(CAST(bid_percent AS TEXT), '') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(seller_name, '') AS TEXT) `+textLike+` ?
   OR CAST(COALESCE(CAST(group_code AS TEXT), '') AS TEXT) LIKE ?
   OR CAST(COALESCE(CAST(federal_lottery AS TEXT), '') AS TEXT) LIKE ?
   OR CAST(COALESCE(group_quota_rd, '') AS TEXT) `+textLike+` ?
ORDER BY id DESC
LIMIT ?`),
			q, like, like, like, like, like, like, like, like, like, like, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AssembleiaRecord, 0, limit)
	for rows.Next() {
		var r AssembleiaRecord
		var dataCont, tipoCont, dataDesc, client_name, seller_name, quota_rd, grupoCota string
		if err := rows.Scan(
			&r.ID,
			&quota_rd,
			&dataCont,
			&tipoCont,
			&dataDesc,
			&client_name,
			&r.PercLance,
			&seller_name,
			&r.Grupo,
			&r.LoteriaFederal,
			&grupoCota,
		); err != nil {
			return nil, err
		}
		r.CotaRD = quota_rd
		r.TipoContemplacao = tipoCont
		r.ClientName = client_name
		r.Vendedor = seller_name
		r.GrupoCotaRD = grupoCota
		r.DataContemplacao = sql.NullString{String: dataCont, Valid: strings.TrimSpace(dataCont) != ""}
		r.DataDesclassificao = sql.NullString{String: dataDesc, Valid: strings.TrimSpace(dataDesc) != ""}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetAssembleiaByID(ctx context.Context, id int64) (*AssembleiaRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id invalido")
	}
	row := s.DB.QueryRowContext(ctx, s.bind(`
SELECT CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(quota_rd, '') AS TEXT),
       CAST(COALESCE(CAST(contemplation_date AS TEXT), '') AS TEXT),
       CAST(COALESCE(contemplation_type, '') AS TEXT),
       CAST(COALESCE(CAST(disqualification_date AS TEXT), '') AS TEXT),
       CAST(COALESCE(client_name, '') AS TEXT),
       CASE WHEN TRIM(COALESCE(CAST(bid_percent AS TEXT), '')) = '' THEN NULL ELSE CAST(REPLACE(CAST(bid_percent AS TEXT), ',', '.') AS REAL) END,
       CAST(COALESCE(seller_name, '') AS TEXT),
       CASE WHEN TRIM(COALESCE(CAST(group_code AS TEXT), '')) = '' THEN NULL ELSE CAST(CAST(group_code AS TEXT) AS INTEGER) END,
       CASE WHEN TRIM(COALESCE(CAST(federal_lottery AS TEXT), '')) = '' THEN NULL ELSE CAST(CAST(federal_lottery AS TEXT) AS INTEGER) END,
       CAST(COALESCE(group_quota_rd, '') AS TEXT)
  FROM assemblies
 WHERE id=?`), id)
	var r AssembleiaRecord
	var dataCont, tipoCont, dataDesc, client_name, seller_name, quota_rd, grupoCota string
	if err := row.Scan(
		&r.ID,
		&quota_rd,
		&dataCont,
		&tipoCont,
		&dataDesc,
		&client_name,
		&r.PercLance,
		&seller_name,
		&r.Grupo,
		&r.LoteriaFederal,
		&grupoCota,
	); err != nil {
		return nil, err
	}
	r.CotaRD = quota_rd
	r.TipoContemplacao = tipoCont
	r.ClientName = client_name
	r.Vendedor = seller_name
	r.GrupoCotaRD = grupoCota
	r.DataContemplacao = sql.NullString{String: dataCont, Valid: strings.TrimSpace(dataCont) != ""}
	r.DataDesclassificao = sql.NullString{String: dataDesc, Valid: strings.TrimSpace(dataDesc) != ""}
	return &r, nil
}

func (s *Store) SaveAssembleia(ctx context.Context, r AssembleiaRecord) (int64, error) {
	if r.ID > 0 {
		_, err := s.DB.ExecContext(
			ctx,
			s.bind(`UPDATE assemblies
SET quota_rd=?,
    contemplation_date=?,
    contemplation_type=?,
    disqualification_date=?,
    client_name=?,
    bid_percent=?,
    seller_name=?,
    group_code=?,
    federal_lottery=?
WHERE id=?`),
			strings.TrimSpace(r.CotaRD),
			nullStringOrNil(r.DataContemplacao),
			strings.TrimSpace(r.TipoContemplacao),
			nullStringOrNil(r.DataDesclassificao),
			strings.TrimSpace(r.ClientName),
			nullFloat64OrNil(r.PercLance),
			strings.TrimSpace(r.Vendedor),
			nullInt64OrNil(r.Grupo),
			nullInt64OrNil(r.LoteriaFederal),
			r.ID,
		)
		if err != nil {
			return 0, err
		}
		return r.ID, nil
	}

	if s.driver == "pgx" {
		var id int64
		err := s.DB.QueryRowContext(
			ctx,
			s.bind(`INSERT INTO assemblies
(quota_rd, contemplation_date, contemplation_type, disqualification_date, client_name, bid_percent, seller_name, group_code, federal_lottery)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
			strings.TrimSpace(r.CotaRD),
			nullStringOrNil(r.DataContemplacao),
			strings.TrimSpace(r.TipoContemplacao),
			nullStringOrNil(r.DataDesclassificao),
			strings.TrimSpace(r.ClientName),
			nullFloat64OrNil(r.PercLance),
			strings.TrimSpace(r.Vendedor),
			nullInt64OrNil(r.Grupo),
			nullInt64OrNil(r.LoteriaFederal),
		).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		s.bind(`INSERT INTO assemblies
(quota_rd, contemplation_date, contemplation_type, disqualification_date, client_name, bid_percent, seller_name, group_code, federal_lottery)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		strings.TrimSpace(r.CotaRD),
		nullStringOrNil(r.DataContemplacao),
		strings.TrimSpace(r.TipoContemplacao),
		nullStringOrNil(r.DataDesclassificao),
		strings.TrimSpace(r.ClientName),
		nullFloat64OrNil(r.PercLance),
		strings.TrimSpace(r.Vendedor),
		nullInt64OrNil(r.Grupo),
		nullInt64OrNil(r.LoteriaFederal),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteAssembleia(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM assemblies WHERE id=?"), id)
	return err
}

func (s *Store) SearchGruposAtivos(ctx context.Context, query, column string, limit int) ([]GrupoAtivoRecord, error) {
	if limit <= 0 {
		limit = 1000
	}
	q := strings.TrimSpace(query)
	col := strings.ToLower(strings.TrimSpace(column))

	const cols = `CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER) AS id,
CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) AS group_code,
CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER) AS due_day,
CAST(COALESCE(NULLIF(CAST(participants_count AS TEXT), ''), '0') AS INTEGER) AS participants_count,
CAST(COALESCE(CAST(first_assembly_date AS TEXT), '') AS TEXT) AS first_assembly_date,
CAST(COALESCE(plan, '') AS TEXT) AS plan,
CAST(COALESCE(NULLIF(CAST(term_months AS TEXT), ''), '0') AS INTEGER) AS term_months,
CAST(COALESCE(group_type, '') AS TEXT) AS group_type,
CAST(COALESCE(models, '') AS TEXT) AS models,
CAST(COALESCE(status, '') AS TEXT) AS status,
CAST(COALESCE(CAST(created_at AS TEXT), '') AS TEXT) AS created_at,
CAST(COALESCE(CAST(updated_at AS TEXT), '') AS TEXT) AS updated_at`

	whereSQL := ""
	whereArgs := make([]any, 0, 12)
	if q != "" {
		textLike := "LIKE"
		if s.driver == "pgx" {
			textLike = "ILIKE"
		}
		like := "%" + q + "%"
		if col == "" {
			whereSQL = `
WHERE CAST(id AS TEXT)=?
   OR CAST(group_code AS TEXT) LIKE ?
   OR CAST(due_day AS TEXT) LIKE ?
   OR CAST(participants_count AS TEXT) LIKE ?
   OR CAST(first_assembly_date AS TEXT) ` + textLike + ` ?
   OR CAST(plan AS TEXT) ` + textLike + ` ?
   OR CAST(term_months AS TEXT) LIKE ?
   OR CAST(group_type AS TEXT) ` + textLike + ` ?
   OR CAST(models AS TEXT) ` + textLike + ` ?
   OR CAST(status AS TEXT) ` + textLike + ` ?
   OR CAST(created_at AS TEXT) LIKE ?
   OR CAST(updated_at AS TEXT) LIKE ?`
			whereArgs = []any{q, like, like, like, like, like, like, like, like, like, like, like}
		} else {
			numericCols := map[string]string{
				"id":                 "id",
				"group_code":         "group_code",
				"due_day":            "due_day",
				"participants_count": "participants_count",
				"term_months":        "term_months",
			}
			textCols := map[string]string{
				"first_assembly_date": "first_assembly_date",
				"plan":                "plan",
				"group_type":          "group_type",
				"models":              "models",
				"status":              "status",
				"created_at":          "created_at",
				"updated_at":          "updated_at",
			}
			if rawCol, ok := numericCols[col]; ok {
				n, convErr := parseInt64Safe(q)
				if convErr != nil {
					return []GrupoAtivoRecord{}, nil
				}
				whereSQL = "WHERE CAST(COALESCE(NULLIF(CAST(" + rawCol + " AS TEXT), ''), '0') AS INTEGER)=?"
				whereArgs = []any{n}
			} else if rawCol, ok := textCols[col]; ok {
				whereSQL = "WHERE CAST(COALESCE(" + rawCol + ", '') AS TEXT) " + textLike + " ?"
				whereArgs = []any{"%" + q + "%"}
			} else {
				return nil, fmt.Errorf("coluna de busca invalida")
			}
		}
	}

	querySQL := "SELECT " + cols + " FROM active_groups " + whereSQL + " ORDER BY id DESC LIMIT ?"
	args := append([]any{}, whereArgs...)
	args = append(args, limit)
	rows, err := s.DB.QueryContext(ctx, s.bind(querySQL), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]GrupoAtivoRecord, 0, limit)
	for rows.Next() {
		var r GrupoAtivoRecord
		var dataAssembleia, createdAt, updatedAt string
		if err := rows.Scan(
			&r.ID,
			&r.Grupo,
			&r.Vencimento,
			&r.QtdParticipantes,
			&dataAssembleia,
			&r.Plano,
			&r.Prazo,
			&r.TipoGrupo,
			&r.Modelos,
			&r.Status,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		r.DataAssembleiaInaugural = sql.NullString{String: dataAssembleia, Valid: strings.TrimSpace(dataAssembleia) != ""}
		r.CreatedAt = sql.NullString{String: createdAt, Valid: strings.TrimSpace(createdAt) != ""}
		r.UpdatedAt = sql.NullString{String: updatedAt, Valid: strings.TrimSpace(updatedAt) != ""}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetGrupoAtivoByID(ctx context.Context, id int64) (*GrupoAtivoRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("id invalido")
	}
	row := s.DB.QueryRowContext(ctx, s.bind(`
SELECT CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(participants_count AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(CAST(first_assembly_date AS TEXT), '') AS TEXT),
       CAST(COALESCE(plan, '') AS TEXT),
       CAST(COALESCE(NULLIF(CAST(term_months AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(group_type, '') AS TEXT),
       CAST(COALESCE(models, '') AS TEXT),
       CAST(COALESCE(status, '') AS TEXT),
       CAST(COALESCE(CAST(created_at AS TEXT), '') AS TEXT),
       CAST(COALESCE(CAST(updated_at AS TEXT), '') AS TEXT)
  FROM active_groups
 WHERE id=?`), id)
	var r GrupoAtivoRecord
	var dataAssembleia, createdAt, updatedAt string
	if err := row.Scan(
		&r.ID,
		&r.Grupo,
		&r.Vencimento,
		&r.QtdParticipantes,
		&dataAssembleia,
		&r.Plano,
		&r.Prazo,
		&r.TipoGrupo,
		&r.Modelos,
		&r.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	r.DataAssembleiaInaugural = sql.NullString{String: dataAssembleia, Valid: strings.TrimSpace(dataAssembleia) != ""}
	r.CreatedAt = sql.NullString{String: createdAt, Valid: strings.TrimSpace(createdAt) != ""}
	r.UpdatedAt = sql.NullString{String: updatedAt, Valid: strings.TrimSpace(updatedAt) != ""}
	return &r, nil
}

func (s *Store) SaveGrupoAtivo(ctx context.Context, r GrupoAtivoRecord) (int64, error) {
	if r.Grupo <= 0 {
		return 0, fmt.Errorf("group_code obrigatorio")
	}
	if r.Vencimento <= 0 {
		return 0, fmt.Errorf("due_day obrigatorio")
	}
	if r.Status == "" {
		r.Status = "is_active"
	}
	nowStr := formatDateTimeUTCMinus3(nowUTCMinus3())
	if !r.UpdatedAt.Valid || strings.TrimSpace(r.UpdatedAt.String) == "" {
		r.UpdatedAt = sql.NullString{String: nowStr, Valid: true}
	}

	if r.ID > 0 {
		_, err := s.DB.ExecContext(ctx, s.bind(`
UPDATE active_groups
SET group_code=?,
    due_day=?,
    participants_count=?,
    first_assembly_date=?,
    plan=?,
    term_months=?,
    group_type=?,
    models=?,
    status=?,
    updated_at=?
WHERE id=?`),
			r.Grupo,
			r.Vencimento,
			r.QtdParticipantes,
			nullStringOrNil(r.DataAssembleiaInaugural),
			strings.TrimSpace(r.Plano),
			r.Prazo,
			strings.TrimSpace(r.TipoGrupo),
			strings.TrimSpace(r.Modelos),
			strings.TrimSpace(r.Status),
			r.UpdatedAt.String,
			r.ID,
		)
		if err != nil {
			return 0, err
		}
		return r.ID, nil
	}

	created := r.CreatedAt
	if !created.Valid || strings.TrimSpace(created.String) == "" {
		created = sql.NullString{String: nowStr, Valid: true}
	}
	if s.driver == "pgx" {
		var id int64
		err := s.DB.QueryRowContext(ctx, s.bind(`
INSERT INTO active_groups
  (group_code, due_day, participants_count, first_assembly_date, plan, term_months, group_type, models, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
			r.Grupo,
			r.Vencimento,
			r.QtdParticipantes,
			nullStringOrNil(r.DataAssembleiaInaugural),
			strings.TrimSpace(r.Plano),
			r.Prazo,
			strings.TrimSpace(r.TipoGrupo),
			strings.TrimSpace(r.Modelos),
			strings.TrimSpace(r.Status),
			created.String,
			r.UpdatedAt.String,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.DB.ExecContext(ctx, s.bind(`
INSERT INTO active_groups
  (group_code, due_day, participants_count, first_assembly_date, plan, term_months, group_type, models, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		r.Grupo,
		r.Vencimento,
		r.QtdParticipantes,
		nullStringOrNil(r.DataAssembleiaInaugural),
		strings.TrimSpace(r.Plano),
		r.Prazo,
		strings.TrimSpace(r.TipoGrupo),
		strings.TrimSpace(r.Modelos),
		strings.TrimSpace(r.Status),
		created.String,
		r.UpdatedAt.String,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) DeleteGrupoAtivo(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM active_groups WHERE id=?"), id)
	return err
}

func (s *Store) DeleteGruposAtivosByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			args = append(args, id)
		}
	}
	if len(args) == 0 {
		return 0, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		s.bind("DELETE FROM active_groups WHERE id IN ("+placeholders(len(args))+")"),
		args...,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) GetGrupoAtivoByGrupo(ctx context.Context, group_code int64) (*GrupoAtivoRecord, error) {
	if group_code <= 0 {
		return nil, fmt.Errorf("group_code invalido")
	}
	row := s.DB.QueryRowContext(ctx, s.bind(`
SELECT CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(participants_count AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(CAST(first_assembly_date AS TEXT), '') AS TEXT),
       CAST(COALESCE(plan, '') AS TEXT),
       CAST(COALESCE(NULLIF(CAST(term_months AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(group_type, '') AS TEXT),
       CAST(COALESCE(models, '') AS TEXT),
       CAST(COALESCE(status, '') AS TEXT),
       CAST(COALESCE(CAST(created_at AS TEXT), '') AS TEXT),
       CAST(COALESCE(CAST(updated_at AS TEXT), '') AS TEXT)
  FROM active_groups
 WHERE CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER)=?
 ORDER BY id DESC
 LIMIT 1`), group_code)
	var r GrupoAtivoRecord
	var dataAssembleia, createdAt, updatedAt string
	if err := row.Scan(
		&r.ID,
		&r.Grupo,
		&r.Vencimento,
		&r.QtdParticipantes,
		&dataAssembleia,
		&r.Plano,
		&r.Prazo,
		&r.TipoGrupo,
		&r.Modelos,
		&r.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	r.DataAssembleiaInaugural = sql.NullString{String: dataAssembleia, Valid: strings.TrimSpace(dataAssembleia) != ""}
	r.CreatedAt = sql.NullString{String: createdAt, Valid: strings.TrimSpace(createdAt) != ""}
	r.UpdatedAt = sql.NullString{String: updatedAt, Valid: strings.TrimSpace(updatedAt) != ""}
	return &r, nil
}

func (s *Store) GetGruposAtivosByGrupos(ctx context.Context, grupos []int64) (map[int64]GrupoAtivoRecord, error) {
	out := make(map[int64]GrupoAtivoRecord)
	if len(grupos) == 0 {
		return out, nil
	}
	unique := make(map[int64]struct{}, len(grupos))
	args := make([]any, 0, len(grupos))
	for _, g := range grupos {
		if g <= 0 {
			continue
		}
		if _, ok := unique[g]; ok {
			continue
		}
		unique[g] = struct{}{}
		args = append(args, g)
	}
	if len(args) == 0 {
		return out, nil
	}
	sqlQ := `
SELECT CAST(COALESCE(NULLIF(CAST(id AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(due_day AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(NULLIF(CAST(participants_count AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(CAST(first_assembly_date AS TEXT), '') AS TEXT),
       CAST(COALESCE(plan, '') AS TEXT),
       CAST(COALESCE(NULLIF(CAST(term_months AS TEXT), ''), '0') AS INTEGER),
       CAST(COALESCE(group_type, '') AS TEXT),
       CAST(COALESCE(models, '') AS TEXT),
       CAST(COALESCE(status, '') AS TEXT),
       CAST(COALESCE(CAST(created_at AS TEXT), '') AS TEXT),
       CAST(COALESCE(CAST(updated_at AS TEXT), '') AS TEXT)
  FROM active_groups
 WHERE CAST(COALESCE(NULLIF(CAST(group_code AS TEXT), ''), '0') AS INTEGER) IN (` + placeholders(len(args)) + `)
 ORDER BY id DESC`
	rows, err := s.DB.QueryContext(ctx, s.bind(sqlQ), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r GrupoAtivoRecord
		var dataAssembleia, createdAt, updatedAt string
		if err := rows.Scan(
			&r.ID,
			&r.Grupo,
			&r.Vencimento,
			&r.QtdParticipantes,
			&dataAssembleia,
			&r.Plano,
			&r.Prazo,
			&r.TipoGrupo,
			&r.Modelos,
			&r.Status,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if _, exists := out[r.Grupo]; exists {
			continue
		}
		r.DataAssembleiaInaugural = sql.NullString{String: dataAssembleia, Valid: strings.TrimSpace(dataAssembleia) != ""}
		r.CreatedAt = sql.NullString{String: createdAt, Valid: strings.TrimSpace(createdAt) != ""}
		r.UpdatedAt = sql.NullString{String: updatedAt, Valid: strings.TrimSpace(updatedAt) != ""}
		out[r.Grupo] = r
	}
	return out, rows.Err()
}

func (s *Store) ListActiveNationalHolidayDates(ctx context.Context) (map[string]struct{}, error) {
	rows, err := s.DB.QueryContext(ctx, `
SELECT CAST(COALESCE(CAST(holiday_date AS TEXT), '') AS TEXT)
  FROM holidays
 WHERE COALESCE(is_active, 1)=1
   AND LOWER(COALESCE(holiday_type, 'nacional'))='nacional'
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		out[d] = struct{}{}
	}
	return out, rows.Err()
}

func (s *Store) ReserveUser(ctx context.Context, userID int64, now time.Time, cooldownUntil time.Time) error {
	nowStr := formatDateTimeUTCMinus3(now)
	cooldownStr := formatDateTimeUTCMinus3(cooldownUntil)
	_, err := s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE api_accounts
		 SET in_flight=COALESCE(in_flight,0)+1,
		     last_request_at=?,
		     cooldown_until=?
		 WHERE id=?`),
		nowStr, cooldownStr, userID,
	)
	return err
}

func (s *Store) ReleaseUser(ctx context.Context, userID int64) error {
	_, err := s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE api_accounts
		 SET in_flight=CASE WHEN COALESCE(in_flight,0)>0 THEN in_flight-1 ELSE 0 END
		 WHERE id=?`),
		userID,
	)
	return err
}

func (s *Store) MarkUserStatus(ctx context.Context, userID int64, status int) error {
	switch status {
	case 200:
		_, err := s.DB.ExecContext(
			ctx,
			s.bind(`UPDATE api_accounts
			 SET error_401_count=0, error_429_count=0, priority_score=COALESCE(priority_score,0)+1
			 WHERE id=?`),
			userID,
		)
		return err
	case 401:
		blockedUntil := formatDateTimeUTCMinus3(nowUTCMinus3().Add(10 * time.Minute))
		_, err := s.DB.ExecContext(
			ctx,
			s.bind(`UPDATE api_accounts
			 SET error_401_count=COALESCE(error_401_count,0)+1,
			     blocked_until=CASE WHEN COALESCE(error_401_count,0)+1 >= 3
			                        THEN ?
			                        ELSE blocked_until END,
			     priority_score=COALESCE(priority_score,0)-1
			 WHERE id=?`),
			blockedUntil, userID,
		)
		return err
	case 429:
		blockedUntil := formatDateTimeUTCMinus3(nowUTCMinus3().Add(2 * time.Minute))
		_, err := s.DB.ExecContext(
			ctx,
			s.bind(`UPDATE api_accounts
			 SET error_429_count=COALESCE(error_429_count,0)+1,
			     blocked_until=CASE WHEN COALESCE(error_429_count,0)+1 >= 3
			                        THEN ?
			                        ELSE blocked_until END,
			     priority_score=COALESCE(priority_score,0)-0.5
			 WHERE id=?`),
			blockedUntil, userID,
		)
		return err
	default:
		return nil
	}
}

func (s *Store) SetUserToken(ctx context.Context, userID int64, token string) error {
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	_, err := s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE api_accounts
		 SET token=?,
		     b3_token=?,
		     last_request_at=?
		 WHERE id=?`),
		token, token, now, userID,
	)
	return err
}

func (s *Store) ClearUserToken(ctx context.Context, userID int64) error {
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	_, err := s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE api_accounts
		 SET token=NULL,
		     b3_token=NULL,
		     last_request_at=?
		 WHERE id=?`),
		now, userID,
	)
	return err
}

func (s *Store) LoadUserForAuthByID(ctx context.Context, userID int64) (*User, error) {
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT
  id, cpf, company_code, account_user, account_password, COALESCE(token, '') AS token, last_request_at,
  cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score
FROM api_accounts
WHERE id=?`),
		userID,
	)
	var u User
	if err := row.Scan(
		&u.ID, &u.CPF, &u.CodEmpresa, &u.CodUsuario, &u.Senha, &u.Token, &u.LastRequest,
		&u.CooldownUntil, &u.InFlight, &u.Error401Count, &u.Error429Count, &u.BlockedUntil, &u.PriorityScore,
	); err != nil {
		return nil, err
	}
	if err := revealUserPassword(&u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) FindAuthRecord(ctx context.Context, query string) (*AuthRecord, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("consulta vazia")
	}

	const cols = `id, cpf, company_code, account_user, dealer_code, account_password, token, b3_token, last_request_at, cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score`

	// Busca exata primeiro (id/cpf/account_user).
	row := s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT `+cols+` FROM api_accounts
WHERE CAST(id AS TEXT)=? OR cpf=? OR account_user=?
ORDER BY id ASC
LIMIT 1`),
		q, q, q,
	)
	var r AuthRecord
	err := row.Scan(
		&r.ID, &r.CPF, &r.CodEmpresa, &r.CodUsuario, &r.CodConcessionaria, &r.Senha,
		&r.Token, &r.TokenB3, &r.LastRequest, &r.CooldownUntil, &r.InFlight,
		&r.Error401Count, &r.Error429Count, &r.BlockedUntil, &r.PriorityScore,
	)
	if err == nil {
		if err := revealAuthPassword(&r); err != nil {
			return nil, err
		}
		return &r, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	likeOp := "LIKE"
	if s.driver == "pgx" {
		likeOp = "ILIKE"
	}
	like := "%" + q + "%"
	row = s.DB.QueryRowContext(
		ctx,
		s.bind(`SELECT `+cols+` FROM api_accounts
WHERE CAST(COALESCE(cpf, '') AS TEXT) `+likeOp+` ? OR CAST(COALESCE(account_user, '') AS TEXT) `+likeOp+` ? OR CAST(COALESCE(company_code, '') AS TEXT) `+likeOp+` ?
ORDER BY id ASC
LIMIT 1`),
		like, like, like,
	)
	err = row.Scan(
		&r.ID, &r.CPF, &r.CodEmpresa, &r.CodUsuario, &r.CodConcessionaria, &r.Senha,
		&r.Token, &r.TokenB3, &r.LastRequest, &r.CooldownUntil, &r.InFlight,
		&r.Error401Count, &r.Error429Count, &r.BlockedUntil, &r.PriorityScore,
	)
	if err != nil {
		return nil, err
	}
	if err := revealAuthPassword(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) UpdateAuthRecord(ctx context.Context, r AuthRecord) error {
	if r.ID <= 0 {
		return fmt.Errorf("id invalido")
	}
	encryptedPassword, err := encryptAtRest(r.Senha)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(
		ctx,
		s.bind(`UPDATE api_accounts
		 SET cpf=?,
		     company_code=?,
		     account_user=?,
		     dealer_code=?,
		     account_password=?
		 WHERE id=?`),
		strings.TrimSpace(r.CPF),
		strings.TrimSpace(r.CodEmpresa),
		strings.TrimSpace(r.CodUsuario),
		strings.TrimSpace(r.CodConcessionaria),
		encryptedPassword,
		r.ID,
	)
	return err
}

func (s *Store) InsertAuthRecord(ctx context.Context, r AuthRecord) (int64, error) {
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	encryptedPassword, err := encryptAtRest(r.Senha)
	if err != nil {
		return 0, err
	}
	if s.driver == "pgx" {
		var id int64
		err := s.DB.QueryRowContext(
			ctx,
			s.bind(`INSERT INTO api_accounts (cpf, company_code, account_user, dealer_code, account_password, last_request_at)
		 VALUES (?, ?, ?, ?, ?, ?) RETURNING id`),
			strings.TrimSpace(r.CPF),
			strings.TrimSpace(r.CodEmpresa),
			strings.TrimSpace(r.CodUsuario),
			strings.TrimSpace(r.CodConcessionaria),
			encryptedPassword,
			now,
		).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := s.DB.ExecContext(
		ctx,
		s.bind(`INSERT INTO api_accounts (cpf, company_code, account_user, dealer_code, account_password, last_request_at)
		 VALUES (?, ?, ?, ?, ?, ?)`),
		strings.TrimSpace(r.CPF),
		strings.TrimSpace(r.CodEmpresa),
		strings.TrimSpace(r.CodUsuario),
		strings.TrimSpace(r.CodConcessionaria),
		encryptedPassword,
		now,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) DeleteAuthRecord(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return fmt.Errorf("id invalido")
	}
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM api_accounts WHERE id=?"), userID)
	return err
}

func (s *Store) GetAuthRecordByID(ctx context.Context, userID int64) (*AuthRecord, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("id invalido")
	}
	const cols = `id, cpf, company_code, account_user, dealer_code, account_password, token, b3_token, last_request_at, cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score`
	row := s.DB.QueryRowContext(ctx, s.bind("SELECT "+cols+" FROM api_accounts WHERE id=?"), userID)
	var r AuthRecord
	if err := row.Scan(
		&r.ID, &r.CPF, &r.CodEmpresa, &r.CodUsuario, &r.CodConcessionaria, &r.Senha,
		&r.Token, &r.TokenB3, &r.LastRequest, &r.CooldownUntil, &r.InFlight,
		&r.Error401Count, &r.Error429Count, &r.BlockedUntil, &r.PriorityScore,
	); err != nil {
		return nil, err
	}
	if err := revealAuthPassword(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) GetAuthRecordByCPF(ctx context.Context, cpf string) (*AuthRecord, error) {
	c := strings.TrimSpace(cpf)
	if c == "" {
		return nil, fmt.Errorf("cpf vazio")
	}
	const cols = `id, cpf, company_code, account_user, dealer_code, account_password, token, b3_token, last_request_at, cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score`
	row := s.DB.QueryRowContext(
		ctx,
		s.bind("SELECT "+cols+" FROM api_accounts WHERE cpf = ? OR REPLACE(REPLACE(cpf, '.', ''), '-', '') = ? LIMIT 1"),
		c, c,
	)
	var r AuthRecord
	if err := row.Scan(
		&r.ID, &r.CPF, &r.CodEmpresa, &r.CodUsuario, &r.CodConcessionaria, &r.Senha,
		&r.Token, &r.TokenB3, &r.LastRequest, &r.CooldownUntil, &r.InFlight,
		&r.Error401Count, &r.Error429Count, &r.BlockedUntil, &r.PriorityScore,
	); err != nil {
		return nil, err
	}
	if err := revealAuthPassword(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) SearchAuthRecords(ctx context.Context, query string, limit int) ([]AuthRecord, error) {
	if limit <= 0 {
		limit = 200
	}
	q := strings.TrimSpace(query)
	const cols = `id, cpf, company_code, account_user, dealer_code, account_password, token, b3_token, last_request_at, cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score`

	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = s.DB.QueryContext(ctx, s.bind("SELECT "+cols+" FROM api_accounts ORDER BY id ASC LIMIT ?"), limit)
	} else {
		likeOp := "LIKE"
		if s.driver == "pgx" {
			likeOp = "ILIKE"
		}
		like := "%" + q + "%"
		rows, err = s.DB.QueryContext(
			ctx,
			s.bind(`SELECT `+cols+` FROM api_accounts
WHERE CAST(id AS TEXT)=? OR CAST(COALESCE(cpf, '') AS TEXT) `+likeOp+` ? OR CAST(COALESCE(account_user, '') AS TEXT) `+likeOp+` ? OR CAST(COALESCE(company_code, '') AS TEXT) `+likeOp+` ?
ORDER BY id ASC
LIMIT ?`),
			q, like, like, like, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AuthRecord, 0, limit)
	for rows.Next() {
		var r AuthRecord
		if err := rows.Scan(
			&r.ID, &r.CPF, &r.CodEmpresa, &r.CodUsuario, &r.CodConcessionaria, &r.Senha,
			&r.Token, &r.TokenB3, &r.LastRequest, &r.CooldownUntil, &r.InFlight,
			&r.Error401Count, &r.Error429Count, &r.BlockedUntil, &r.PriorityScore,
		); err != nil {
			return nil, err
		}
		if err := revealAuthPassword(&r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func nullIntArg(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func nullFloatArg(v sql.NullFloat64) any {
	if !v.Valid {
		return nil
	}
	return v.Float64
}

func nullStringArg(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	s := strings.TrimSpace(v.String)
	if s == "" {
		return nil
	}
	return s
}

func splitDateTimeParts(v sql.NullString) (sql.NullString, sql.NullString) {
	if !v.Valid {
		return sql.NullString{}, sql.NullString{}
	}
	raw := strings.TrimSpace(v.String)
	if raw == "" {
		return sql.NullString{}, sql.NullString{}
	}

	type layoutSpec struct {
		layout  string
		hasTime bool
	}
	layouts := []layoutSpec{
		{time.RFC3339Nano, true},
		{time.RFC3339, true},
		{"2006-01-02 15:04:05", true},
		{"2006-01-02 15:04", true},
		{"2006-01-02T15:04:05", true},
		{"2006-01-02T15:04", true},
		{"02/01/2006 15:04:05", true},
		{"02/01/2006 15:04", true},
		{"2006-01-02", false},
		{"02/01/2006", false},
	}
	for _, spec := range layouts {
		t, err := time.Parse(spec.layout, raw)
		if err != nil {
			continue
		}
		datePart := sql.NullString{String: t.Format("2006-01-02"), Valid: true}
		if spec.hasTime {
			return datePart, sql.NullString{String: t.Format("15:04:05"), Valid: true}
		}
		return datePart, sql.NullString{}
	}

	if len(raw) >= 10 {
		dateRaw := raw[:10]
		if len(dateRaw) == 10 && dateRaw[2] == '/' && dateRaw[5] == '/' {
			dateRaw = dateRaw[6:10] + "-" + dateRaw[3:5] + "-" + dateRaw[0:2]
		}
		datePart := sql.NullString{String: dateRaw, Valid: strings.TrimSpace(dateRaw) != ""}
		if len(raw) >= 19 {
			timeRaw := raw[11:19]
			timePart := sql.NullString{String: timeRaw, Valid: strings.TrimSpace(timeRaw) != ""}
			return datePart, timePart
		}
		return datePart, sql.NullString{}
	}

	return sql.NullString{}, sql.NullString{}
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]string, n)
	for i := range out {
		out[i] = "?"
	}
	return strings.Join(out, ",")
}

func (s *Store) GetRolePermissions(ctx context.Context, roleName string) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, s.bind("SELECT p.permission_key FROM role_permissions p INNER JOIN roles r ON p.role_id = r.id WHERE r.name = ?"), roleName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

type RoleMatrix struct {
	RoleID      int64    `json:"role_id"`
	RoleName    string   `json:"role_name"`
	Permissions []string `json:"permissions"`
}

func (s *Store) GetAllRolesAndPermissions(ctx context.Context) ([]RoleMatrix, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id, name FROM roles ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []RoleMatrix
	for rows.Next() {
		var r RoleMatrix
		if err := rows.Scan(&r.RoleID, &r.RoleName); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	for i, r := range roles {
		perms, err := s.GetRolePermissions(ctx, r.RoleName)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = perms
	}

	return roles, nil
}

func (s *Store) UpdateRolePermissions(ctx context.Context, roleName string, permissions []string) error {
	roleName = strings.TrimSpace(roleName)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var roleID int64
	// Busca case-insensitive para evitar falhas de match entre frontend e banco
	err = tx.QueryRowContext(ctx, s.bind("SELECT id FROM roles WHERE LOWER(name) = LOWER(?)"), roleName).Scan(&roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("perfil '%s' nao encontrado", roleName)
		}
		return err
	}

	_, err = tx.ExecContext(ctx, s.bind("DELETE FROM role_permissions WHERE role_id = ?"), roleID)
	if err != nil {
		return err
	}

	for _, p := range permissions {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, err = tx.ExecContext(ctx, s.bind("INSERT INTO role_permissions (role_id, permission_key) VALUES (?, ?)"), roleID, p)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) CreateAuditLog(ctx context.Context, r AuditLogRecord) error {
	now := formatDateTimeUTCMinus3(nowUTCMinus3())
	_, err := s.DB.ExecContext(
		ctx,
		s.bind(`INSERT INTO audit_log (username, action, entity, entity_id, before_state, after_state, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`),
		r.Username, r.Action, r.Entity, r.EntityID, r.BeforeState, r.AfterState, now,
	)
	return err
}

func (s *Store) SearchAuditLogs(ctx context.Context, query string, limit int) ([]AuditLogRecord, error) {
	if limit <= 0 {
		limit = 500
	}
	q := strings.TrimSpace(query)
	cols := `id,
COALESCE(CAST(username AS TEXT), '') AS username,
COALESCE(CAST(action AS TEXT), '') AS action,
COALESCE(CAST(entity AS TEXT), '') AS entity,
COALESCE(CAST(entity_id AS TEXT), '') AS entity_id,
COALESCE(CAST(before_state AS TEXT), '') AS before_state,
COALESCE(CAST(after_state AS TEXT), '') AS after_state,
COALESCE(CAST(created_at AS TEXT), '') AS created_at`

	var rows *sql.Rows
	var err error
	if q == "" {
		rows, err = s.DB.QueryContext(ctx, s.bind("SELECT "+cols+" FROM audit_log ORDER BY id DESC LIMIT ?"), limit)
	} else {
		likeOp := "LIKE"
		if s.driver == "pgx" {
			likeOp = "ILIKE"
		}
		like := "%" + q + "%"
		rows, err = s.DB.QueryContext(
			ctx,
			s.bind("SELECT "+cols+" FROM audit_log WHERE CAST(COALESCE(username,'') AS TEXT) "+likeOp+" ? OR CAST(COALESCE(action,'') AS TEXT) "+likeOp+" ? OR CAST(COALESCE(entity,'') AS TEXT) "+likeOp+" ? OR CAST(COALESCE(entity_id,'') AS TEXT) "+likeOp+" ? ORDER BY id DESC LIMIT ?"),
			like, like, like, like, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditLogRecord
	for rows.Next() {
		var r AuditLogRecord
		if err := rows.Scan(&r.ID, &r.Username, &r.Action, &r.Entity, &r.EntityID, &r.BeforeState, &r.AfterState, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) SaveAppSession(ctx context.Context, sess *AppSessionRecord) error {
	perms, _ := json.Marshal(sess.Permissions)
	query := `
INSERT OR REPLACE INTO app_sessions (token, user_id, username, full_name, cpf, branch, role, permissions, authenticated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	if s.driver == "pgx" {
		query = `
INSERT INTO app_sessions (token, user_id, username, full_name, cpf, branch, role, permissions, authenticated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (token) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  username = EXCLUDED.username,
  full_name = EXCLUDED.full_name,
  cpf = EXCLUDED.cpf,
  branch = EXCLUDED.branch,
  role = EXCLUDED.role,
  permissions = EXCLUDED.permissions,
  authenticated_at = EXCLUDED.authenticated_at
`
	}
	_, err := s.DB.ExecContext(ctx, s.bind(query), sess.Token, sess.UserID, sess.Username, sess.DisplayName, sess.CPF.String, sess.Filial.String, sess.Role, string(perms), formatDateTimeUTCMinus3(sess.AuthenticatedAt))
	return err
}

func (s *Store) GetAppSession(ctx context.Context, token string) (*AppSessionRecord, error) {
	var r AppSessionRecord
	var perms string
	var authAt string
	err := s.DB.QueryRowContext(ctx, s.bind(`
SELECT token, user_id, username, full_name, cpf, branch, role, permissions, authenticated_at
FROM app_sessions WHERE token = ?
`), token).Scan(&r.Token, &r.UserID, &r.Username, &r.DisplayName, &r.CPF.String, &r.Filial.String, &r.Role, &perms, &authAt)
	if err != nil {
		return nil, err
	}
	r.CPF.Valid = r.CPF.String != ""
	r.Filial.Valid = r.Filial.String != ""
	_ = json.Unmarshal([]byte(perms), &r.Permissions)
	if t, ok := parseSessionTimestamp(authAt); ok {
		r.AuthenticatedAt = t
	}
	return &r, nil
}

func parseSessionTimestamp(raw string) (time.Time, bool) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
		time.RFC3339Nano,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, v, utcMinus3Loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func revealUserPassword(u *User) error {
	if u == nil {
		return nil
	}
	plain, err := decryptAtRest(u.Senha)
	if err != nil {
		return err
	}
	u.Senha = plain
	return nil
}

func revealAuthPassword(r *AuthRecord) error {
	if r == nil {
		return nil
	}
	plain, err := decryptAtRest(r.Senha)
	if err != nil {
		return err
	}
	r.Senha = plain
	return nil
}

func (s *Store) DeleteAppSession(ctx context.Context, token string) error {
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM app_sessions WHERE token = ?"), token)
	return err
}

func (s *Store) DeleteAppSessionsByRole(ctx context.Context, role string) error {
	_, err := s.DB.ExecContext(ctx, s.bind("DELETE FROM app_sessions WHERE role = ?"), role)
	return err
}

func (s *Store) BackupSQL(ctx context.Context, w io.Writer) error {
	tables := LegacyTableNames()

	fmt.Fprintf(w, "-- Honda Go Backup SQL Generated at %s\n", formatDateTimeUTCMinus3(nowUTCMinus3()))
	fmt.Fprintf(w, "PRAGMA foreign_keys = OFF;\n\n")

	// Deletes in reverse order to avoid FK issues
	for i := len(tables) - 1; i >= 0; i-- {
		fmt.Fprintf(w, "DELETE FROM %s;\n", quoteIdent(tables[i]))
	}
	fmt.Fprintf(w, "\n")

	for _, tableName := range tables {
		cols, err := s.tableColumns(ctx, tableName)
		if err != nil {
			return err
		}
		if len(cols) == 0 {
			continue
		}

		colList := ""
		for i, c := range cols {
			if i > 0 {
				colList += ", "
			}
			colList += quoteIdent(c)
		}

		rows, err := s.DB.QueryContext(ctx, "SELECT "+colList+" FROM "+quoteIdent(tableName))
		if err != nil {
			return err
		}

		for rows.Next() {
			vals := make([]any, len(cols))
			valPtrs := make([]any, len(cols))
			for i := range vals {
				valPtrs[i] = &vals[i]
			}

			if err := rows.Scan(valPtrs...); err != nil {
				rows.Close()
				return err
			}

			valStrings := make([]string, len(vals))
			for i, v := range vals {
				valStrings[i] = formatSQLValue(v)
			}

			fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES (%s);\n", quoteIdent(tableName), colList, strings.Join(valStrings, ", "))
		}
		rows.Close()
		fmt.Fprintf(w, "\n")
	}

	fmt.Fprintf(w, "PRAGMA foreign_keys = ON;\n")
	return nil
}

func formatSQLValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	case []byte:
		return "'" + strings.ReplaceAll(string(val), "'", "''") + "'"
	case time.Time:
		return "'" + val.Format("2006-01-02 15:04:05") + "'"
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	default:
		// Fallback safe enough for numbers and basics
		s := fmt.Sprintf("%v", val)
		if s == "<nil>" {
			return "NULL"
		}
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
}

func (s *Store) RestoreSQL(ctx context.Context, r io.Reader) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	scanner := bufio.NewScanner(r)
	const maxCapacity = 2 * 1024 * 1024 // 2MB per statement line
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		// Skip SQLite specific pragmas if we want to be postgres-friendly,
		// but since this is for restore in SQLite, we keep them or handle them.
		if strings.HasPrefix(strings.ToUpper(line), "PRAGMA") {
			// We can execute them, they won't hurt in SQLite.
		}

		if _, err := tx.ExecContext(ctx, line); err != nil {
			return fmt.Errorf("erro na linha %q: %w", line, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return tx.Commit()
}
