BEGIN;

INSERT INTO roles (id, name, description) VALUES
(1, 'admin', 'Administrador Global do Sistema'),
(2, 'operador', 'Operador de Vendas e Reservas'),
(3, 'vendedor', 'Vendedor'),
(4, 'viewer', 'Somente Leitura'),
(5, 'supervisor', 'Supervisor de Equipe'),
(6, 'gerente', 'Gerente de Filial')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(1,'dashboard:read'), (1,'solicitacoes:read'), (1,'solicitacoes:create'), (1,'solicitacoes:edit'), (1,'solicitacoes:delete'), (1,'solicitacoes:print'),
(1,'cotas:reserve'), (1,'cotas:export'), (1,'users:read'), (1,'users:create'), (1,'users:edit'), (1,'users:delete'),
(1,'roles:manage'), (1,'configs:manage'), (1,'logs:read'), (1,'logs:delete'), (1,'audit:view'),
(1,'nav:dashboard'), (1,'nav:reservas'), (1,'nav:monitor'), (1,'nav:logs'), (1,'nav:config'),
(1,'monitor:read'),
(1,'reservas:home'), (1,'reservas:solicitacoes'), (1,'reservas:minhas'), (1,'reservas:solicitar'), (1,'reservas:reservadas'), (1,'reservas:mensagens'), (1,'reservas:config'),
(1,'config:users'), (1,'config:appusers'), (1,'config:rbac'), (1,'config:audit'), (1,'config:database'), (1,'config:idsgrupos'), (1,'config:active_groups'), (1,'config:assemblies'), (1,'config:models'), (1,'config:produtos')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(2,'dashboard:read'), (2,'solicitacoes:read'), (2,'solicitacoes:create'), (2,'solicitacoes:edit'), (2,'cotas:reserve'),
(2,'nav:dashboard'), (2,'nav:reservas'),
(2,'reservas:solicitacoes'), (2,'reservas:minhas'), (2,'reservas:solicitar'), (2,'reservas:reservadas')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(3,'dashboard:read'), (3,'solicitacoes:read'), (3,'solicitacoes:create'), (3,'solicitacoes:edit'), (3,'cotas:reserve'),
(3,'nav:reservas'),
(3,'reservas:home'), (3,'reservas:minhas'), (3,'reservas:solicitar')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(4,'dashboard:read'), (4,'solicitacoes:read'),
(4,'nav:dashboard'), (4,'nav:reservas'),
(4,'reservas:solicitacoes'), (4,'reservas:minhas')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(5,'dashboard:read'), (5,'solicitacoes:read'),
(5,'nav:dashboard'), (5,'nav:reservas'),
(5,'reservas:home'), (5,'reservas:solicitacoes'), (5,'reservas:minhas')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(6,'dashboard:read'), (6,'solicitacoes:read'),
(6,'nav:dashboard'), (6,'nav:reservas'),
(6,'reservas:home'), (6,'reservas:solicitacoes'), (6,'reservas:minhas')
ON CONFLICT DO NOTHING;

COMMIT;
