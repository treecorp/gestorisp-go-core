package routeros

import (
	"fmt"
	"time"

	"github.com/go-routeros/routeros/v3"
)

type DadosConexao struct {
	IPv4 string
	Port string
	User string
	Pass string
}

func Conectar(d DadosConexao) (*routeros.Client, error) {
	port := d.Port
	if port == "" {
		port = "8728"
	}
	addr := fmt.Sprintf("%s:%s", d.IPv4, port)
	return routeros.DialTimeout(addr, d.User, d.Pass, 5*time.Second)
}

func VerificarUsuarioAtivo(c *routeros.Client, username string) (ativo bool, sessionID string, err error) {
	reply, err := c.Run("/ppp/active/print", "?name="+username)
	if err != nil {
		return false, "", fmt.Errorf("routeros query: %w", err)
	}
	if len(reply.Re) > 0 {
		return true, reply.Re[0].Map[".id"], nil
	}
	return false, "", nil
}

func DesconectarUsuario(c *routeros.Client, id string) error {
	_, err := c.Run("/ppp/active/remove", "=.id="+id)
	return err
}
