# Decisao 013 — Refatoracao da Estrutura do Projeto (v2)

**Status:** Implementado
**Data:** 26/06/2026
**Contexto:** Os arquivos estavam grandes demais (524 linhas em processar.go, 574 em run_cluster.go) e misturavam queries SQL com regras de negocio. Nao havia separacao clara entre camadas.

**Decisao:** Adotar arquitetura em camadas inspirada no WMVC do CodeIgniter 3:

| Camada | Analogia WMVC | Responsabilidade |
|--------|--------------|-----------------|
| entity/ | Model (dados) | Structs com metodos de comportamento |
| helpers/ | Helper | Funcoes puras (sem estado, sem DB) |
| lib/ | Library | Servicos com dependencia externa (Iugu API) |
| repositorio/ | Model (queries) | SQL isolado, uma entidade por arquivo |
| service/ | Regra de negocio | Logica pura, usa repositorio por interface |
| handler/ | Controller | Orquestracao (HTTP, cron, worker) |
| infra/ | — | Infraestrutura base (banco, mensageria, etc) |

**Regras:**
- NUNCA misturar SQL com regra de negocio
- handler → service → repositorio (via interfaces)
- NUNCA repositorio → service
- Arquivos com mais de 200 linhas devem ser quebrados
- Doc comments em portugues em TODAS as funcoes
- Cada package tem doc.go

**Arquivos afetados:** Todos os packages foram refatorados em branch v2.
