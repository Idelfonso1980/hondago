# Plano de Modo Agressivo Controlado para Captura de Cotas (Versão Enxuta + Híbrida)

Data: 2026-05-18

## Objetivo
Aumentar captura total em janela curta de disputa (ex.: ~7,55s), sem abrir mão das melhores cotas.

## Contexto Confirmado
- Cenário observado: ~750 disponíveis, 41 capturadas.
- Loteria federal define o desfecho final (aleatório/imprevisível).
- Heurísticas como `qtd`, histórico de `% lance` e recência não devem dirigir prioridade.
- Faixa de parcelas relevante: 36 a 74 (74 = mais valiosa).

## Priorização de Fila (Simples e Eficiente)
Ordenação base:
1. `parcelas DESC`
2. desempate técnico: `id_cota ASC`

## Estratégia Híbrida (Top + Volume)

### Fase 1 - Burst Topo (curta)
- Janela inicial: 1 a 2 segundos.
- 100% da capacidade nos maiores níveis de parcelas (74, 73, 72...).
- Objetivo: capturar premium antes de esgotar.

### Fase 2 - Híbrida 80/20
- Após burst inicial:
- 80% da capacidade em faixa intermediária (volume).
- 20% da capacidade mantendo caça ao topo remanescente.
- Objetivo: garantir volume sem abandonar qualidade.

### Fase 3 - Descida contínua
- Se ainda houver janela e estoque: seguir descendo por parcelas (DESC).
- Mantém regra simples e previsível.

## Parâmetros Operacionais
- `Limite = 0` para processar toda fila elegível.
- `Processos Paralelos`: alto conforme capacidade operacional.
- `Timeout`: curto o suficiente para evitar worker preso.
- `Retry`: só para falha transitória, com limite baixo.

## Critérios de Sucesso
1. Mais capturas totais no mesmo intervalo.
2. Manter parte relevante das cotas premium.
3. Melhor taxa de confirmação nos primeiros segundos.

## Implementação Objetiva
1. Implementar burst inicial 100% topo (1-2s).
2. Implementar distribuição dinâmica 80/20 (intermediário/topo).
3. Manter ordenação base por `parcelas DESC`.
4. Medir resultado por execução (premium + volume).

## Resultado Esperado
- Sair do cenário de baixa captura inicial.
- Melhor equilíbrio entre qualidade (topo) e quantidade (volume).
- Fluxo simples, auditável e replicável.
