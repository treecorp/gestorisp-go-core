# SDD-021 — Repositórios de Infraestrutura

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** SDD-018 (usa `entity.Pop`)

## 1. Objetivo

Extrair queries SQL de `cron_1.go`, `run_cluster.go`, `sync_conexoes_radius_arquivo.go` e `gispinstancia.go` para o pacote `internal/repositorio/`, completando a centralização de acesso a dados do sistema.

## 2. Escopo

### 2.1 internal/repositorio/pop_repo.go

```go
// BuscarPopsOperacionais retorna todos os POPs com status ativo.
// Extraído de carregarPops + gispinstancia.go.
func BuscarPopsOperacionais(db *sql.DB) ([]entity.Pop, error)

// AtualizarStatusTimeout marca um POP como timeout.
// Extraído de gispinstancia.go.
func AtualizarStatusTimeout(db *sql.DB, popID int) error
```

### 2.2 internal/repositorio/radacct_repo.go

```go
// ContarConexoes retorna o total de conexões online e offline na tabela
// radacct. Extraído de cron_1.go.
func ContarConexoes(db *sql.DB) (on, off int, err error)

// BuscarSessoesOrphan retorna sessões órfãs (sem término).
// Extraído de cron_1.go.
func BuscarSessoesOrphan(db *sql.DB) ([]SessaoOrphan, error)

// BuscarSessoesTravadas retorna sessões travadas.
// Extraído de cron_1.go.
func BuscarSessoesTravadas(db *sql.DB) ([]SessaoTravada, error)

// DetectarColunasRadacct faz introspecção do schema da tabela radacct
// para detectar colunas disponíveis.
func DetectarColunasRadacct(db *sql.DB) (map[string]bool, error)

// BuscarRadacctPendenteArquivo retorna registros radacct pendentes de
// sincronização por arquivo.
func BuscarRadacctPendenteArquivo(db *sql.DB, colunas map[string]bool) ([]RadacctRecord, error)

// InserirRadacct insere um novo registro na tabela radacct.
func InserirRadacct(tx *sql.Tx, rec RadacctRecord) error

// AtualizarRadacct atualiza um registro existente na tabela radacct.
func AtualizarRadacct(tx *sql.Tx, rec RadacctRecord, id int) error
```

### 2.3 internal/repositorio/cluster_repo.go

```go
// ContarConexoesCluster retorna total de conexões no cluster.
func ContarConexoesCluster(db *sql.DB) (on, off int, err error)

// ContarBloqueados retorna total de clientes bloqueados.
func ContarBloqueados(db *sql.DB) int

// ContarOSPendentes retorna total de ordens de serviço pendentes.
func ContarOSPendentes(db *sql.DB) int

// BuscarClientesColmeia retorna clientes aptos para colmeia (agrupação).
func BuscarClientesColmeia(db *sql.DB) ([]ClienteColmeia, error)

// BuscarDesconexoes retorna mapa de contrato_id -> status para clientes
// com desconexão pendente.
func BuscarDesconexoes(db *sql.DB) (map[int]int, error)

// AtualizarClusterContratos atualiza dados de cluster para contratos.
func AtualizarClusterContratos(db *sql.DB, clusterJSON string, ...) error

// BuscarClientesCoordenadas retorna clientes com coordenadas geográficas.
func BuscarClientesCoordenadas(db *sql.DB) ([]ClienteCoordenada, error)

// AtualizarCoordenadas atualiza coordenadas de clientes no banco.
func AtualizarCoordenadas(db *sql.DB, coordenadasJSON string) error
```

## 3. Tipos auxiliares

```go
// SessaoOrphan representa uma sessão radacct sem registro de término.
type SessaoOrphan struct { ... }

// SessaoTravada representa uma sessão radacct travada.
type SessaoTravada struct { ... }

// RadacctRecord representa um registro completo da tabela radacct.
type RadacctRecord struct { ... }

// ClienteColmeia representa um cliente apto para agrupação colmeia.
type ClienteColmeia struct { ... }

// ClienteCoordenada representa um cliente com coordenadas geográficas.
type ClienteCoordenada struct { ... }
```

## 4. Dependências

- `database/sql` — interface de banco
- `internal/entity` — structs `Pop`

## 5. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `internal/repositorio/pop_repo.go` | Criar — queries de POP |
| `internal/repositorio/radacct_repo.go` | Criar — queries radacct |
| `internal/repositorio/cluster_repo.go` | Criar — queries de cluster |

## 6. Documentação

Toda função deve ter **doc comment em português**:

```go
// ContarConexoes executa SELECT COUNT(*) nas sessões radacct agrupando
// por status online/offline. Retorna (on, off, err).
func ContarConexoes(db *sql.DB) (on, off int, err error)
```

## 7. Critérios de Aceite

- [ ] `internal/repositorio/pop_repo.go` compila sem erros
- [ ] `internal/repositorio/radacct_repo.go` compila sem erros
- [ ] `internal/repositorio/cluster_repo.go` compila sem erros
- [ ] Todas as funções possuem doc comment em português
- [ ] Código fonte original (`cron_1.go`, `run_cluster.go`, etc.) permanece intacto
