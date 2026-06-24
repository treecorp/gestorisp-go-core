package worker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gestor/internal/dominio"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/cripto"
	"gestor/internal/infra/logger"
)

const ws2BaseURL = "https://ws2.gestorisp.com.br"

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func HandlerCheckPopStatus(instancia dominio.Instancia) error {
	tag := "check_pop_status"
	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	pops, err := banco.BuscarPopsOperacionais(db)
	if err != nil {
		return fmt.Errorf("falha ao buscar POPs da instancia %d: %w", instancia.ID, err)
	}

	if len(pops) == 0 {
		logger.Aviso(tag, "Instancia %d: nenhum POP operacional encontrado", instancia.ID)
		return nil
	}

	logger.Info(tag, "Instancia %d: %d POPs operacionais", instancia.ID, len(pops))

	chaveMestra := []byte("sjlkjl32oiPOIjkl2")

	for _, pop := range pops {
		if err := processarPop(tag, db, pop, chaveMestra); err != nil {
			logger.Erro(tag, "POP %d: %v", pop.ID, err)
		}
	}

	return nil
}

func processarPop(tag string, db *sql.DB, pop dominio.Pop, chaveMestra []byte) error {
	payload := map[string]interface{}{
		"host": pop.IPv4,
		"port": pop.APIPort,
		"user": pop.User,
		"pass": pop.Pass,
		"pop":  pop,
	}

	encrypted, err := cripto.CI3Encrypt(payload, chaveMestra)
	if err != nil {
		return fmt.Errorf("erro ao criptografar dados do POP: %w", err)
	}

	url := fmt.Sprintf("%s/WS/routeros_system_health/%s", ws2BaseURL, encrypted)

	resp, err := httpClient.Get(url)
	if err != nil {
		return atualizarOffline(db, pop)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return atualizarOffline(db, pop)
	}

	var resultado []interface{}
	if err := json.Unmarshal(body, &resultado); err != nil {
		return atualizarOffline(db, pop)
	}

	if len(resultado) > 0 {
		return atualizarOnline(db, pop)
	}

	return atualizarOffline(db, pop)
}

func atualizarOnline(db *sql.DB, pop dominio.Pop) error {
	if pop.StatusTimeout > 0 {
		if err := banco.AtualizarStatusTimeout(db, pop.ID, 0, nil); err != nil {
			return err
		}
	}
	return nil
}

func atualizarOffline(db *sql.DB, pop dominio.Pop) error {
	switch pop.StatusTimeout {
	case 3:
		dados := 4
		return banco.AtualizarStatusTimeout(db, pop.ID, dados, nil)
	case 2:
		dados := 3
		agora := time.Now().Format("2006-01-02 15:04:05")
		return banco.AtualizarStatusTimeout(db, pop.ID, dados, &agora)
	default:
		dados := pop.StatusTimeout + 1
		return banco.AtualizarStatusTimeout(db, pop.ID, dados, nil)
	}
}
