# GO/NO-GO Lite (Subida Imediata)

Data: 02/05/2026  
Objetivo: liberar a versão atual com risco controlado, sem travar a operação.

## 1) Pré-condições obrigatórias (15 min)
- [ ] Build final definida e executável correto separado (`honda-go-gui.exe`)
- [ ] `.env` de produção validado (`DATABASE_URL`, `HONDAGO_SECRETS_KEY`, token/API URL)
- [ ] Banco Postgres acessível no ambiente alvo

Responsável: `__________________`  
Evidência: `__________________`

## 2) Backup e rollback (15 min)
- [ ] Backup do banco feito antes do deploy
- [ ] Procedimento de rollback documentado (voltar binário + restaurar backup)
- [ ] Responsável de rollback definido

Responsável: `__________________`  
Evidência: `__________________`

## 3) Smoke crítico (20-30 min)
- [ ] Login/logout
- [ ] Dashboard carregando sem erro SQL
- [ ] Solicitações: listar + incluir + reservar
- [ ] Configuração de Usuário: autenticar + não autenticados
- [ ] Auditoria gravando alterações

Referência: `docs/SMOKE_TEST_POS_MIGRACAO_POSTGRES.md`

Responsável: `__________________`  
Evidência: `__________________`

## 4) Go-live (janela curta)
- [ ] Deploy executado
- [ ] Healthcheck OK
- [ ] Monitoramento de logs em tempo real por 30-60 min
- [ ] Sem erro crítico bloqueante

Responsável: `__________________`  
Evidência: `__________________`

## 5) Plano paralelo (pós-subida)
- [ ] Manter melhorias em branch/ambiente local
- [ ] Publicar correções em lotes pequenos
- [ ] Repetir smoke crítico a cada atualização

Responsável: `__________________`  
Evidência: `__________________`

---

## Decisão
- [ ] `GO`
- [ ] `NO-GO`

Motivo/observações:  
`__________________________________________________________________________`

Aprovador técnico: `__________________`  
Aprovador operação: `__________________`  
Data/Hora: `__________________`
