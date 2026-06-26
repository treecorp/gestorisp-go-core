# Convencao 005 — Arquitetura em Camadas (entity/repositorio/service/handler)

A partir da branch v2, o projeto segue arquitetura em camadas inspirada no WMVC do CodeIgniter 3:

handler (Controller)
  → service (Regra de Negocio) — recebe repositorio por interface
  → repositorio (Model/Queries) — SQL puro
  → entity (Model/Dados) — structs com comportamento
  → helpers (Helper) — funcoes puras
  → lib (Library) — servicos com dependencia externa

Regras:
- handler NUNCA chama repositorio direto (sempre via service)
- service usa repositorio via interfaces (testavel sem DB real)
- repositorio NUNCA importa service ou handler
- entity e helpers NAO importam nada do projeto
- Arquivos > 200 linhas devem ser quebrados em arquivos menores
- Cada package tem doc.go descrevendo sua responsabilidade
