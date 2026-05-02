PRAGMA foreign_keys = ON;

BEGIN;

CREATE TABLE IF NOT EXISTS users_v2 (
  id INTEGER PRIMARY KEY,
  username VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  display_name VARCHAR(255) NOT NULL DEFAULT '',
  cpf VARCHAR(32),
  filial VARCHAR(64),
  email VARCHAR(255),
  role VARCHAR(64) NOT NULL DEFAULT 'operador',
  is_active INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0,1)),
  failed_login_attempts INTEGER NOT NULL DEFAULT 0,
  locked_until DATETIME,
  last_login_at DATETIME,
  updated_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS api_accounts_v2 (
  id INTEGER PRIMARY KEY,
  user_id INTEGER,
  cpf VARCHAR(32) NOT NULL,
  cod_empresa VARCHAR(255) NOT NULL,
  cod_usuario VARCHAR(255) NOT NULL,
  cod_concessionaria VARCHAR(255),
  senha VARCHAR(255) NOT NULL,
  token VARCHAR(255),
  token_b3 VARCHAR(255),
  last_request DATETIME,
  cooldown_until DATETIME,
  in_flight INTEGER DEFAULT 0 CHECK (in_flight IN (0,1)),
  error_401_count INTEGER DEFAULT 0,
  error_429_count INTEGER DEFAULT 0,
  blocked_until DATETIME,
  priority_score REAL DEFAULT 0,
  created_at DATETIME,
  updated_at DATETIME,
  FOREIGN KEY (user_id) REFERENCES users_v2(id)
);

CREATE TABLE IF NOT EXISTS vendor_identity_map_v2 (
  id INTEGER PRIMARY KEY,
  cpf VARCHAR(32),
  vendedor VARCHAR(255) NOT NULL,
  filial VARCHAR(64),
  user_id INTEGER,
  source VARCHAR(32) NOT NULL DEFAULT 'legacy',
  created_at DATETIME,
  updated_at DATETIME,
  FOREIGN KEY (user_id) REFERENCES users_v2(id)
);

CREATE TABLE IF NOT EXISTS requests_v2 (
  id INTEGER PRIMARY KEY,
  requester_user_id INTEGER,
  vendor_identity_id INTEGER,
  api_account_id INTEGER,
  data_hora_solicitacao DATETIME,
  data_solicitacao DATE,
  hora_solicitacao TIME,
  filial VARCHAR(64),
  vendedor VARCHAR(255),
  cpf VARCHAR(32),
  modelo VARCHAR(255),
  licenciado VARCHAR(16),
  qtde_parcelas INTEGER,
  perc_lance REAL,
  com_restricao VARCHAR(16),
  grupo INTEGER,
  observacao TEXT,
  id_cota INTEGER,
  grupo_atendido INTEGER,
  cota_r_d VARCHAR(32),
  data_hora_atendimento DATETIME,
  data_atendimento DATE,
  hora_atendimento TIME,
  situacao VARCHAR(64),
  lance_contemplacao VARCHAR(64),
  created_at DATETIME,
  updated_at DATETIME,
  FOREIGN KEY (requester_user_id) REFERENCES users_v2(id),
  FOREIGN KEY (vendor_identity_id) REFERENCES vendor_identity_map_v2(id),
  FOREIGN KEY (api_account_id) REFERENCES api_accounts_v2(id)
);

CREATE TABLE IF NOT EXISTS reservations_v2 (
  id INTEGER PRIMARY KEY,
  request_id INTEGER,
  usuario_reserva VARCHAR(255) NOT NULL,
  cpf VARCHAR(32) NOT NULL,
  cod_grupo VARCHAR(255),
  cota_rd VARCHAR(255),
  cod_modelo VARCHAR(255),
  id_cota_reposicao VARCHAR(255),
  created_at DATETIME,
  FOREIGN KEY (request_id) REFERENCES requests_v2(id)
);

CREATE TABLE IF NOT EXISTS manual_notifications_v2 (
  id INTEGER PRIMARY KEY,
  request_id INTEGER,
  cpf VARCHAR(32),
  vendedor VARCHAR(255),
  filial VARCHAR(64),
  canal VARCHAR(32) NOT NULL DEFAULT 'whatsapp',
  mensagem TEXT NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pendente',
  copiada_em DATETIME,
  enviada_em DATETIME,
  usuario_acao VARCHAR(255),
  created_at DATETIME,
  updated_at DATETIME,
  FOREIGN KEY (request_id) REFERENCES requests_v2(id)
);

CREATE INDEX IF NOT EXISTS idx_users_v2_cpf ON users_v2(cpf);
CREATE INDEX IF NOT EXISTS idx_users_v2_role ON users_v2(role);
CREATE INDEX IF NOT EXISTS idx_api_accounts_v2_cpf ON api_accounts_v2(cpf);
CREATE INDEX IF NOT EXISTS idx_api_accounts_v2_emp_user ON api_accounts_v2(cod_empresa, cod_usuario);
CREATE INDEX IF NOT EXISTS idx_requests_v2_cpf ON requests_v2(cpf);
CREATE INDEX IF NOT EXISTS idx_requests_v2_situacao ON requests_v2(situacao);
CREATE INDEX IF NOT EXISTS idx_requests_v2_solicitacao ON requests_v2(data_hora_solicitacao);
CREATE INDEX IF NOT EXISTS idx_requests_v2_vendor_identity ON requests_v2(vendor_identity_id);
CREATE INDEX IF NOT EXISTS idx_reservations_v2_cpf ON reservations_v2(cpf);
CREATE INDEX IF NOT EXISTS idx_reservations_v2_idcota ON reservations_v2(id_cota_reposicao);
CREATE INDEX IF NOT EXISTS idx_manual_notifications_v2_status_created ON manual_notifications_v2(status, created_at);
CREATE INDEX IF NOT EXISTS idx_vendor_identity_v2_cpf ON vendor_identity_map_v2(cpf);
CREATE INDEX IF NOT EXISTS idx_vendor_identity_v2_vendor_filial ON vendor_identity_map_v2(vendedor, filial);

COMMIT;
