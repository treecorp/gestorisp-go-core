package main

import (
	"flag"
	"os"

	"gestor/internal/infra/logger"
	"gestor/internal/infra/routeros"
)

func main() {
	logger.Destaque("testedesconexao", "Teste de Desconexao RouterOS")

	executar := flag.Bool("executar", false, "Executa a desconexao (sem a flag, apenas consulta)")
	flag.Parse()

	pppoeUser := "04720186475"
	routerIP := "10.20.1.2"
	routerPort := "8728"
	routerUser := "api"
	routerPass := "33223200#*"

	logger.Info("testedesconexao", "Conectando a %s:%s...", routerIP, routerPort)

	conn, err := routeros.Conectar(routeros.DadosConexao{
		IPv4: routerIP,
		Port: routerPort,
		User: routerUser,
		Pass: routerPass,
	})
	if err != nil {
		logger.Erro("testedesconexao", "Falha ao conectar: %v", err)
		os.Exit(1)
	}
	defer conn.Close()
	logger.Sucesso("testedesconexao", "Conectado ao RouterOS")

	ativo, sessionID, err := routeros.VerificarUsuarioAtivo(conn, pppoeUser)
	if err != nil {
		logger.Erro("testedesconexao", "Erro ao consultar usuario %s: %v", pppoeUser, err)
		os.Exit(1)
	}

	if !ativo {
		logger.Info("testedesconexao", "Usuario %s NAO esta ativo no concentrador", pppoeUser)
		logger.Info("testedesconexao", "Nao ha o que desconectar. Teste OK.")
		return
	}

	logger.Info("testedesconexao", "Usuario %s esta ATIVO", pppoeUser)
	logger.Info("testedesconexao", "  Session ID: %s", sessionID)

	reply, err := conn.Run("/ppp/active/print", "?.id="+sessionID)
	if err == nil && len(reply.Re) > 0 {
		m := reply.Re[0].Map
		if v, ok := m["name"]; ok {
			logger.Info("testedesconexao", "  Nome: %s", v)
		}
		if v, ok := m["address"]; ok {
			logger.Info("testedesconexao", "  Endereco: %s", v)
		}
		if v, ok := m["uptime"]; ok {
			logger.Info("testedesconexao", "  Uptime: %s", v)
		}
		if v, ok := m["service"]; ok {
			logger.Info("testedesconexao", "  Servico: %s", v)
		}
	}

	if !*executar {
		logger.Info("testedesconexao", "Modo consulta apenas. Use --executar para desconectar.")
		return
	}

	logger.Info("testedesconexao", "Desconectando usuario %s (session %s)...", pppoeUser, sessionID)

	if err := routeros.DesconectarUsuario(conn, sessionID); err != nil {
		logger.Erro("testedesconexao", "Falha ao desconectar: %v", err)
		os.Exit(1)
	}

	logger.Sucesso("testedesconexao", "Usuario %s desconectado com sucesso", pppoeUser)
}
