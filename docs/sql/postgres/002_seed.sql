BEGIN;

INSERT INTO roles (id, name, description) VALUES
(1, 'admin', 'Administrador Global do Sistema'),
(2, 'operador', 'Operador de Vendas e Reservas'),
(3, 'vendedor', 'Vendedor'),
(4, 'viewer', 'Somente Leitura'),
(5, 'supervisor', 'Supervisor de Equipe')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(1,'dashboard:read'), (1,'solicitacoes:read'), (1,'solicitacoes:create'), (1,'solicitacoes:edit'), (1,'solicitacoes:delete'), (1,'solicitacoes:print'),
(1,'cotas:reserve'), (1,'cotas:export'), (1,'users:read'), (1,'users:create'), (1,'users:edit'), (1,'users:delete'),
(1,'roles:manage'), (1,'configs:manage'), (1,'logs:read'), (1,'logs:delete'), (1,'audit:view')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(2,'dashboard:read'), (2,'solicitacoes:read'), (2,'solicitacoes:create'), (2,'solicitacoes:edit'), (2,'cotas:reserve')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(3,'dashboard:read'), (3,'solicitacoes:read'), (3,'solicitacoes:create'), (3,'solicitacoes:edit'), (3,'cotas:reserve')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(4,'dashboard:read'), (4,'solicitacoes:read')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(5,'dashboard:read'), (5,'solicitacoes:read')
ON CONFLICT DO NOTHING;

COMMIT;
