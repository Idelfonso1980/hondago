BEGIN;

INSERT INTO roles (id, name, description) VALUES
(1, 'admin', 'Administrador do sistema'),
(2, 'operador', 'Operador padrão')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key) VALUES
(1, 'dashboard:view'),
(1, 'requests:view'),
(1, 'requests:edit'),
(1, 'reservations:view'),
(1, 'reservations:edit'),
(1, 'messages:view'),
(1, 'messages:edit'),
(1, 'users:view'),
(1, 'users:edit'),
(1, 'rbac:view'),
(1, 'rbac:edit'),
(1, 'audit:view'),
(1, 'settings:view'),
(1, 'settings:edit'),
(2, 'dashboard:view'),
(2, 'requests:view'),
(2, 'requests:edit'),
(2, 'reservations:view'),
(2, 'messages:view'),
(2, 'messages:edit'),
(2, 'settings:view')
ON CONFLICT DO NOTHING;

COMMIT;
