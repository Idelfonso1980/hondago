-- NAO EXECUTAR NESTA FASE (placeholder)
-- Este script existe apenas para a fase futura, quando o backend ja estiver adaptado ao schema v2.
--
-- Sugestao para fase futura:
-- 1) Congelar escrita (janela curta).
-- 2) Backup.
-- 3) Renomear tabelas legadas para *_legacy.
-- 4) Renomear *_v2 para nomes oficiais.
-- 5) Recriar indices e validar smoke.
--
-- Exemplo (comentado):
-- ALTER TABLE appuser RENAME TO appuser_legacy;
-- ALTER TABLE users_v2 RENAME TO appuser;

SELECT 'cutover_placeholder' AS info, 'Nao executar ate adaptar backend para schema v2' AS observacao;

