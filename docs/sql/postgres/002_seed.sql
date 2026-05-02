BEGIN;

INSERT INTO roles (id, name, description)
VALUES
  (1, 'admin', 'Administrador Global do Sistema'),
  (2, 'operador', 'Operador de Vendas e Reservas')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
  (1, 'dashboard:read'),
  (1, 'solicitacoes:read'),
  (1, 'solicitacoes:create'),
  (1, 'solicitacoes:edit'),
  (1, 'solicitacoes:delete'),
  (1, 'solicitacoes:print'),
  (1, 'cotas:reserve'),
  (1, 'cotas:export'),
  (1, 'users:read'),
  (1, 'users:create'),
  (1, 'users:edit'),
  (1, 'users:delete'),
  (1, 'roles:manage'),
  (1, 'configs:manage'),
  (1, 'logs:read'),
  (1, 'logs:delete'),
  (2, 'dashboard:read'),
  (2, 'solicitacoes:read'),
  (2, 'solicitacoes:create'),
  (2, 'solicitacoes:edit'),
  (2, 'cotas:reserve')
ON CONFLICT (role_id, permission_key) DO NOTHING;

INSERT INTO holidays (holiday_date, description, holiday_type, is_active, created_at, updated_at) VALUES
  ('2025-01-01', 'Confraternizacao Universal', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2025-04-21', 'Tiradentes', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2025-05-01', 'Dia do Trabalhador', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2025-09-07', 'Independencia do Brasil', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2025-10-12', 'Nossa Senhora Aparecida', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2025-11-02', 'Finados', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2025-11-15', 'Proclamacao da Republica', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2025-11-20', 'Dia da Consciencia Negra', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2025-12-25', 'Natal', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

  ('2026-01-01', 'Confraternizacao Universal', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2026-04-21', 'Tiradentes', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2026-05-01', 'Dia do Trabalhador', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2026-09-07', 'Independencia do Brasil', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2026-10-12', 'Nossa Senhora Aparecida', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2026-11-02', 'Finados', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2026-11-15', 'Proclamacao da Republica', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2026-11-20', 'Dia da Consciencia Negra', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2026-12-25', 'Natal', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),

  ('2027-01-01', 'Confraternizacao Universal', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2027-04-21', 'Tiradentes', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2027-05-01', 'Dia do Trabalhador', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2027-09-07', 'Independencia do Brasil', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2027-10-12', 'Nossa Senhora Aparecida', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2027-11-02', 'Finados', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2027-11-15', 'Proclamacao da Republica', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2027-11-20', 'Dia da Consciencia Negra', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('2027-12-25', 'Natal', 'Nacional', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (holiday_date, holiday_type) DO NOTHING;

COMMIT;
