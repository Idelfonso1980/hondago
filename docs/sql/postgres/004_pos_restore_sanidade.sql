-- 004_pos_restore_sanidade.sql
-- Objetivo: sanidade pós-restore em Postgres (integridade e preparo operacional).
-- Uso: execute após restaurar um backup SQL.

BEGIN;

-- 1) Alinhar sequences com MAX(id)
SELECT setval(pg_get_serial_sequence('public.users','id'), COALESCE((SELECT MAX(id) FROM public.users),1), true);
SELECT setval(pg_get_serial_sequence('public.api_accounts','id'), COALESCE((SELECT MAX(id) FROM public.api_accounts),1), true);
SELECT setval(pg_get_serial_sequence('public.vendor_identity_map','id'), COALESCE((SELECT MAX(id) FROM public.vendor_identity_map),1), true);
SELECT setval(pg_get_serial_sequence('public.requests','id'), COALESCE((SELECT MAX(id) FROM public.requests),1), true);
SELECT setval(pg_get_serial_sequence('public.reservations','id'), COALESCE((SELECT MAX(id) FROM public.reservations),1), true);
SELECT setval(pg_get_serial_sequence('public.available_group_ids','id'), COALESCE((SELECT MAX(id) FROM public.available_group_ids),1), true);
SELECT setval(pg_get_serial_sequence('public.models','id'), COALESCE((SELECT MAX(id) FROM public.models),1), true);
SELECT setval(pg_get_serial_sequence('public.products','id'), COALESCE((SELECT MAX(id) FROM public.products),1), true);
SELECT setval(pg_get_serial_sequence('public.active_groups','id'), COALESCE((SELECT MAX(id) FROM public.active_groups),1), true);
SELECT setval(pg_get_serial_sequence('public.holidays','id'), COALESCE((SELECT MAX(id) FROM public.holidays),1), true);
SELECT setval(pg_get_serial_sequence('public.assemblies','id'), COALESCE((SELECT MAX(id) FROM public.assemblies),1), true);
SELECT setval(pg_get_serial_sequence('public.manual_notifications','id'), COALESCE((SELECT MAX(id) FROM public.manual_notifications),1), true);
SELECT setval(pg_get_serial_sequence('public.audit_log','id'), COALESCE((SELECT MAX(id) FROM public.audit_log),1), true);

-- 2) Constraints críticas
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.users'::regclass AND contype='p') THEN
    ALTER TABLE public.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.api_accounts'::regclass AND contype='p') THEN
    ALTER TABLE public.api_accounts ADD CONSTRAINT api_accounts_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.vendor_identity_map'::regclass AND contype='p') THEN
    ALTER TABLE public.vendor_identity_map ADD CONSTRAINT vendor_identity_map_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.requests'::regclass AND contype='p') THEN
    ALTER TABLE public.requests ADD CONSTRAINT requests_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.reservations'::regclass AND contype='p') THEN
    ALTER TABLE public.reservations ADD CONSTRAINT reservations_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.available_group_ids'::regclass AND contype='p') THEN
    ALTER TABLE public.available_group_ids ADD CONSTRAINT available_group_ids_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.models'::regclass AND contype='p') THEN
    ALTER TABLE public.models ADD CONSTRAINT models_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.products'::regclass AND contype='p') THEN
    ALTER TABLE public.products ADD CONSTRAINT products_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.active_groups'::regclass AND contype='p') THEN
    ALTER TABLE public.active_groups ADD CONSTRAINT active_groups_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.holidays'::regclass AND contype='p') THEN
    ALTER TABLE public.holidays ADD CONSTRAINT holidays_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.assemblies'::regclass AND contype='p') THEN
    ALTER TABLE public.assemblies ADD CONSTRAINT assemblies_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.manual_notifications'::regclass AND contype='p') THEN
    ALTER TABLE public.manual_notifications ADD CONSTRAINT manual_notifications_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.audit_log'::regclass AND contype='p') THEN
    ALTER TABLE public.audit_log ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);
  END IF;
END$$;

ALTER TABLE public.role_permissions
  DROP CONSTRAINT IF EXISTS fk_role_permissions_role,
  ADD CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;

ALTER TABLE public.api_accounts
  DROP CONSTRAINT IF EXISTS fk_api_accounts_user,
  ADD CONSTRAINT fk_api_accounts_user FOREIGN KEY (user_id) REFERENCES public.users(id);

ALTER TABLE public.vendor_identity_map
  DROP CONSTRAINT IF EXISTS fk_vendor_identity_user,
  ADD CONSTRAINT fk_vendor_identity_user FOREIGN KEY (user_id) REFERENCES public.users(id);

ALTER TABLE public.requests
  DROP CONSTRAINT IF EXISTS fk_requests_requester_user,
  ADD CONSTRAINT fk_requests_requester_user FOREIGN KEY (requester_user_id) REFERENCES public.users(id);

ALTER TABLE public.requests
  DROP CONSTRAINT IF EXISTS fk_requests_vendor_identity,
  ADD CONSTRAINT fk_requests_vendor_identity FOREIGN KEY (vendor_identity_id) REFERENCES public.vendor_identity_map(id);

ALTER TABLE public.requests
  DROP CONSTRAINT IF EXISTS fk_requests_api_account,
  ADD CONSTRAINT fk_requests_api_account FOREIGN KEY (api_account_id) REFERENCES public.api_accounts(id);

ALTER TABLE public.reservations
  DROP CONSTRAINT IF EXISTS fk_reservations_request,
  ADD CONSTRAINT fk_reservations_request FOREIGN KEY (request_id) REFERENCES public.requests(id);

ALTER TABLE public.manual_notifications
  DROP CONSTRAINT IF EXISTS fk_manual_notifications_request,
  ADD CONSTRAINT fk_manual_notifications_request FOREIGN KEY (request_id) REFERENCES public.requests(id);

ALTER TABLE public.app_sessions
  DROP CONSTRAINT IF EXISTS fk_app_sessions_user,
  ADD CONSTRAINT fk_app_sessions_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

-- 3) Uniques operacionais importantes
CREATE UNIQUE INDEX IF NOT EXISTS ux_users_username ON public.users(username);
CREATE UNIQUE INDEX IF NOT EXISTS ux_roles_name ON public.roles(name);
CREATE UNIQUE INDEX IF NOT EXISTS ux_app_sessions_token ON public.app_sessions(token);

COMMIT;

-- 4) Consultas de validação (esperado: zero duplicidades)
-- SELECT id, COUNT(*) FROM public.users GROUP BY id HAVING COUNT(*) > 1;
-- SELECT id, COUNT(*) FROM public.requests GROUP BY id HAVING COUNT(*) > 1;
-- SELECT id, COUNT(*) FROM public.reservations GROUP BY id HAVING COUNT(*) > 1;
-- SELECT id, COUNT(*) FROM public.vendor_identity_map GROUP BY id HAVING COUNT(*) > 1;
-- SELECT id, COUNT(*) FROM public.audit_log GROUP BY id HAVING COUNT(*) > 1;
-- SELECT username, COUNT(*) FROM public.users GROUP BY username HAVING COUNT(*) > 1;
