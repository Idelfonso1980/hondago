-- Migração SQLite -> PostgreSQL (via CSV)
-- Pré-requisito:
-- 1) Executar 001_init.sql e 002_seed.sql antes deste arquivo.
-- 2) Exportar CSVs do SQLite para a pasta definida em :csv_dir.
-- 3) Rodar no psql com:
--    \set csv_dir 'C:/temp/hondago_csv'
--    \i docs/sql/postgres/003_migrate_from_sqlite.sql

\echo 'Usando CSV em /tmp/hondago_csv'

BEGIN;
SET LOCAL datestyle TO 'ISO, DMY';

-- Ordem para evitar violação de FK
TRUNCATE TABLE
  app_sessions,
  audit_log,
  manual_notifications,
  assemblies,
  holidays,
  active_groups,
  products,
  models,
  available_group_ids,
  reservations,
  requests,
  vendor_identity_map,
  api_accounts,
  role_permissions,
  users,
  roles
RESTART IDENTITY CASCADE;

-- 1) Base de segurança e usuários
\copy roles (id, name, description) FROM '/tmp/hondago_csv/roles.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
\copy users (id, username, password_hash, full_name, cpf, branch, email, phone, role, is_active, failed_login_attempts, mfa_enabled, mfa_secret, locked_until, last_login_at, updated_at, created_at) FROM '/tmp/hondago_csv/users.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
\copy role_permissions (role_id, permission_key) FROM '/tmp/hondago_csv/role_permissions.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');

-- 2) Identidade/API
\copy api_accounts (id, user_id, cpf, company_code, account_user, dealer_code, account_password, token, b3_token, last_request_at, cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score, created_at, updated_at) FROM '/tmp/hondago_csv/api_accounts.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
\copy vendor_identity_map (id, user_id, cpf, seller_name, branch, source, is_active, created_at, updated_at) FROM '/tmp/hondago_csv/vendor_identity_map.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');

-- 3) Operação
\copy requests (id, requester_user_id, vendor_identity_id, api_account_id, requested_at, branch, seller_name, cpf, model_name, licensed, installments, bid_percent, with_restriction, group_code, notes, requested_quota_id, served_group, quota_rd, served_at, status, contemplation_bid, requested_date, requested_time, served_date, served_time) FROM '/tmp/hondago_csv/requests.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
CREATE TEMP TABLE tmp_reservations (
  id BIGINT,
  request_id BIGINT,
  reserved_by VARCHAR(255),
  person_document VARCHAR(255),
  group_code VARCHAR(255),
  quota_rd VARCHAR(255),
  model_name VARCHAR(255),
  replacement_quota_id VARCHAR(255),
  created_at TIMESTAMP
);
\copy tmp_reservations (id, request_id, reserved_by, person_document, group_code, quota_rd, model_name, replacement_quota_id, created_at) FROM '/tmp/hondago_csv/reservations.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
INSERT INTO reservations (id, request_id, reserved_by, person_document, group_code, quota_rd, model_name, replacement_quota_id, created_at)
SELECT
  t.id,
  CASE
    WHEN t.request_id IS NULL OR t.request_id <= 0 THEN NULL
    WHEN EXISTS (SELECT 1 FROM requests r WHERE r.id = t.request_id) THEN t.request_id
    ELSE NULL
  END AS request_id,
  t.reserved_by,
  t.person_document,
  t.group_code,
  t.quota_rd,
  t.model_name,
  t.replacement_quota_id,
  t.created_at
FROM tmp_reservations t;
DROP TABLE tmp_reservations;
CREATE TEMP TABLE tmp_available_group_ids (
  id BIGINT,
  group_api_id INTEGER,
  products VARCHAR(255),
  due_day INTEGER,
  term_months INTEGER,
  group_kind VARCHAR(255),
  group_code INTEGER,
  quota INTEGER,
  r INTEGER,
  d INTEGER,
  booked INTEGER,
  created_at TIMESTAMP,
  participants INTEGER,
  failed INTEGER
);
\copy tmp_available_group_ids (id, group_api_id, products, due_day, term_months, group_kind, group_code, quota, r, d, booked, created_at, participants, failed) FROM '/tmp/hondago_csv/available_group_ids.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
INSERT INTO available_group_ids (id, group_api_id, products, due_day, term_months, group_kind, group_code, quota, r, d, booked, created_at, participants, failed)
SELECT
  id,
  COALESCE(group_api_id, 0) AS group_api_id,
  COALESCE(NULLIF(products, ''), '') AS products,
  COALESCE(due_day, 0) AS due_day,
  COALESCE(term_months, 0) AS term_months,
  COALESCE(NULLIF(group_kind, ''), '') AS group_kind,
  COALESCE(group_code, 0) AS group_code,
  COALESCE(quota, 0) AS quota,
  COALESCE(r, 0) AS r,
  COALESCE(d, 0) AS d,
  COALESCE(booked, 0) AS booked,
  created_at,
  COALESCE(participants, 0) AS participants,
  COALESCE(failed, 0) AS failed
FROM tmp_available_group_ids;
DROP TABLE tmp_available_group_ids;
\copy models (id, model_api_id, model_name, status) FROM '/tmp/hondago_csv/models.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
\copy products (id, product_api_id, products, status) FROM '/tmp/hondago_csv/products.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
CREATE TEMP TABLE tmp_active_groups (
  id BIGINT,
  group_code INTEGER,
  due_day INTEGER,
  participants_count INTEGER,
  first_assembly_date DATE,
  plan VARCHAR(64),
  term_months INTEGER,
  group_type VARCHAR(16),
  models TEXT,
  status VARCHAR(32),
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
\copy tmp_active_groups (id, group_code, due_day, participants_count, first_assembly_date, plan, term_months, group_type, models, status, created_at, updated_at) FROM '/tmp/hondago_csv/active_groups.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
INSERT INTO active_groups (id, group_code, due_day, participants_count, first_assembly_date, plan, term_months, group_type, models, status, created_at, updated_at)
SELECT
  id,
  COALESCE(group_code, 0) AS group_code,
  COALESCE(due_day, 0) AS due_day,
  participants_count,
  first_assembly_date,
  plan,
  term_months,
  group_type,
  models,
  COALESCE(NULLIF(status, ''), 'is_active') AS status,
  created_at,
  updated_at
FROM tmp_active_groups;
DROP TABLE tmp_active_groups;
\copy holidays (id, holiday_date, description, holiday_type, is_active, created_at, updated_at) FROM '/tmp/hondago_csv/holidays.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
\copy assemblies (id, quota_rd, contemplation_date, contemplation_type, disqualification_date, client_name, bid_percent, seller_name, group_code, federal_lottery) FROM '/tmp/hondago_csv/assemblies.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
\copy manual_notifications (id, request_id, cpf, seller_name, branch, channel, message, status, copied_at, sent_at, action_user, created_at, updated_at) FROM '/tmp/hondago_csv/manual_notifications.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
\copy audit_log (id, username, action, entity, entity_id, before_state, after_state, created_at) FROM '/tmp/hondago_csv/audit_log.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');
\copy app_sessions (token, user_id, username, full_name, cpf, branch, role, permissions, authenticated_at) FROM '/tmp/hondago_csv/app_sessions.csv' WITH (FORMAT csv, HEADER true, ENCODING 'UTF8');

-- Ajuste de sequências para tabelas com ID identity
SELECT setval(pg_get_serial_sequence('roles', 'id'), COALESCE((SELECT MAX(id) FROM roles), 1), true);
SELECT setval(pg_get_serial_sequence('users', 'id'), COALESCE((SELECT MAX(id) FROM users), 1), true);
SELECT setval(pg_get_serial_sequence('api_accounts', 'id'), COALESCE((SELECT MAX(id) FROM api_accounts), 1), true);
SELECT setval(pg_get_serial_sequence('vendor_identity_map', 'id'), COALESCE((SELECT MAX(id) FROM vendor_identity_map), 1), true);
SELECT setval(pg_get_serial_sequence('requests', 'id'), COALESCE((SELECT MAX(id) FROM requests), 1), true);
SELECT setval(pg_get_serial_sequence('reservations', 'id'), COALESCE((SELECT MAX(id) FROM reservations), 1), true);
SELECT setval(pg_get_serial_sequence('available_group_ids', 'id'), COALESCE((SELECT MAX(id) FROM available_group_ids), 1), true);
SELECT setval(pg_get_serial_sequence('models', 'id'), COALESCE((SELECT MAX(id) FROM models), 1), true);
SELECT setval(pg_get_serial_sequence('products', 'id'), COALESCE((SELECT MAX(id) FROM products), 1), true);
SELECT setval(pg_get_serial_sequence('active_groups', 'id'), COALESCE((SELECT MAX(id) FROM active_groups), 1), true);
SELECT setval(pg_get_serial_sequence('holidays', 'id'), COALESCE((SELECT MAX(id) FROM holidays), 1), true);
SELECT setval(pg_get_serial_sequence('assemblies', 'id'), COALESCE((SELECT MAX(id) FROM assemblies), 1), true);
SELECT setval(pg_get_serial_sequence('manual_notifications', 'id'), COALESCE((SELECT MAX(id) FROM manual_notifications), 1), true);
SELECT setval(pg_get_serial_sequence('audit_log', 'id'), COALESCE((SELECT MAX(id) FROM audit_log), 1), true);

COMMIT;

-- Validação rápida (opcional)
-- SELECT 'users' AS tabela, COUNT(*) FROM users
-- UNION ALL SELECT 'api_accounts', COUNT(*) FROM api_accounts
-- UNION ALL SELECT 'requests', COUNT(*) FROM requests
-- UNION ALL SELECT 'reservations', COUNT(*) FROM reservations;
