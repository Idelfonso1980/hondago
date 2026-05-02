-- Contagens simples origem x destino
SELECT 'appuser' AS tabela, COUNT(*) AS origem FROM appuser
UNION ALL
SELECT 'users_v2', COUNT(*) FROM users_v2
UNION ALL
SELECT 'auth', COUNT(*) FROM auth
UNION ALL
SELECT 'api_accounts_v2', COUNT(*) FROM api_accounts_v2
UNION ALL
SELECT 'solicitacoes', COUNT(*) FROM solicitacoes
UNION ALL
SELECT 'requests_v2', COUNT(*) FROM requests_v2
UNION ALL
SELECT 'cotasreservadas', COUNT(*) FROM cotasreservadas
UNION ALL
SELECT 'reservations_v2', COUNT(*) FROM reservations_v2
UNION ALL
SELECT 'mensagens_notificacao', COUNT(*) FROM mensagens_notificacao
UNION ALL
SELECT 'manual_notifications_v2', COUNT(*) FROM manual_notifications_v2
UNION ALL
SELECT 'vendor_identity_map_v2', COUNT(*) FROM vendor_identity_map_v2;

-- Chaves não vinculadas (diagnóstico)
SELECT 'requests_sem_user' AS check_name, COUNT(*) AS total
FROM requests_v2
WHERE requester_user_id IS NULL;

SELECT 'requests_sem_vendor_identity' AS check_name, COUNT(*) AS total
FROM requests_v2
WHERE vendor_identity_id IS NULL;

SELECT 'requests_sem_user_e_sem_vendor_identity' AS check_name, COUNT(*) AS total
FROM requests_v2
WHERE requester_user_id IS NULL
  AND vendor_identity_id IS NULL;

SELECT 'requests_sem_api_account' AS check_name, COUNT(*) AS total
FROM requests_v2
WHERE api_account_id IS NULL;

SELECT 'reservations_sem_request' AS check_name, COUNT(*) AS total
FROM reservations_v2
WHERE request_id IS NULL;

SELECT 'notifications_sem_request' AS check_name, COUNT(*) AS total
FROM manual_notifications_v2
WHERE request_id IS NULL;
