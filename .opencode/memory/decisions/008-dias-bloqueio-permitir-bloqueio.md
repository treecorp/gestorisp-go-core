# Decisao 008 — Precedencia de dias_bloqueio + permitir_bloqueio no bloqueio de inadimplentes

**Data:** 24/06/2026
**Contexto:** Handler `listar_clientes_vencidos` — bloqueio automatico de contratos inadimplentes.

## Problema
No PHP legado, dois campos existiam na tabela `sgp_clientes_contratos` mas nunca eram lidos no bloqueio automatico:
- `permitir_bloqueio int(1) DEFAULT 1` — campo de frontend ignorado pelo codigo
- `dias_bloqueio varchar(3) NULL` — per-contrato, mas o codigo usava apenas o global

Isso impedia que contratos especificos fossem protegidos do bloqueio automatico.

## Decisao
Implementar a logica completa no handler Go:

1. **`permitir_bloqueio`**: Se `= 0`, o contrato e pulado (log + continue). So contrataos com `permitir_bloqueio != 0` sao elegiveis.

2. **`dias_bloqueio`**: Precedencia:
   | Prioridade | Origem | Campo |
   |------------|--------|-------|
   | 1º | Contrato (se NOT NULL) | `sgp_clientes_contratos.dias_bloqueio` |
   | 2º | Parametro global | `sgp_parametros.dias_bloqueio` (default 5) |

## Consequencias
- Contratos com `permitir_bloqueio=0` (ex: cliente especial, em negociacao) nunca serao bloqueados automaticamente
- Contratos podem ter tolerancia personalizada sem afetar os demais
- Compatibilidade com versoes anteriores: contratos com `dias_bloqueio=NULL` e `permitir_bloqueio=1` (default) se comportam exatamente como no PHP legado
