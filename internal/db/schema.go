package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type tableSchema struct {
	Name         string
	CreateFormat string
	Columns      []string
}

func legacySchemas() []tableSchema {
	return []tableSchema{
		{
			Name:         "roles",
			CreateFormat: `CREATE TABLE %s ("id" INTEGER NOT NULL PRIMARY KEY, "name" VARCHAR(64) NOT NULL UNIQUE, "description" TEXT)`,
			Columns:      []string{"id", "name", "description"},
		},
		{
			Name:         "role_permissions",
			CreateFormat: `CREATE TABLE %s ("role_id" INTEGER NOT NULL, "permission_key" VARCHAR(255) NOT NULL, PRIMARY KEY ("role_id", "permission_key"), FOREIGN KEY ("role_id") REFERENCES roles(id) ON DELETE CASCADE)`,
			Columns:      []string{"role_id", "permission_key"},
		},
		{
			Name:         "users",
			CreateFormat: `CREATE TABLE %s ("id" INTEGER NOT NULL PRIMARY KEY, "username" VARCHAR(255) NOT NULL, "password_hash" VARCHAR(255) NOT NULL, "full_name" VARCHAR(255) NOT NULL DEFAULT '', "manager" VARCHAR(255), "supervisor" VARCHAR(255), "cpf" VARCHAR(32), "branch" VARCHAR(64), "email" VARCHAR(255), "phone" VARCHAR(20), "role" VARCHAR(64) NOT NULL DEFAULT 'operador', "is_active" INTEGER NOT NULL DEFAULT 1, "failed_login_attempts" INTEGER NOT NULL DEFAULT 0, "mfa_enabled" INTEGER NOT NULL DEFAULT 0, "mfa_secret" VARCHAR(255), "locked_until" DATETIME, "last_login_at" DATETIME, "must_change_password" INTEGER NOT NULL DEFAULT 0, "password_changed_at" DATETIME, "temp_password_issued_at" DATETIME, "updated_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, "created_at" DATETIME NOT NULL)`,
			Columns:      []string{"id", "username", "password_hash", "full_name", "manager", "supervisor", "cpf", "branch", "email", "phone", "role", "is_active", "failed_login_attempts", "mfa_enabled", "mfa_secret", "locked_until", "last_login_at", "must_change_password", "password_changed_at", "temp_password_issued_at", "updated_at", "created_at"},
		},
		{
			Name:         "app_settings",
			CreateFormat: `CREATE TABLE %s ("key" VARCHAR(128) NOT NULL PRIMARY KEY, "value" TEXT, "updated_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
			Columns:      []string{"key", "value", "updated_at"},
		},
		{
			Name:         "api_accounts",
			CreateFormat: `CREATE TABLE %s ("id" INTEGER NOT NULL PRIMARY KEY, "user_id" INTEGER, "cpf" VARCHAR(255) NOT NULL, "company_code" VARCHAR(255) NOT NULL, "account_user" VARCHAR(255) NOT NULL, "dealer_code" VARCHAR(255) NOT NULL, "account_password" VARCHAR(255) NOT NULL, "token" VARCHAR(255), "b3_token" VARCHAR(255), "last_request_at" DATETIME NOT NULL, "cooldown_until" DATETIME, "in_flight" INTEGER DEFAULT 0, "error_401_count" INTEGER DEFAULT 0, "error_429_count" INTEGER DEFAULT 0, "blocked_until" DATETIME, "priority_score" REAL DEFAULT 0, "created_at" DATETIME, "updated_at" DATETIME, FOREIGN KEY (user_id) REFERENCES users(id))`,
			Columns:      []string{"id", "user_id", "cpf", "company_code", "account_user", "dealer_code", "account_password", "token", "b3_token", "last_request_at", "cooldown_until", "in_flight", "error_401_count", "error_429_count", "blocked_until", "priority_score", "created_at", "updated_at"},
		},
		{
			Name:         "vendor_identity_map",
			CreateFormat: `CREATE TABLE %s ("id" INTEGER NOT NULL PRIMARY KEY, "user_id" INTEGER, "cpf" VARCHAR(32), "seller_name" VARCHAR(255) NOT NULL, "branch" VARCHAR(64), "source" VARCHAR(32) NOT NULL DEFAULT 'legacy', "is_active" INTEGER NOT NULL DEFAULT 1, "created_at" DATETIME, "updated_at" DATETIME, FOREIGN KEY (user_id) REFERENCES users(id))`,
			Columns:      []string{"id", "user_id", "cpf", "seller_name", "branch", "source", "is_active", "created_at", "updated_at"},
		},
		{
			Name:         "reservations",
			CreateFormat: `CREATE TABLE %s ("id" INTEGER NOT NULL PRIMARY KEY, "request_id" INTEGER, "reserved_by" VARCHAR(255) NOT NULL, "person_document" VARCHAR(255) NOT NULL, "group_code" VARCHAR(255) NOT NULL, "quota_rd" VARCHAR(255) NOT NULL, "model_name" VARCHAR(255) NOT NULL, "replacement_quota_id" VARCHAR(255) NOT NULL, "created_at" DATETIME, FOREIGN KEY (request_id) REFERENCES requests(id))`,
			Columns:      []string{"id", "request_id", "reserved_by", "person_document", "group_code", "quota_rd", "model_name", "replacement_quota_id", "created_at"},
		},
		{
			Name: "requests",
			CreateFormat: `CREATE TABLE %s (
				"id" INTEGER NOT NULL PRIMARY KEY,
				"requester_user_id" INTEGER,
				"vendor_identity_id" INTEGER,
				"api_account_id" INTEGER,
				"requested_at" DATETIME,
				"branch" VARCHAR(255),
				"seller_name" VARCHAR(255),
				"cpf" VARCHAR(32),
				"model_name" VARCHAR(255),
				"licensed" VARCHAR(255),
				"installments" INTEGER,
				"bid_percent" REAL,
				"with_restriction" VARCHAR(16),
				"group_code" INTEGER,
				"notes" TEXT,
				"requested_quota_id" INTEGER,
				"served_group" INTEGER,
				"installments_served" INTEGER,
				"bid_percent_served" REAL,
				"quota_rd" VARCHAR(32),
				"served_at" DATETIME,
				"status" VARCHAR(64),
				"contemplation_bid" VARCHAR(64),
				"requested_date" DATE,
				"requested_time" TIME,
				"served_date" DATE,
				"served_time" TIME,
				FOREIGN KEY (requester_user_id) REFERENCES users(id),
				FOREIGN KEY (vendor_identity_id) REFERENCES vendor_identity_map(id),
				FOREIGN KEY (api_account_id) REFERENCES api_accounts(id)
			)`,
			Columns: []string{
				"id",
				"requester_user_id",
				"vendor_identity_id",
				"api_account_id",
				"requested_at",
				"branch",
				"seller_name",
				"cpf",
				"model_name",
				"licensed",
				"installments",
				"bid_percent",
				"with_restriction",
				"group_code",
				"notes",
				"requested_quota_id",
				"served_group",
				"installments_served",
				"bid_percent_served",
				"quota_rd",
				"served_at",
				"status",
				"contemplation_bid",
				"requested_date",
				"requested_time",
				"served_date",
				"served_time",
			},
		},
		{
			Name:         "available_group_ids",
			CreateFormat: `CREATE TABLE %s ("id" INTEGER NOT NULL PRIMARY KEY, "group_api_id" INTEGER NOT NULL, "products" VARCHAR(255) NOT NULL, "due_day" INTEGER NOT NULL, "term_months" INTEGER NOT NULL, "group_kind" VARCHAR(255) NOT NULL, "group_code" INTEGER NOT NULL, "quota" INTEGER NOT NULL, "r" INTEGER NOT NULL, "d" INTEGER NOT NULL, "booked" INTEGER NOT NULL, "created_at" DATETIME, "participants" INTEGER NOT NULL, "failed" INTEGER NOT NULL)`,
			Columns:      []string{"id", "group_api_id", "products", "due_day", "term_months", "group_kind", "group_code", "quota", "r", "d", "booked", "created_at", "participants", "failed"},
		},
		{
			Name:         "models",
			CreateFormat: `CREATE TABLE %s ("id" INTEGER NOT NULL PRIMARY KEY, "model_api_id" INTEGER NOT NULL, "model_name" VARCHAR(255) NOT NULL, "status" VARCHAR(64) NOT NULL)`,
			Columns:      []string{"id", "model_api_id", "model_name", "status"},
		},
		{
			Name:         "products",
			CreateFormat: `CREATE TABLE %s ("id" INTEGER NOT NULL PRIMARY KEY, "product_api_id" INTEGER NOT NULL, "products" VARCHAR(255) NOT NULL, "status" VARCHAR(64) NOT NULL)`,
			Columns:      []string{"id", "product_api_id", "products", "status"},
		},
		{
			Name: "active_groups",
			CreateFormat: `CREATE TABLE %s (
				"id" INTEGER NOT NULL PRIMARY KEY,
				"group_code" INTEGER NOT NULL,
				"due_day" INTEGER NOT NULL,
				"participants_count" INTEGER,
				"first_assembly_date" DATE,
				"plan" VARCHAR(64),
				"term_months" INTEGER,
				"group_type" VARCHAR(16),
				"models" TEXT,
				"status" VARCHAR(32) NOT NULL DEFAULT 'is_active',
				"created_at" DATETIME,
				"updated_at" DATETIME
			)`,
			Columns: []string{
				"id",
				"group_code",
				"due_day",
				"participants_count",
				"first_assembly_date",
				"plan",
				"term_months",
				"group_type",
				"models",
				"status",
				"created_at",
				"updated_at",
			},
		},
		{
			Name: "holidays",
			CreateFormat: `CREATE TABLE %s (
				"id" INTEGER NOT NULL PRIMARY KEY,
				"holiday_date" DATE NOT NULL,
				"description" VARCHAR(255) NOT NULL,
				"holiday_type" VARCHAR(32) NOT NULL DEFAULT 'Nacional',
				"is_active" INTEGER NOT NULL DEFAULT 1,
				"created_at" DATETIME,
				"updated_at" DATETIME
			)`,
			Columns: []string{
				"id",
				"holiday_date",
				"description",
				"holiday_type",
				"is_active",
				"created_at",
				"updated_at",
			},
		},
		{
			Name: "assemblies",
			CreateFormat: `CREATE TABLE %s (
				"id" INTEGER NOT NULL PRIMARY KEY,
				"quota_rd" VARCHAR(32),
				"contemplation_date" DATETIME,
				"contemplation_type" VARCHAR(255),
				"disqualification_date" DATETIME,
				"client_name" VARCHAR(255),
				"bid_percent" REAL,
				"seller_name" VARCHAR(255),
				"group_code" INTEGER,
				"federal_lottery" INTEGER,
				"group_quota_rd" VARCHAR(64) GENERATED ALWAYS AS (
					CASE
						WHEN TRIM(COALESCE(CAST(group_code AS TEXT), '')) = '' AND TRIM(COALESCE(quota_rd, '')) = '' THEN ''
						WHEN TRIM(COALESCE(CAST(group_code AS TEXT), '')) = '' THEN TRIM(COALESCE(quota_rd, ''))
						WHEN TRIM(COALESCE(quota_rd, '')) = '' THEN TRIM(COALESCE(CAST(group_code AS TEXT), ''))
						ELSE TRIM(COALESCE(CAST(group_code AS TEXT), '')) || '-' || TRIM(COALESCE(quota_rd, ''))
					END
				) VIRTUAL
			)`,
			Columns: []string{
				"id",
				"quota_rd",
				"contemplation_date",
				"contemplation_type",
				"disqualification_date",
				"client_name",
				"bid_percent",
				"seller_name",
				"group_code",
				"federal_lottery",
				"group_quota_rd",
			},
		},
		{
			Name: "manual_notifications",
			CreateFormat: `CREATE TABLE %s (
				"id" INTEGER NOT NULL PRIMARY KEY,
				"request_id" INTEGER,
				"cpf" VARCHAR(32),
				"seller_name" VARCHAR(255),
				"branch" VARCHAR(64),
				"channel" VARCHAR(32) NOT NULL DEFAULT 'whatsapp',
				"message" TEXT NOT NULL,
				"status" VARCHAR(32) NOT NULL DEFAULT 'pendente',
				"copied_at" DATETIME,
				"sent_at" DATETIME,
				"action_user" VARCHAR(255),
				"created_at" DATETIME,
				"updated_at" DATETIME,
				FOREIGN KEY (request_id) REFERENCES requests(id)
			)`,
			Columns: []string{
				"id",
				"request_id",
				"cpf",
				"seller_name",
				"branch",
				"channel",
				"message",
				"status",
				"copied_at",
				"sent_at",
				"action_user",
				"created_at",
				"updated_at",
			},
		},
		{
			Name: "audit_log",
			CreateFormat: `CREATE TABLE %s (
				"id" INTEGER NOT NULL PRIMARY KEY,
				"username" VARCHAR(255) NOT NULL,
				"action" VARCHAR(100) NOT NULL,
				"entity" VARCHAR(100),
				"entity_id" VARCHAR(100),
				"before_state" TEXT,
				"after_state" TEXT,
				"created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`,
			Columns: []string{"id", "username", "action", "entity", "entity_id", "before_state", "after_state", "created_at"},
		},
		{
			Name: "app_sessions",
			CreateFormat: `CREATE TABLE %s (
				"token" VARCHAR(64) NOT NULL PRIMARY KEY,
				"user_id" INTEGER NOT NULL,
				"username" VARCHAR(255) NOT NULL,
				"full_name" VARCHAR(255) NOT NULL,
				"cpf" VARCHAR(32),
				"branch" VARCHAR(64),
				"role" VARCHAR(64) NOT NULL,
				"permissions" TEXT,
				"authenticated_at" DATETIME NOT NULL,
				FOREIGN KEY ("user_id") REFERENCES users(id) ON DELETE CASCADE
			)`,
			Columns: []string{"token", "user_id", "username", "full_name", "cpf", "branch", "role", "permissions", "authenticated_at"},
		},
	}
}

// LegacyTableNames returns the canonical legacy table list in creation order.
func LegacyTableNames() []string {
	schemas := legacySchemas()
	out := make([]string, 0, len(schemas))
	for _, schema := range schemas {
		out = append(out, schema.Name)
	}
	return out
}

// EnsureLegacySchema guarantees all legacy tables/columns exist in the expected order.
// If an existing table diverges, it rebuilds the table preserving common-column holiday_date.
func (s *Store) EnsureLegacySchema(ctx context.Context) error {
	// Legacy auto-repair path is SQLite-oriented (sqlite_master/PRAGMA/rebuild table).
	// In Postgres, schema is managed by SQL migrations/bootstrap scripts.
	if s != nil && s.driver == "pgx" {
		if err := s.ensurePostgresSchemaBootstrap(ctx); err != nil {
			return err
		}
		if err := s.ensurePostgresDefaultRolePermissions(ctx); err != nil {
			return err
		}
		return s.ensurePostgresIDIntegrity(ctx)
	}

	if err := s.ensureSolicitacoesLicenciadoColumn(ctx); err != nil {
		return err
	}

	for _, schema := range legacySchemas() {
		exists, err := s.tableExists(ctx, schema.Name)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := s.DB.ExecContext(ctx, fmt.Sprintf(schema.CreateFormat, quoteIdent(schema.Name))); err != nil {
				return fmt.Errorf("create table %s: %w", schema.Name, err)
			}
			continue
		}

		actual, err := s.tableColumns(ctx, schema.Name)
		if err != nil {
			return err
		}
		if equalColumns(actual, schema.Columns) {
			continue
		}
		if err := s.rebuildTableToSchema(ctx, schema, actual); err != nil {
			return fmt.Errorf("rebuild table %s: %w", schema.Name, err)
		}
	}
	if err := s.ensureAppUserIndexes(ctx); err != nil {
		return err
	}
	if err := s.ensureOperationalIndexes(ctx); err != nil {
		return err
	}
	if err := s.ensureNationalHolidaysSeed(ctx); err != nil {
		return err
	}
	if err := s.ensureRolesSeed(ctx); err != nil {
		return err
	}
	if err := s.backfillSolicitacoesDateParts(ctx); err != nil {
		return err
	}
	if err := s.backfillCotasReservadasCreatedAt(ctx); err != nil {
		return err
	}
	if err := s.normalizeCotasReservadasDocumento(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensurePostgresSchemaBootstrap(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS roles (
			id BIGINT PRIMARY KEY,
			name VARCHAR(64) NOT NULL UNIQUE,
			description TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			username VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			full_name VARCHAR(255) NOT NULL DEFAULT '',
			manager VARCHAR(255),
			supervisor VARCHAR(255),
			cpf VARCHAR(32),
			branch VARCHAR(64),
			email VARCHAR(255),
			phone VARCHAR(20),
			role VARCHAR(64) NOT NULL DEFAULT 'operador',
			is_active SMALLINT NOT NULL DEFAULT 1,
			failed_login_attempts INTEGER NOT NULL DEFAULT 0,
			mfa_enabled SMALLINT NOT NULL DEFAULT 0,
			mfa_secret VARCHAR(255),
			locked_until TIMESTAMP,
			last_login_at TIMESTAMP,
			must_change_password SMALLINT NOT NULL DEFAULT 0,
			password_changed_at TIMESTAMP,
			temp_password_issued_at TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key VARCHAR(128) PRIMARY KEY,
			value TEXT,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS role_permissions (
			role_id BIGINT NOT NULL,
			permission_key VARCHAR(255) NOT NULL,
			PRIMARY KEY (role_id, permission_key)
		)`,
		`CREATE TABLE IF NOT EXISTS api_accounts (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			user_id BIGINT,
			cpf VARCHAR(255) NOT NULL,
			company_code VARCHAR(255) NOT NULL,
			account_user VARCHAR(255) NOT NULL,
			dealer_code VARCHAR(255) NOT NULL,
			account_password VARCHAR(255) NOT NULL,
			token VARCHAR(255),
			b3_token VARCHAR(255),
			last_request_at TIMESTAMP NOT NULL,
			cooldown_until TIMESTAMP,
			in_flight INTEGER DEFAULT 0,
			error_401_count INTEGER DEFAULT 0,
			error_429_count INTEGER DEFAULT 0,
			blocked_until TIMESTAMP,
			priority_score DOUBLE PRECISION DEFAULT 0,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS vendor_identity_map (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			user_id BIGINT,
			cpf VARCHAR(32),
			seller_name VARCHAR(255) NOT NULL,
			branch VARCHAR(64),
			source VARCHAR(32) NOT NULL DEFAULT 'legacy',
			is_active SMALLINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS requests (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			requester_user_id BIGINT,
			vendor_identity_id BIGINT,
			api_account_id BIGINT,
			requested_at TIMESTAMP,
			branch VARCHAR(255),
			seller_name VARCHAR(255),
			cpf VARCHAR(32),
			model_name VARCHAR(255),
			licensed VARCHAR(255),
			installments INTEGER,
			bid_percent DOUBLE PRECISION,
			with_restriction VARCHAR(16),
			group_code INTEGER,
			notes TEXT,
			requested_quota_id BIGINT,
			served_group BIGINT,
			installments_served INTEGER,
			bid_percent_served DOUBLE PRECISION,
			quota_rd VARCHAR(32),
			served_at TIMESTAMP,
			status VARCHAR(64),
			contemplation_bid VARCHAR(64),
			requested_date DATE,
			requested_time TIME,
			served_date DATE,
			served_time TIME
		)`,
		`CREATE TABLE IF NOT EXISTS reservations (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			request_id BIGINT,
			reserved_by VARCHAR(255) NOT NULL,
			person_document VARCHAR(255) NOT NULL,
			group_code VARCHAR(255) NOT NULL,
			quota_rd VARCHAR(255) NOT NULL,
			model_name VARCHAR(255) NOT NULL,
			replacement_quota_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS available_group_ids (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			group_api_id INTEGER NOT NULL,
			products VARCHAR(255) NOT NULL,
			due_day INTEGER NOT NULL,
			term_months INTEGER NOT NULL,
			group_kind VARCHAR(255) NOT NULL,
			group_code INTEGER NOT NULL,
			quota INTEGER NOT NULL,
			r INTEGER NOT NULL,
			d INTEGER NOT NULL,
			booked INTEGER NOT NULL,
			created_at TIMESTAMP,
			participants INTEGER NOT NULL,
			failed INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS models (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			model_api_id INTEGER NOT NULL,
			model_name VARCHAR(255) NOT NULL,
			status VARCHAR(64) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS products (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			product_api_id INTEGER NOT NULL,
			products VARCHAR(255) NOT NULL,
			status VARCHAR(64) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS active_groups (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			group_code INTEGER NOT NULL,
			due_day INTEGER NOT NULL,
			participants_count INTEGER,
			first_assembly_date DATE,
			plan VARCHAR(64),
			term_months INTEGER,
			group_type VARCHAR(16),
			models TEXT,
			status VARCHAR(32) NOT NULL DEFAULT 'is_active',
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS holidays (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			holiday_date DATE NOT NULL,
			description VARCHAR(255) NOT NULL,
			holiday_type VARCHAR(32) NOT NULL DEFAULT 'Nacional',
			is_active SMALLINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS assemblies (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			quota_rd VARCHAR(32),
			contemplation_date TIMESTAMP,
			contemplation_type VARCHAR(255),
			disqualification_date TIMESTAMP,
			client_name VARCHAR(255),
			bid_percent DOUBLE PRECISION,
			seller_name VARCHAR(255),
			group_code INTEGER,
			federal_lottery INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS manual_notifications (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			request_id BIGINT,
			cpf VARCHAR(32),
			seller_name VARCHAR(255),
			branch VARCHAR(64),
			channel VARCHAR(32) NOT NULL DEFAULT 'whatsapp',
			message TEXT NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'pendente',
			copied_at TIMESTAMP,
			sent_at TIMESTAMP,
			action_user VARCHAR(255),
			created_at TIMESTAMP,
			updated_at TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_log (
			id BIGINT PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
			username VARCHAR(255) NOT NULL,
			action VARCHAR(100) NOT NULL,
			entity VARCHAR(100),
			entity_id VARCHAR(100),
			before_state TEXT,
			after_state TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS app_sessions (
			token VARCHAR(64) PRIMARY KEY,
			user_id BIGINT NOT NULL,
			username VARCHAR(255) NOT NULL,
			full_name VARCHAR(255) NOT NULL,
			cpf VARCHAR(32),
			branch VARCHAR(64),
			role VARCHAR(64) NOT NULL,
			permissions TEXT,
			authenticated_at TIMESTAMP NOT NULL
		)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS supervisor VARCHAR(255)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS manager VARCHAR(255)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS must_change_password SMALLINT NOT NULL DEFAULT 0`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMP`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS temp_password_issued_at TIMESTAMP`,
		`ALTER TABLE requests ADD COLUMN IF NOT EXISTS installments_served INTEGER`,
		`ALTER TABLE requests ADD COLUMN IF NOT EXISTS bid_percent_served DOUBLE PRECISION`,
		`ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS fk_role_permissions_role`,
		`ALTER TABLE role_permissions ADD CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE`,
		`ALTER TABLE api_accounts DROP CONSTRAINT IF EXISTS fk_api_accounts_user`,
		`ALTER TABLE api_accounts ADD CONSTRAINT fk_api_accounts_user FOREIGN KEY (user_id) REFERENCES users(id)`,
		`ALTER TABLE vendor_identity_map DROP CONSTRAINT IF EXISTS fk_vendor_identity_user`,
		`ALTER TABLE vendor_identity_map ADD CONSTRAINT fk_vendor_identity_user FOREIGN KEY (user_id) REFERENCES users(id)`,
		`ALTER TABLE requests DROP CONSTRAINT IF EXISTS fk_requests_requester_user`,
		`ALTER TABLE requests ADD CONSTRAINT fk_requests_requester_user FOREIGN KEY (requester_user_id) REFERENCES users(id)`,
		`ALTER TABLE requests DROP CONSTRAINT IF EXISTS fk_requests_vendor_identity`,
		`ALTER TABLE requests ADD CONSTRAINT fk_requests_vendor_identity FOREIGN KEY (vendor_identity_id) REFERENCES vendor_identity_map(id)`,
		`ALTER TABLE requests DROP CONSTRAINT IF EXISTS fk_requests_api_account`,
		`ALTER TABLE requests ADD CONSTRAINT fk_requests_api_account FOREIGN KEY (api_account_id) REFERENCES api_accounts(id)`,
		`ALTER TABLE reservations DROP CONSTRAINT IF EXISTS fk_reservations_request`,
		`ALTER TABLE reservations ADD CONSTRAINT fk_reservations_request FOREIGN KEY (request_id) REFERENCES requests(id)`,
		`ALTER TABLE manual_notifications DROP CONSTRAINT IF EXISTS fk_manual_notifications_request`,
		`ALTER TABLE manual_notifications ADD CONSTRAINT fk_manual_notifications_request FOREIGN KEY (request_id) REFERENCES requests(id)`,
		`ALTER TABLE app_sessions DROP CONSTRAINT IF EXISTS fk_app_sessions_user`,
		`ALTER TABLE app_sessions ADD CONSTRAINT fk_app_sessions_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE`,
		`CREATE INDEX IF NOT EXISTS idx_appuser_username ON users(username)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_appuser_cpf ON users(cpf)`,
		`CREATE INDEX IF NOT EXISTS idx_appuser_filial ON users(branch)`,
		`CREATE INDEX IF NOT EXISTS idx_appuser_supervisor ON users(supervisor)`,
		`CREATE INDEX IF NOT EXISTS idx_appuser_manager ON users(manager)`,
		`CREATE INDEX IF NOT EXISTS idx_appuser_role ON users(role)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_roles_name ON roles(name)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS ux_app_sessions_token ON app_sessions(token)`,
		`CREATE INDEX IF NOT EXISTS idx_gruposativos_grupo ON active_groups(group_code)`,
		`CREATE INDEX IF NOT EXISTS idx_gruposativos_vencimento ON active_groups(due_day)`,
		`CREATE INDEX IF NOT EXISTS idx_gruposativos_status ON active_groups(status)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_feriados_data_tipo ON holidays(holiday_date, holiday_type)`,
		`CREATE INDEX IF NOT EXISTS idx_feriados_ativo ON holidays(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_msgnotif_status_created ON manual_notifications(status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_msgnotif_request ON manual_notifications(request_id)`,
		`CREATE INDEX IF NOT EXISTS idx_msgnotif_cpf ON manual_notifications(cpf)`,
		`CREATE INDEX IF NOT EXISTS idx_api_accounts_user ON api_accounts(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vendor_identity_user ON vendor_identity_map(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vendor_identity_cpf ON vendor_identity_map(cpf)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_requester ON requests(requester_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_vendor_identity ON requests(vendor_identity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_api_account ON requests(api_account_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reservations_request ON reservations(request_id)`,
		`INSERT INTO roles (id, name, description) VALUES (1, 'admin', 'Administrador Global do Sistema') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO roles (id, name, description) VALUES (2, 'operador', 'Operador de Vendas e Reservas') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO roles (id, name, description) VALUES (3, 'vendedor', 'Vendedor') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO roles (id, name, description) VALUES (4, 'viewer', 'Somente Leitura') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO roles (id, name, description) VALUES (5, 'supervisor', 'Supervisor de Equipe') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO roles (id, name, description) VALUES (6, 'gerente', 'Gerente de Filial') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO roles (id, name, description) VALUES (7, 'super_usuario', 'Super Usuário (sem operações destrutivas)') ON CONFLICT (id) DO NOTHING`,
		`INSERT INTO role_permissions (role_id, permission_key) VALUES
			(1,'dashboard:read'), (1,'solicitacoes:read'), (1,'solicitacoes:create'), (1,'solicitacoes:edit'), (1,'solicitacoes:delete'), (1,'solicitacoes:print'),
			(1,'cotas:reserve'), (1,'cotas:export'), (1,'users:read'), (1,'users:create'), (1,'users:edit'), (1,'users:delete'),
			(1,'roles:manage'), (1,'configs:manage'), (1,'logs:read'), (1,'logs:delete'), (1,'audit:view'),
			(1,'db:tables:create'), (1,'db:tables:clear'), (1,'db:tables:drop'), (1,'db:backup'), (1,'db:restore'),
			(1,'nav:dashboard'), (1,'nav:reservas'), (1,'nav:monitor'), (1,'nav:logs'), (1,'nav:config'),
			(1,'monitor:read'),
			(1,'reservas:home'), (1,'reservas:solicitacoes'), (1,'reservas:minhas'), (1,'reservas:solicitar'), (1,'reservas:reservadas'), (1,'reservas:mensagens'), (1,'reservas:config'),
			(1,'config:users'), (1,'config:appusers'), (1,'config:rbac'), (1,'config:audit'), (1,'config:database'), (1,'config:password_policy'), (1,'config:idsgrupos'), (1,'config:active_groups'), (1,'config:assemblies'), (1,'config:models'), (1,'config:produtos')
		ON CONFLICT DO NOTHING`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("postgres bootstrap schema: %w", err)
		}
	}
	return nil
}

func (s *Store) ensurePostgresDefaultRolePermissions(ctx context.Context) error {
	if s == nil || s.driver != "pgx" {
		return nil
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	vendedorPerms := []string{
		"dashboard:read", "solicitacoes:read", "solicitacoes:create", "solicitacoes:edit", "cotas:reserve",
		"nav:reservas",
		"reservas:home", "reservas:minhas", "reservas:solicitar",
	}
	superUsuarioPerms := []string{
		"dashboard:read", "solicitacoes:read", "solicitacoes:create", "solicitacoes:edit", "solicitacoes:print",
		"cotas:reserve", "cotas:export",
		"users:read", "users:create", "users:edit",
		"configs:manage", "logs:read", "audit:view", "monitor:read",
		"nav:dashboard", "nav:reservas", "nav:monitor", "nav:logs", "nav:config",
		"reservas:home", "reservas:solicitacoes", "reservas:minhas", "reservas:solicitar", "reservas:reservadas", "reservas:mensagens", "reservas:config",
		"config:users", "config:appusers", "config:audit", "config:idsgrupos", "config:active_groups", "config:assemblies", "config:models", "config:produtos",
	}
	seedRolePermsIfEmpty := func(roleID int64, perms []string) error {
		var count int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM role_permissions WHERE role_id = $1", roleID).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		for _, p := range perms {
			if _, err := tx.ExecContext(ctx, "INSERT INTO role_permissions (role_id, permission_key) VALUES ($1, $2) ON CONFLICT DO NOTHING", roleID, p); err != nil {
				return err
			}
		}
		return nil
	}

	for _, roleID := range []int64{2, 3, 4, 5, 6} {
		if err := seedRolePermsIfEmpty(roleID, vendedorPerms); err != nil {
			return err
		}
	}
	if err := seedRolePermsIfEmpty(7, superUsuarioPerms); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ensurePostgresIDIntegrity(ctx context.Context) error {
	if s == nil || s.driver != "pgx" {
		return nil
	}

	tables := []string{
		"users",
		"api_accounts",
		"vendor_identity_map",
		"requests",
		"reservations",
		"available_group_ids",
		"models",
		"products",
		"active_groups",
		"holidays",
		"assemblies",
		"manual_notifications",
		"audit_log",
	}

	for _, table := range tables {
		var exists bool
		if err := s.DB.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = $1
)`, table).Scan(&exists); err != nil {
			return fmt.Errorf("check table %s exists: %w", table, err)
		}
		if !exists {
			continue
		}

		// Prevent silent startup with duplicated IDs.
		var dupCount int64
		dupQuery := fmt.Sprintf(`
SELECT COUNT(1)
FROM (
  SELECT id
  FROM %s
  GROUP BY id
  HAVING COUNT(1) > 1
) d`, quoteIdent(table))
		if err := s.DB.QueryRowContext(ctx, dupQuery).Scan(&dupCount); err != nil {
			return fmt.Errorf("check duplicated ids on %s: %w", table, err)
		}
		if dupCount > 0 {
			return fmt.Errorf("tabela %s possui ids duplicados (%d); execute script de saneamento antes de iniciar", table, dupCount)
		}

		var hasPK bool
		if err := s.DB.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM pg_index i
  JOIN pg_class t ON t.oid = i.indrelid
  JOIN pg_namespace n ON n.oid = t.relnamespace
  JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(i.indkey)
  WHERE n.nspname = 'public'
    AND t.relname = $1
    AND i.indisprimary
    AND a.attname = 'id'
)`, table).Scan(&hasPK); err != nil {
			return fmt.Errorf("check primary key on %s.id: %w", table, err)
		}
		if !hasPK {
			if _, err := s.DB.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (id)", quoteIdent(table))); err != nil {
				return fmt.Errorf("add primary key on %s.id: %w", table, err)
			}
		}

		var seqName sql.NullString
		seqQuery := fmt.Sprintf("SELECT pg_get_serial_sequence('public.%s','id')", table)
		if err := s.DB.QueryRowContext(ctx, seqQuery).Scan(&seqName); err != nil {
			return fmt.Errorf("resolve sequence for %s.id: %w", table, err)
		}
		if seqName.Valid && strings.TrimSpace(seqName.String) != "" {
			setvalQuery := fmt.Sprintf(
				"SELECT setval($1::regclass, COALESCE((SELECT MAX(id) FROM %s), 1), true)",
				quoteIdent(table),
			)
			if _, err := s.DB.ExecContext(ctx, setvalQuery, seqName.String); err != nil {
				return fmt.Errorf("align sequence for %s.id: %w", table, err)
			}
		}
	}

	// Garantias adicionais de integridade lógica usadas no login/sessão.
	var dupUsernames int64
	if err := s.DB.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM (
  SELECT username
  FROM public.users
  WHERE COALESCE(TRIM(username), '') <> ''
  GROUP BY username
  HAVING COUNT(1) > 1
) d`).Scan(&dupUsernames); err != nil {
		return fmt.Errorf("check duplicated usernames on users: %w", err)
	}
	if dupUsernames > 0 {
		return fmt.Errorf("tabela users possui usernames duplicados (%d); execute saneamento antes de iniciar", dupUsernames)
	}

	var dupSessionTokens int64
	if err := s.DB.QueryRowContext(ctx, `
SELECT COUNT(1)
FROM (
  SELECT token
  FROM public.app_sessions
  WHERE COALESCE(TRIM(token), '') <> ''
  GROUP BY token
  HAVING COUNT(1) > 1
) d`).Scan(&dupSessionTokens); err != nil {
		return fmt.Errorf("check duplicated tokens on app_sessions: %w", err)
	}
	if dupSessionTokens > 0 {
		return fmt.Errorf("tabela app_sessions possui tokens duplicados (%d); execute saneamento antes de iniciar", dupSessionTokens)
	}

	return nil
}

func (s *Store) ensureOperationalIndexes(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_gruposativos_grupo ON active_groups(group_code)`); err != nil {
		return fmt.Errorf("create idx_gruposativos_grupo: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_gruposativos_vencimento ON active_groups(due_day)`); err != nil {
		return fmt.Errorf("create idx_gruposativos_vencimento: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_gruposativos_status ON active_groups(status)`); err != nil {
		return fmt.Errorf("create idx_gruposativos_status: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_feriados_data_tipo ON holidays(holiday_date, holiday_type)`); err != nil {
		return fmt.Errorf("create idx_feriados_data_tipo: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_feriados_ativo ON holidays(is_active)`); err != nil {
		return fmt.Errorf("create idx_feriados_ativo: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_msgnotif_status_created ON manual_notifications(status, created_at)`); err != nil {
		return fmt.Errorf("create idx_msgnotif_status_created: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_msgnotif_request ON manual_notifications(request_id)`); err != nil {
		return fmt.Errorf("create idx_msgnotif_request: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_msgnotif_cpf ON manual_notifications(cpf)`); err != nil {
		return fmt.Errorf("create idx_msgnotif_cpf: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_api_accounts_user ON api_accounts(user_id)`); err != nil {
		return fmt.Errorf("create idx_api_accounts_user: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_vendor_identity_user ON vendor_identity_map(user_id)`); err != nil {
		return fmt.Errorf("create idx_vendor_identity_user: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_vendor_identity_cpf ON vendor_identity_map(cpf)`); err != nil {
		return fmt.Errorf("create idx_vendor_identity_cpf: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_requests_requester ON requests(requester_user_id)`); err != nil {
		return fmt.Errorf("create idx_requests_requester: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_requests_vendor_identity ON requests(vendor_identity_id)`); err != nil {
		return fmt.Errorf("create idx_requests_vendor_identity: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_requests_api_account ON requests(api_account_id)`); err != nil {
		return fmt.Errorf("create idx_requests_api_account: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_reservations_request ON reservations(request_id)`); err != nil {
		return fmt.Errorf("create idx_reservations_request: %w", err)
	}
	return nil
}

func (s *Store) ensureNationalHolidaysSeed(ctx context.Context) error {
	exists, err := s.tableExists(ctx, "holidays")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	now := nowUTCMinus3()
	nowStr := formatDateTimeUTCMinus3(now)
	startYear := now.Year() - 1
	endYear := now.Year() + 5

	type holidayDef struct {
		month int
		day   int
		name  string
	}
	fixed := []holidayDef{
		{month: 1, day: 1, name: "Confraternizacao Universal"},
		{month: 4, day: 21, name: "Tiradentes"},
		{month: 5, day: 1, name: "Dia do Trabalhador"},
		{month: 9, day: 7, name: "Independencia do Brasil"},
		{month: 10, day: 12, name: "Nossa Senhora Aparecida"},
		{month: 11, day: 2, name: "Finados"},
		{month: 11, day: 15, name: "Proclamacao da Republica"},
		{month: 11, day: 20, name: "Dia da Consciencia Negra"},
		{month: 12, day: 25, name: "Natal"},
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
INSERT OR IGNORE INTO holidays (holiday_date, description, holiday_type, is_active, created_at, updated_at)
VALUES (?, ?, 'Nacional', 1, ?, ?)
`)
	if err != nil {
		return fmt.Errorf("prepare holidays seed: %w", err)
	}
	defer stmt.Close()

	for y := startYear; y <= endYear; y++ {
		for _, h := range fixed {
			dt := time.Date(y, time.Month(h.month), h.day, 0, 0, 0, 0, time.UTC)
			dateStr := dt.Format("2006-01-02")
			if _, err := stmt.ExecContext(ctx, dateStr, h.name, nowStr, nowStr); err != nil {
				return fmt.Errorf("insert feriado nacional %s/%04d: %w", h.name, y, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureRolesSeed(ctx context.Context) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Ensure default roles
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO roles (id, name, description) VALUES (1, 'admin', 'Administrador Global do Sistema')"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO roles (id, name, description) VALUES (2, 'operador', 'Operador de Vendas e Reservas')"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO roles (id, name, description) VALUES (3, 'vendedor', 'Vendedor')"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO roles (id, name, description) VALUES (4, 'viewer', 'Somente Leitura')"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO roles (id, name, description) VALUES (5, 'supervisor', 'Supervisor de Equipe')"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO roles (id, name, description) VALUES (6, 'gerente', 'Gerente de Filial')"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO roles (id, name, description) VALUES (7, 'super_usuario', 'Super Usuário (sem operações destrutivas)')"); err != nil {
		return err
	}

	// Admin permissions (example mapping)
	adminPerms := []string{
		"dashboard:read", "solicitacoes:read", "solicitacoes:create", "solicitacoes:edit", "solicitacoes:delete", "solicitacoes:print",
		"cotas:reserve", "cotas:export", "users:read", "users:create", "users:edit", "users:delete", "roles:manage", "configs:manage", "logs:read", "logs:delete", "audit:view",
		"db:tables:create", "db:tables:clear", "db:tables:drop", "db:backup", "db:restore",
		"nav:dashboard", "nav:reservas", "nav:monitor", "nav:logs", "nav:config",
		"monitor:read",
		"reservas:home", "reservas:solicitacoes", "reservas:minhas", "reservas:solicitar", "reservas:reservadas", "reservas:mensagens", "reservas:config",
		"config:users", "config:appusers", "config:rbac", "config:audit", "config:database", "config:password_policy", "config:idsgrupos", "config:active_groups", "config:assemblies", "config:models", "config:produtos",
	}
	for _, p := range adminPerms {
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO role_permissions (role_id, permission_key) VALUES (1, ?)", p); err != nil {
			return err
		}
	}

	vendedorPerms := []string{
		"dashboard:read", "solicitacoes:read", "solicitacoes:create", "solicitacoes:edit", "cotas:reserve",
		"nav:reservas",
		"reservas:home", "reservas:minhas", "reservas:solicitar",
	}
	superUsuarioPerms := []string{
		"dashboard:read", "solicitacoes:read", "solicitacoes:create", "solicitacoes:edit", "solicitacoes:print",
		"cotas:reserve", "cotas:export",
		"users:read", "users:create", "users:edit",
		"configs:manage", "logs:read", "audit:view", "monitor:read",
		"nav:dashboard", "nav:reservas", "nav:monitor", "nav:logs", "nav:config",
		"reservas:home", "reservas:solicitacoes", "reservas:minhas", "reservas:solicitar", "reservas:reservadas", "reservas:mensagens", "reservas:config",
		"config:users", "config:appusers", "config:audit", "config:idsgrupos", "config:active_groups", "config:assemblies", "config:models", "config:produtos",
	}
	seedRolePermsIfEmpty := func(roleID int64, perms []string) error {
		var count int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM role_permissions WHERE role_id = ?", roleID).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		for _, p := range perms {
			if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO role_permissions (role_id, permission_key) VALUES (?, ?)", roleID, p); err != nil {
				return err
			}
		}
		return nil
	}
	// Em instalação nova: todos não-admin iniciam com mesmo pacote do vendedor.
	for _, roleID := range []int64{2, 3, 4, 5, 6} {
		if err := seedRolePermsIfEmpty(roleID, vendedorPerms); err != nil {
			return err
		}
	}
	if err := seedRolePermsIfEmpty(7, superUsuarioPerms); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) ensureSolicitacoesLicenciadoColumn(ctx context.Context) error {
	exists, err := s.tableExists(ctx, "requests")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	cols, err := s.tableColumns(ctx, "requests")
	if err != nil {
		return err
	}
	hasPlano := false
	hasLicenciado := false
	for _, c := range cols {
		if strings.EqualFold(c, "plan") {
			hasPlano = true
		}
		if strings.EqualFold(c, "licensed") {
			hasLicenciado = true
		}
	}

	if hasPlano && !hasLicenciado {
		if _, err := s.DB.ExecContext(ctx, `ALTER TABLE requests RENAME COLUMN plan TO licensed`); err != nil {
			return fmt.Errorf("rename requests.plan -> licensed: %w", err)
		}
	}
	return nil
}

func (s *Store) ensureAppUserIndexes(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_appuser_username ON users(username)`); err != nil {
		return fmt.Errorf("create idx_appuser_username: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_appuser_cpf ON users(cpf)`); err != nil {
		return fmt.Errorf("create idx_appuser_cpf: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_appuser_filial ON users(branch)`); err != nil {
		return fmt.Errorf("create idx_appuser_filial: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_appuser_supervisor ON users(supervisor)`); err != nil {
		return fmt.Errorf("create idx_appuser_supervisor: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_appuser_manager ON users(manager)`); err != nil {
		return fmt.Errorf("create idx_appuser_manager: %w", err)
	}
	if _, err := s.DB.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_appuser_role ON users(role)`); err != nil {
		return fmt.Errorf("create idx_appuser_role: %w", err)
	}
	return nil
}

func (s *Store) backfillSolicitacoesDateParts(ctx context.Context) error {
	exists, err := s.tableExists(ctx, "requests")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	_, err = s.DB.ExecContext(ctx, `
UPDATE requests
SET
  requested_date = CASE
    WHEN requested_at IS NULL OR TRIM(requested_at) = '' THEN NULL
    WHEN requested_date IS NOT NULL AND TRIM(requested_date) <> '' THEN requested_date
    WHEN LENGTH(requested_at) >= 10 THEN SUBSTR(requested_at, 1, 10)
    ELSE NULL
  END,
  requested_time = CASE
    WHEN requested_at IS NULL OR TRIM(requested_at) = '' THEN NULL
    WHEN requested_time IS NOT NULL AND TRIM(requested_time) <> '' THEN requested_time
    WHEN LENGTH(requested_at) >= 19 THEN SUBSTR(requested_at, 12, 8)
    ELSE NULL
  END,
  served_date = CASE
    WHEN served_at IS NULL OR TRIM(served_at) = '' THEN NULL
    WHEN served_date IS NOT NULL AND TRIM(served_date) <> '' THEN served_date
    WHEN LENGTH(served_at) >= 10 THEN SUBSTR(served_at, 1, 10)
    ELSE NULL
  END,
  served_time = CASE
    WHEN served_at IS NULL OR TRIM(served_at) = '' THEN NULL
    WHEN served_time IS NOT NULL AND TRIM(served_time) <> '' THEN served_time
    WHEN LENGTH(served_at) >= 19 THEN SUBSTR(served_at, 12, 8)
    ELSE NULL
  END
WHERE
  (requested_at IS NOT NULL AND TRIM(requested_at) <> '')
  OR
  (served_at IS NOT NULL AND TRIM(served_at) <> '')
`)
	if err != nil {
		return fmt.Errorf("backfill requests date/time parts: %w", err)
	}
	return nil
}

func (s *Store) backfillCotasReservadasCreatedAt(ctx context.Context) error {
	exists, err := s.tableExists(ctx, "reservations")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	_, err = s.DB.ExecContext(ctx, `
UPDATE reservations
SET created_at = CURRENT_TIMESTAMP
WHERE created_at IS NULL OR TRIM(CAST(created_at AS TEXT)) = ''
`)
	if err != nil {
		return fmt.Errorf("backfill reservations.created_at: %w", err)
	}
	return nil
}

func (s *Store) normalizeCotasReservadasDocumento(ctx context.Context) error {
	exists, err := s.tableExists(ctx, "reservations")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	_, err = s.DB.ExecContext(ctx, `
UPDATE reservations
SET person_document = REPLACE(
  REPLACE(
    REPLACE(
      REPLACE(
        REPLACE(
          REPLACE(TRIM(COALESCE(CAST(person_document AS TEXT), '')), '.', ''),
        '-', ''),
      '/', ''),
    '(', ''),
  ')', ''),
' ', '')
WHERE person_document IS NOT NULL AND TRIM(CAST(person_document AS TEXT)) <> ''
`)
	if err != nil {
		return fmt.Errorf("normalize reservations.person_document: %w", err)
	}
	return nil
}

// ClearLegacyData deletes holiday_date from all legacy tables that exist in the database.
// It keeps table structures intact and resets SQLite autoincrement sequences.
func (s *Store) ClearLegacyData(ctx context.Context) (int, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	cleared := 0
	for _, tableName := range LegacyTableNames() {
		exists, err := s.tableExistsTx(ctx, tx, tableName)
		if err != nil {
			return cleared, err
		}
		if !exists {
			continue
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+quoteIdent(tableName)); err != nil {
			return cleared, err
		}
		cleared++
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM sqlite_sequence"); err != nil {
		// Some SQLite setups may not have sqlite_sequence yet; ignore "no such table".
		if !strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return cleared, err
		}
	}
	if err := tx.Commit(); err != nil {
		return cleared, err
	}
	return cleared, nil
}

// DropLegacyTables drops all legacy tables that exist in the database.
// Drop order is reversed to reduce dependency issues.
func (s *Store) DropLegacyTables(ctx context.Context) (int, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	names := LegacyTableNames()
	dropped := 0
	for i := len(names) - 1; i >= 0; i-- {
		tableName := names[i]
		exists, err := s.tableExistsTx(ctx, tx, tableName)
		if err != nil {
			return dropped, err
		}
		if !exists {
			continue
		}
		if _, err := tx.ExecContext(ctx, "DROP TABLE "+quoteIdent(tableName)); err != nil {
			return dropped, err
		}
		dropped++
	}
	if err := tx.Commit(); err != nil {
		return dropped, err
	}
	return dropped, nil
}

func (s *Store) tableExists(ctx context.Context, tableName string) (bool, error) {
	if s != nil && s.IsPostgres() {
		var one int
		err := s.DB.QueryRowContext(ctx, `
SELECT 1
FROM information_schema.tables
WHERE table_schema = 'public' AND table_name = $1
LIMIT 1
`, tableName).Scan(&one)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}

	var one int
	err := s.DB.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type='table' AND name=? LIMIT 1", tableName).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) tableExistsTx(ctx context.Context, tx *sql.Tx, tableName string) (bool, error) {
	if s != nil && s.IsPostgres() {
		var one int
		err := tx.QueryRowContext(ctx, `
SELECT 1
FROM information_schema.tables
WHERE table_schema = 'public' AND table_name = $1
LIMIT 1
`, tableName).Scan(&one)
		if err == sql.ErrNoRows {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return true, nil
	}

	var one int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type='table' AND name=? LIMIT 1", tableName).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) tableColumns(ctx context.Context, tableName string) ([]string, error) {
	if s != nil && s.IsPostgres() {
		rows, err := s.DB.QueryContext(ctx, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = $1
ORDER BY ordinal_position
`, tableName)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		cols := make([]string, 0, 16)
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				return nil, err
			}
			cols = append(cols, name)
		}
		return cols, rows.Err()
	}

	rows, err := s.DB.QueryContext(ctx, "PRAGMA table_info("+quoteIdent(tableName)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make([]string, 0, 16)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

func (s *Store) rebuildTableToSchema(ctx context.Context, schema tableSchema, actualCols []string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tmpName := schema.Name + "__new"
	if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+quoteIdent(tmpName)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(schema.CreateFormat, quoteIdent(tmpName))); err != nil {
		return err
	}

	existingMap := make(map[string]struct{}, len(actualCols))
	for _, col := range actualCols {
		existingMap[strings.ToLower(col)] = struct{}{}
	}
	common := make([]string, 0, len(schema.Columns))
	for _, col := range schema.Columns {
		if _, ok := existingMap[strings.ToLower(col)]; ok {
			common = append(common, col)
		}
	}
	if len(common) > 0 {
		quoted := make([]string, 0, len(common))
		for _, col := range common {
			quoted = append(quoted, quoteIdent(col))
		}
		colList := strings.Join(quoted, ",")
		insertSQL := "INSERT INTO " + quoteIdent(tmpName) + " (" + colList + ") SELECT " + colList + " FROM " + quoteIdent(schema.Name)
		if _, err := tx.ExecContext(ctx, insertSQL); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, "DROP TABLE "+quoteIdent(schema.Name)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "ALTER TABLE "+quoteIdent(tmpName)+" RENAME TO "+quoteIdent(schema.Name)); err != nil {
		return err
	}
	return tx.Commit()
}

func equalColumns(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range actual {
		if !strings.EqualFold(actual[i], expected[i]) {
			return false
		}
	}
	return true
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
