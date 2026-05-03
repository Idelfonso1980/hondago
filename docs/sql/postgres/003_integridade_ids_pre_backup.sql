-- 003_integridade_ids_pre_backup.sql
-- Objetivo: garantir integridade de IDs antes de gerar backup.
-- Uso recomendado: executar em ambiente já populado, antes do dump.

BEGIN;

-- Auxiliar: remove duplicidade de IDs em uma tabela mantendo a primeira linha (por ctid).
-- Ajuste manualmente a ordenação se quiser priorizar alguma coluna de data.

-- users
DO $$
BEGIN
  IF to_regclass('public.users_fix_seq') IS NULL THEN
    CREATE SEQUENCE public.users_fix_seq;
  END IF;
END$$;
SELECT setval('public.users_fix_seq', COALESCE((SELECT MAX(id) FROM public.users),1), true);
WITH ranked AS (
  SELECT ctid, id, row_number() OVER (PARTITION BY id ORDER BY ctid) rn
  FROM public.users
), fix AS (
  SELECT ctid, nextval('public.users_fix_seq') AS new_id
  FROM ranked
  WHERE rn > 1
)
UPDATE public.users t
SET id = f.new_id
FROM fix f
WHERE t.ctid = f.ctid;

-- requests
DO $$
BEGIN
  IF to_regclass('public.requests_fix_seq') IS NULL THEN
    CREATE SEQUENCE public.requests_fix_seq;
  END IF;
END$$;
SELECT setval('public.requests_fix_seq', COALESCE((SELECT MAX(id) FROM public.requests),1), true);
WITH ranked AS (
  SELECT ctid, id, row_number() OVER (PARTITION BY id ORDER BY requested_at NULLS LAST, ctid) rn
  FROM public.requests
), fix AS (
  SELECT ctid, nextval('public.requests_fix_seq') AS new_id
  FROM ranked
  WHERE rn > 1
)
UPDATE public.requests t
SET id = f.new_id
FROM fix f
WHERE t.ctid = f.ctid;

-- reservations
DO $$
BEGIN
  IF to_regclass('public.reservations_fix_seq') IS NULL THEN
    CREATE SEQUENCE public.reservations_fix_seq;
  END IF;
END$$;
SELECT setval('public.reservations_fix_seq', COALESCE((SELECT MAX(id) FROM public.reservations),1), true);
WITH ranked AS (
  SELECT ctid, id, row_number() OVER (PARTITION BY id ORDER BY created_at NULLS LAST, ctid) rn
  FROM public.reservations
), fix AS (
  SELECT ctid, nextval('public.reservations_fix_seq') AS new_id
  FROM ranked
  WHERE rn > 1
)
UPDATE public.reservations t
SET id = f.new_id
FROM fix f
WHERE t.ctid = f.ctid;

-- manual_notifications
DO $$
BEGIN
  IF to_regclass('public.manual_notifications_fix_seq') IS NULL THEN
    CREATE SEQUENCE public.manual_notifications_fix_seq;
  END IF;
END$$;
SELECT setval('public.manual_notifications_fix_seq', COALESCE((SELECT MAX(id) FROM public.manual_notifications),1), true);
WITH ranked AS (
  SELECT ctid, id, row_number() OVER (PARTITION BY id ORDER BY created_at NULLS LAST, ctid) rn
  FROM public.manual_notifications
), fix AS (
  SELECT ctid, nextval('public.manual_notifications_fix_seq') AS new_id
  FROM ranked
  WHERE rn > 1
)
UPDATE public.manual_notifications t
SET id = f.new_id
FROM fix f
WHERE t.ctid = f.ctid;

-- Alinha sequences oficiais por tabela quando disponíveis.
SELECT setval(pg_get_serial_sequence('public.users','id'), COALESCE((SELECT MAX(id) FROM public.users),1), true);
SELECT setval(pg_get_serial_sequence('public.requests','id'), COALESCE((SELECT MAX(id) FROM public.requests),1), true);
SELECT setval(pg_get_serial_sequence('public.reservations','id'), COALESCE((SELECT MAX(id) FROM public.reservations),1), true);
SELECT setval(pg_get_serial_sequence('public.manual_notifications','id'), COALESCE((SELECT MAX(id) FROM public.manual_notifications),1), true);

-- Garante PK nas principais tabelas com id.
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.users'::regclass AND contype='p') THEN
    ALTER TABLE public.users ADD CONSTRAINT users_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.requests'::regclass AND contype='p') THEN
    ALTER TABLE public.requests ADD CONSTRAINT requests_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.reservations'::regclass AND contype='p') THEN
    ALTER TABLE public.reservations ADD CONSTRAINT reservations_pkey PRIMARY KEY (id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='public.manual_notifications'::regclass AND contype='p') THEN
    ALTER TABLE public.manual_notifications ADD CONSTRAINT manual_notifications_pkey PRIMARY KEY (id);
  END IF;
END$$;

COMMIT;

-- Validação (deve retornar vazio em todas):
-- SELECT id, COUNT(*) FROM public.users GROUP BY id HAVING COUNT(*) > 1;
-- SELECT id, COUNT(*) FROM public.requests GROUP BY id HAVING COUNT(*) > 1;
-- SELECT id, COUNT(*) FROM public.reservations GROUP BY id HAVING COUNT(*) > 1;
-- SELECT id, COUNT(*) FROM public.manual_notifications GROUP BY id HAVING COUNT(*) > 1;
