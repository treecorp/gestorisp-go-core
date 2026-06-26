# SDD-023 — Service de Bloqueio

**Status:** Pendente
**Autor:** Knowledge Engineer
**Prioridade:** Alta
**Dependencias:** SDD-018 (`entity.Fatura`, `entity.Contrato`), SDD-020 (`repositorio/bloqueio_repo.go`)

## 1. Objetivo

Extrair regras de negócio de bloqueio de clientes de `listar_clientes_vencidos.go` para o pacote `internal/service/bloqueio/`, separando a lógica de decisão do acesso a dados e promovendo testabilidade.

## 2. Escopo

### 2.1 internal/service/bloqueio/cliente.go

```go
// ProcessarFatura avalia uma fatura vencida e decide se o cliente deve
// ser bloqueado. Retorna um ClienteBloqueado se o bloqueio for necessário,
// ou nil se o cliente não deve ser bloqueado.
func ProcessarFatura(
    repo *Repositorios,
    tag string,
    db *sql.DB,
    fatura entity.Fatura,
    diasBloqueioGlobal int,
) (*entity.ClienteBloqueado, error)
```

Funções auxiliares:

```go
// DeveBloquear é uma função pura que decide se um cliente deve ser
// bloqueado com base nos dados da fatura, contrato e configuração de
// dias de bloqueio.
func DeveBloquear(fatura entity.Fatura, contrato entity.Contrato, diasBloqueio int) bool

// CalcularDiasAtraso calcula quantos dias se passaram desde a data de
// vencimento da fatura. Se a data for inválida, retorna 0.
func CalcularDiasAtraso(vencimento string) int
```

### 2.2 Structs auxiliares

```go
// Repositorios agrupa as interfaces de repositório para o service de bloqueio.
type Repositorios struct {
    Fatura   FaturaRepo
    Contrato ContratoRepo
    Bloqueio BloqueioRepo
}

type BloqueioRepo interface {
    BuscarFaturasVencidas(db *sql.DB) ([]entity.Fatura, error)
    LerDesbloqueioConfianca(db *sql.DB, contratoID int) (*DesbloqueioConfianca, error)
    LerDiasBloqueio(db *sql.DB) int
}
```

### 2.3 Regras de negócio a extrair

- Decisão de bloquear baseada em: dias de atraso, valor mínimo, status do contrato, configuração de desbloqueio por confiança
- Cálculo de dias em atraso a partir da data de vencimento
- Lógica de exclusão (clientes com desbloqueio por confiança ativo não são bloqueados)

## 3. Dependências

- `database/sql` — apenas interfaces
- `internal/entity` — structs `Fatura`, `Contrato`, `ClienteBloqueado`
- `internal/helpers` — funções auxiliares de data

## 4. Arquivos a criar

| Arquivo | Ação |
|---------|------|
| `internal/service/bloqueio/cliente.go` | Criar — ProcessarFatura, DeveBloquear, CalcularDiasAtraso |
| `internal/service/bloqueio/service.go` | Criar — interfaces + Repositorios struct |
| `internal/service/bloqueio/doc.go` | Criar — package-level doc |

## 5. Documentação

Toda função exportada deve ter **doc comment em português**:

```go
// Package bloqueio implementa as regras de negócio para bloqueio de
// clientes com faturas vencidas, incluindo a lógica de decisão e
// cálculo de dias em atraso.
package bloqueio
```

```go
// DeveBloquear é uma função pura que avalia se o contrato associado à
// fatura deve ser bloqueado. Considera dias de atraso, configuração
// global de bloqueio e regras de exceção (ex: desbloqueio por confiança).
func DeveBloquear(fatura entity.Fatura, contrato entity.Contrato, diasBloqueio int) bool
```

## 6. Testes

Por ser uma função pura, `DeveBloquear` deve ser testada com:

- Fatura vencida há mais dias que o limite → deve bloquear
- Fatura vencida há menos dias que o limite → não deve bloquear
- Fatura paga → não deve bloquear
- Contrato com desbloqueio por confiança ativo → não deve bloquear
- Datas de vencimento inválidas → comportamento definido (não bloquear)

## 7. Critérios de Aceite

- [ ] `internal/service/bloqueio/` compila sem erros
- [ ] `DeveBloquear` é uma função pura (sem side effects, sem DB)
- [ ] `CalcularDiasAtraso` implementada (se não movida para `helpers`)
- [ ] Todas as funções possuem doc comment em português
- [ ] `doc.go` presente no pacote
- [ ] Código original `listar_clientes_vencidos.go` permanece intacto
