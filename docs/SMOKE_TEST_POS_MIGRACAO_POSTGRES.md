# Smoke Test Pós Migração Postgres

## Pré-condições
- [ ] App online e login admin funcionando
- [ ] Banco PostgreSQL restaurado
- [ ] Variáveis de ambiente configuradas

## Fluxos críticos
- [ ] Login/logout
- [ ] Dashboard carrega sem erro
- [ ] Solicitações: listar/criar/editar/excluir
- [ ] Reserva unitária
- [ ] Reserva em lote
- [ ] Mensagens: listar/copiar/marcar enviada
- [ ] Configuração de usuários: autenticar
- [ ] Auditoria: grava e lista

## Sanidade banco
- [ ] Sem IDs duplicados em users/requests/reservations
- [ ] Sequences alinhadas com MAX(id)
- [ ] PK/FK essenciais presentes
