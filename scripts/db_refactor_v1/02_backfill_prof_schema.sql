PRAGMA foreign_keys = OFF;
BEGIN;

INSERT OR REPLACE INTO users_v2 (
  id, username, password_hash, display_name, cpf, filial, email, role, is_active,
  failed_login_attempts, locked_until, last_login_at, updated_at, created_at
)
SELECT
  id, username, password_hash, display_name, cpf, filial, email, role, is_active,
  failed_login_attempts, locked_until, last_login_at, updated_at, created_at
FROM appuser;

INSERT OR REPLACE INTO api_accounts_v2 (
  id, user_id, cpf, cod_empresa, cod_usuario, cod_concessionaria, senha, token, token_b3, last_request,
  cooldown_until, in_flight, error_401_count, error_429_count, blocked_until, priority_score, created_at, updated_at
)
SELECT
  a.id,
  u.id AS user_id,
  a.cpf, a.cod_empresa, a.cod_usuario, a.cod_concessionaria, a.senha, a.token, a.token_b3, a.last_request,
  a.cooldown_until, a.in_flight, a.error_401_count, a.error_429_count, a.blocked_until, a.priority_score,
  a.last_request AS created_at, a.last_request AS updated_at
FROM auth a
LEFT JOIN users_v2 u ON TRIM(COALESCE(u.cpf, '')) = TRIM(COALESCE(a.cpf, ''));

INSERT INTO vendor_identity_map_v2 (
  cpf, vendedor, filial, user_id, source, created_at, updated_at
)
SELECT DISTINCT
  NULLIF(TRIM(COALESCE(u.cpf, '')), '') AS cpf,
  TRIM(COALESCE(u.display_name, u.username, '')) AS vendedor,
  NULLIF(TRIM(COALESCE(u.filial, '')), '') AS filial,
  u.id AS user_id,
  'appuser' AS source,
  DATETIME('now'),
  DATETIME('now')
FROM users_v2 u
WHERE TRIM(COALESCE(u.display_name, u.username, '')) <> ''
  AND NOT EXISTS (
    SELECT 1
    FROM vendor_identity_map_v2 vm
    WHERE LOWER(TRIM(COALESCE(vm.vendedor, ''))) = LOWER(TRIM(COALESCE(u.display_name, u.username, '')))
      AND LOWER(TRIM(COALESCE(vm.filial, ''))) = LOWER(TRIM(COALESCE(u.filial, '')))
      AND TRIM(COALESCE(vm.cpf, '')) = TRIM(COALESCE(u.cpf, ''))
  );

INSERT INTO vendor_identity_map_v2 (
  cpf, vendedor, filial, user_id, source, created_at, updated_at
)
SELECT DISTINCT
  NULLIF(TRIM(COALESCE(s.cpf, '')), '') AS cpf,
  TRIM(COALESCE(s.vendedor, '')) AS vendedor,
  NULLIF(TRIM(COALESCE(s.filial, '')), '') AS filial,
  NULL AS user_id,
  'solicitacao' AS source,
  DATETIME('now'),
  DATETIME('now')
FROM solicitacoes s
WHERE TRIM(COALESCE(s.vendedor, '')) <> ''
  AND NOT EXISTS (
    SELECT 1
    FROM vendor_identity_map_v2 vm
    WHERE LOWER(TRIM(COALESCE(vm.vendedor, ''))) = LOWER(TRIM(COALESCE(s.vendedor, '')))
      AND LOWER(TRIM(COALESCE(vm.filial, ''))) = LOWER(TRIM(COALESCE(s.filial, '')))
      AND TRIM(COALESCE(vm.cpf, '')) = TRIM(COALESCE(s.cpf, ''))
  );

INSERT INTO users_v2 (
  username, password_hash, display_name, cpf, filial, email, role, is_active,
  failed_login_attempts, locked_until, last_login_at, updated_at, created_at
)
SELECT
  'legacy_vendor_' || printf('%06d', vm.id) AS username,
  '__MIGRATED_NOLOGIN__' AS password_hash,
  vm.vendedor AS display_name,
  vm.cpf AS cpf,
  vm.filial AS filial,
  NULL AS email,
  'vendedor_shadow' AS role,
  0 AS is_active,
  0 AS failed_login_attempts,
  NULL AS locked_until,
  NULL AS last_login_at,
  DATETIME('now') AS updated_at,
  DATETIME('now') AS created_at
FROM vendor_identity_map_v2 vm
WHERE vm.user_id IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM users_v2 u
    WHERE u.username = 'legacy_vendor_' || printf('%06d', vm.id)
  );

UPDATE vendor_identity_map_v2
SET user_id = (
  SELECT u.id
  FROM users_v2 u
  WHERE u.username = 'legacy_vendor_' || printf('%06d', vendor_identity_map_v2.id)
  LIMIT 1
),
updated_at = DATETIME('now')
WHERE user_id IS NULL;

INSERT OR REPLACE INTO requests_v2 (
  id, requester_user_id, vendor_identity_id, api_account_id, data_hora_solicitacao, data_solicitacao, hora_solicitacao, filial, vendedor, cpf, modelo,
  licenciado, qtde_parcelas, perc_lance, com_restricao, grupo, observacao, id_cota, grupo_atendido, cota_r_d,
  data_hora_atendimento, data_atendimento, hora_atendimento, situacao, lance_contemplacao, created_at, updated_at
)
SELECT
  s.id,
  COALESCE(
    (
      SELECT u1.id
      FROM users_v2 u1
      WHERE TRIM(COALESCE(u1.cpf, '')) <> ''
        AND TRIM(COALESCE(u1.cpf, '')) = TRIM(COALESCE(s.cpf, ''))
      ORDER BY u1.id DESC
      LIMIT 1
    ),
    (
      SELECT u2.id
      FROM users_v2 u2
      WHERE LOWER(TRIM(COALESCE(u2.display_name, ''))) = LOWER(TRIM(COALESCE(s.vendedor, '')))
        AND (
          TRIM(COALESCE(s.filial, '')) = ''
          OR LOWER(TRIM(COALESCE(u2.filial, ''))) = LOWER(TRIM(COALESCE(s.filial, '')))
        )
      ORDER BY u2.id DESC
      LIMIT 1
    )
  ) AS requester_user_id,
  (
    SELECT vm.id
    FROM vendor_identity_map_v2 vm
    WHERE LOWER(TRIM(COALESCE(vm.vendedor, ''))) = LOWER(TRIM(COALESCE(s.vendedor, '')))
      AND (
        TRIM(COALESCE(s.filial, '')) = ''
        OR LOWER(TRIM(COALESCE(vm.filial, ''))) = LOWER(TRIM(COALESCE(s.filial, '')))
      )
      AND (
        TRIM(COALESCE(s.cpf, '')) = ''
        OR TRIM(COALESCE(vm.cpf, '')) = TRIM(COALESCE(s.cpf, ''))
      )
    ORDER BY vm.user_id IS NULL, vm.id DESC
    LIMIT 1
  ) AS vendor_identity_id,
  a.id AS api_account_id,
  s.data_hora_solicitacao, s.data_solicitacao, s.hora_solicitacao, s.filial, s.vendedor, s.cpf, s.modelo,
  s.licenciado, s.qtde_parcelas, s.perc_lance, s.com_restricao, s.grupo, s.observacao, s.id_cota, s.grupo_atendido, s.cota_r_d,
  s.data_hora_atendimento, s.data_atendimento, s.hora_atendimento, s.situacao, s.lance_contemplacao,
  COALESCE(s.data_hora_solicitacao, DATETIME('now')) AS created_at,
  DATETIME('now') AS updated_at
FROM solicitacoes s
LEFT JOIN api_accounts_v2 a ON TRIM(COALESCE(a.cpf, '')) = TRIM(COALESCE(s.cpf, ''));

INSERT OR REPLACE INTO reservations_v2 (
  id, request_id, usuario_reserva, cpf, cod_grupo, cota_rd, cod_modelo, id_cota_reposicao, created_at
)
SELECT
  r.id,
  COALESCE(
    (
      SELECT s1.id
      FROM requests_v2 s1
      WHERE TRIM(COALESCE(s1.cpf, '')) = TRIM(COALESCE(r.numDocumentoPessoa, ''))
        AND CAST(COALESCE(s1.id_cota, '') AS TEXT) = CAST(COALESCE(r.idCotaReposicao, '') AS TEXT)
      ORDER BY s1.id DESC
      LIMIT 1
    ),
    (
      SELECT s2.id
      FROM requests_v2 s2
      WHERE TRIM(COALESCE(s2.cpf, '')) = TRIM(COALESCE(r.numDocumentoPessoa, ''))
        AND CAST(COALESCE(s2.grupo_atendido, s2.grupo, '') AS TEXT) = CAST(COALESCE(r.codGrupo, '') AS TEXT)
        AND TRIM(COALESCE(s2.cota_r_d, '')) = TRIM(COALESCE(r.cotaRD, ''))
      ORDER BY s2.id DESC
      LIMIT 1
    )
  ) AS request_id,
  r.usuarioReserva,
  r.numDocumentoPessoa,
  r.codGrupo,
  r.cotaRD,
  r.codModelo,
  r.idCotaReposicao,
  r.created_at
FROM cotasreservadas r;

INSERT OR REPLACE INTO manual_notifications_v2 (
  id, request_id, cpf, vendedor, filial, canal, mensagem, status, copiada_em, enviada_em, usuario_acao, created_at, updated_at
)
SELECT
  m.id,
  m.solicitacao_id,
  m.cpf,
  m.vendedor,
  m.filial,
  m.canal,
  m.mensagem,
  m.status,
  m.copiada_em,
  m.enviada_em,
  m.usuario_acao,
  m.created_at,
  m.updated_at
FROM mensagens_notificacao m;

COMMIT;
PRAGMA foreign_keys = ON;
