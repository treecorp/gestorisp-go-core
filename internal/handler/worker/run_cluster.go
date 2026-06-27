package worker

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gestor/internal/entity"
	"gestor/internal/infra/banco"
	"gestor/internal/infra/fuso"
	"gestor/internal/infra/logger"
	"gestor/internal/repositorio"
)

const (
	imgOnline = "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAQDAwMDAwQDAwQFBAMEBQcFBAQFBwgGBgcGBggKCAgICAgICggKCgsKCggNDQ4ODQ0SEhISEhQUFBQUFBQUFBT/2wBDAQUFBQgHCA8KCg8SDwwPEhYVFRUVFhYUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAAKAAoDAREAAhEBAxEB/8QAFwAAAwEAAAAAAAAAAAAAAAAAAwUGCP/EACAQAAMAAwACAgMAAAAAAAAAAAECAwQFBgAREkETITH/xAAZAQACAwEAAAAAAAAAAAAAAAACBAEFBgf/xAAfEQACAgICAwEAAAAAAAAAAAABAgAEAxExYSJCUbH/2gAMAwEAAhEDEQA/AGfUdP00Om3Mp7jYySexyUSa5NQFCWb4qAG9AL9eVjMdziN69YFjIBkceZ9j9Pc0DzBtk83psnIysil7a7GpWjULMzvJSSSfZJJ8bXib2jtq+MknZRfyC3HM83TI/PTT6972ZnrRsaRZ3b9lmPx9kk/fkFRBsUq5bZxrs9CUODKUcLGjFFnGcUSc0AVVVVAAAH8A8IR7EAEAHGp//9k="

	imgOffline = "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAQDAwMDAwQDAwQFBAMEBQcFBAQFBwgGBgcGBggKCAgICAgICggKCgsKCggNDQ4ODQ0SEhISEhQUFBQUFBQUFBT/2wBDAQUFBQgHCA8KCg8SDwwPEhYVFRUVFhYUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAAKAAoDAREAAhEBAxEB/8QAFwAAAwEAAAAAAAAAAAAAAAAAAwYHCP/EACEQAAICAgIBBQAAAAAAAAAAAAIDAQQFBgARMRIUISJB/8QAGgEAAQUBAAAAAAAAAAAAAAAABgECAwQFB//EACMRAAEDAwMFAQAAAAAAAAAAAAEAAgMEERIhMVEiMkFxkbH/2gAMAwEAAhEDEQA/AJ1u+7brW3TY66dhzCFJzFxa0hdsCICuwfpGIg+ogfzrxwQmmfmdTvyulU1NEYm9Le0eBwtZaTNi7puu3bd6463YxFJz2m0iM2MrgRERT3MzMz5nm1DqweghapsJXAAWyP6g7DpenOt+6dr2JZZsExthx0kEbGH9iIykOymZnuZniPiZwE6GplAtk76U14tCK2Mp166gTXTXUtSljAgACEQIiMfEREeI5M3ZVXm7iSv/2Q=="

	imgBloqueado = "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAQDAwMDAwQDAwQFBAMEBQcFBAQFBwgGBgcGBggKCAgICAgICggKCgsKCggNDQ4ODQ0SEhISEhQUFBQUFBQUFBT/2wBDAQUFBQgHCA8KCg8SDwwPEhYVFRUVFhYUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAAKAAoDAREAAhEBAxEB/8QAFgABAQEAAAAAAAAAAAAAAAAACAQG/8QAHxAAAgICAwEBAQAAAAAAAAAAAgMBBAUHAAYRQRQx/8QAFAEBAAAAAAAAAAAAAAAAAAAAAP/EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAMAwEAAhEDEQA/AClsjY+xqexe3Va/bM/VSjP5BSq68jaAFAm2yAARFnkQHyI/nAdmuJtZLXnUsjfyV+xet4DHWLL2vI2Ma2qsjMiL2Skin2ZmeBL2zXWvrF79tjqeBbdtm19qweOqk1rWekRsOV+kRFPszP3gbrC1q1PDY+pUStFRFRKkIUMAsFguBEAEfIiBiPIiOB//2Q=="

	imgDesconexao = "data:image/gif;base64,R0lGODlhCgAKANUAAPP5//v12f/z2f/tsf/ts//trf/tqf/rrf/rn//nvf/rof/pqf/pr//prf/rnf/pp//nqf/nof/no/vpk//nnf/nj/nnmf/lm//lkf/jn//li//jl//jmf/jnf/hmf/jj//hn//hdPvfcP+7AgC5Wv4BAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH/C05FVFNDQVBFMi4wAwEAAAAh+QQFZAAlACwAAAAACgAKAAAGL0BAJcKRPECMT2AyajqbFsfzqUFMnYbMtVm4bEeLw7dB+BpCX4ogAVF0PBuMaBAEACH5BAVkACUALAAAAQAJAAgAAAYVwBJpSBwKi8QjkqRENovP5NI4JQUBADs="

	imgPendenteAtivacao = "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAQDAwMDAwQDAwQFBAMEBQcFBAQFBwgGBgcGBggKCAgICAgICggKCgsKCggNDQ4ODQ0SEhISEhQUFBQUFBQUFBT/2wBDAQUFBQgHCA8KCg8SDwwPEhYVFRUVFhYUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBT/wAARCAAKAAoDAREAAhEBAxEB/8QAFwAAAwEAAAAAAAAAAAAAAAAAAQQGCP/EAB4QAAICAgIDAAAAAAAAAAAAAAECAwQAERIhMVFh/8QAGAEAAgMAAAAAAAAAAAAAAAAAAAECAwT/xAAWEQEBAQAAAAAAAAAAAAAAAAAAARH/2gAMAwEAAhEDEQA/ANUXr15btlVszKBM4Ch260x+5pkJX0eT0qzu7l2hQsSeySoyqmFijSZ+RrQlmJLEouyT76w0G4lVY0VQAoUAAeNZEP/Z"
)

// clusterContrato representa um contrato no cluster de visualizacao.
type clusterContrato struct {
	Token         string `json:"token"`
	Img           string `json:"img"`
	Cliente       string `json:"cliente"`
	Logradouro    string `json:"logradouro"`
	OrganizacaoID *int   `json:"organizacao_id,omitempty"`
}

// coordenadaGPS representa um marcador de coordenada geografica.
type coordenadaGPS struct {
	Icon              string `json:"icon"`
	InfoWindowContent string `json:"infowindow_content"`
	Lat               string `json:"lat"`
	Lon               string `json:"lon"`
	Title             string `json:"title"`
	OrganizacaoID     int    `json:"organizacao_id"`
}

// HandlerRunCluster processa o cluster de contratos para uma instancia,
// atualizando os dados de visualizacao de mapa e colmeia.
func HandlerRunCluster(instancia entity.Instancia) error {
	tag := "run_cluster"
	logger.Inicio(tag, "Instancia %d: processando...", instancia.ID)

	db, err := banco.ConectarInstancia(
		instancia.EnvDBHost, instancia.EnvDBPort,
		instancia.EnvDBUser, instancia.EnvDBPass, instancia.EnvDBName,
	)
	if err != nil {
		return fmt.Errorf("falha ao conectar na instancia %d: %w", instancia.ID, err)
	}
	defer banco.FecharConexaoInstancia(db, tag)

	if err := processarClusterContratos(db); err != nil {
		return fmt.Errorf("erro em processarClusterContratos: %w", err)
	}

	if err := processarContratosCoordenadas(db); err != nil {
		return fmt.Errorf("erro em processarContratosCoordenadas: %w", err)
	}

	logger.Sucesso(tag, "Instancia %d concluida", instancia.ID)
	return nil
}

func processarClusterContratos(db *sql.DB) error {
	tag := "run_cluster"

	numOn, numOff, err := repositorio.ContarConexoesCluster(db)
	if err != nil {
		return err
	}

	numContratos := numOn + numOff
	numBloqueados := repositorio.ContarBloqueados(db)
	numOSPendentes := repositorio.ContarOSPendentes(db)

	contratos, err := repositorio.BuscarClientesColmeia(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar clientes colmeia: %w", err)
	}

	desconexoes, err := repositorio.BuscarDesconexoes(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar desconexoes: %w", err)
	}

	var a []*clusterContrato

	if contratos == nil {
		a = append(a, nil)
	} else {
		for _, c := range contratos {
			item := montarItemCluster(c, desconexoes)
			a = append(a, item)
		}
	}

	clusterJSON, err := repositorio.JSONEncodeIdentico(a)
	if err != nil {
		return fmt.Errorf("erro ao codificar JSON do cluster: %w", err)
	}

	if err := repositorio.AtualizarClusterContratos(db, clusterJSON, numOn, numOff, numContratos, numBloqueados, numOSPendentes); err != nil {
		return fmt.Errorf("erro ao atualizar sgp_webservices (cluster): %w", err)
	}

	logger.Info(tag, "Cluster: %d contratos, %d online, %d offline, %d bloqueados, %d OS pendentes",
		numContratos, numOn, numOff, numBloqueados, numOSPendentes)

	return nil
}

func montarItemCluster(c repositorio.LinhaClienteColmeia, desconexoes map[int]int) *clusterContrato {
	nomeCliente := ""
	if c.PfNome.Valid {
		nomeCliente = c.PfNome.String
	}
	if c.PjRazaoSocial.Valid && c.PjRazaoSocial.String != "" && nomeCliente == "" {
		nomeCliente = c.PjRazaoSocial.String
	}

	logradouro := ""
	if c.Logradouro.Valid {
		logradouro = c.Logradouro.String
	}
	if c.LogradouroNumero.Valid {
		logradouro += ", " + c.LogradouroNumero.String
	}
	if c.LogradouroBairro.Valid {
		logradouro += ", " + c.LogradouroBairro.String
	}

	item := &clusterContrato{
		Token:      c.ContratoToken,
		Cliente:    nomeCliente,
		Logradouro: logradouro,
	}

	if d, ok := desconexoes[c.ContratoID]; ok && d >= 7 {
		item.Img = imgDesconexao
		if c.OrganizacaoID.Valid {
			orgID := int(c.OrganizacaoID.Int64)
			item.OrganizacaoID = &orgID
		}
	} else if c.Status == "Bloqueado" {
		item.Img = imgBloqueado
		if c.OrganizacaoID.Valid {
			orgID := int(c.OrganizacaoID.Int64)
			item.OrganizacaoID = &orgID
		}
	} else if c.WsUpdateSequencia == 0 {
		item.Img = imgPendenteAtivacao
	} else {
		if c.Conexao == "Offline" {
			item.Img = imgOffline
		} else {
			item.Img = imgOnline
		}
		if c.OrganizacaoID.Valid {
			orgID := int(c.OrganizacaoID.Int64)
			item.OrganizacaoID = &orgID
		}
	}

	return item
}

func processarContratosCoordenadas(db *sql.DB) error {
	tag := "run_cluster"

	clientes, err := repositorio.BuscarClientesCoordenadas(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar clientes coordenadas: %w", err)
	}

	desconexoes, err := repositorio.BuscarDesconexoes(db)
	if err != nil {
		return fmt.Errorf("erro ao buscar desconexoes: %w", err)
	}

	var gps []coordenadaGPS

	if clientes != nil {
		for _, c := range clientes {
			item, incluir := montarItemCoordenada(c, desconexoes)
			if incluir {
				gps = append(gps, item)
			}
		}
	}

	if gps == nil {
		gps = []coordenadaGPS{}
	}

	coordenadasJSON, err := repositorio.JSONEncodeIdentico(gps)
	if err != nil {
		return fmt.Errorf("erro ao codificar JSON das coordenadas: %w", err)
	}

	if err := repositorio.AtualizarCoordenadas(db, coordenadasJSON); err != nil {
		return fmt.Errorf("erro ao atualizar coordenadas_contratos_status: %w", err)
	}

	logger.Info(tag, "Coordenadas: %d marcadores processados", len(gps))

	return nil
}

func montarItemCoordenada(c repositorio.LinhaClienteCoordenada, desconexoes map[int]int) (coordenadaGPS, bool) {
	nomeCliente := ""
	if c.PfNome.Valid {
		nomeCliente = c.PfNome.String
	}
	if c.PjRazaoSocial.Valid && c.PjRazaoSocial.String != "" && nomeCliente == "" {
		nomeCliente = c.PjRazaoSocial.String
	}

	logradouro := ""
	if c.Logradouro.Valid {
		logradouro = c.Logradouro.String
	}
	if c.LogradouroNumero.Valid {
		logradouro += ", " + c.LogradouroNumero.String
	}
	if c.LogradouroBairro.Valid {
		logradouro += " - " + c.LogradouroBairro.String
	}

	ultimaAtividade := fuso.Agora()
	if c.DataHoraUltimaConexaoAtividade.Valid {
		if parsed, err := time.Parse("2006-01-02 15:04:05", c.DataHoraUltimaConexaoAtividade.String); err == nil {
			ultimaAtividade = parsed
		}
	}

	diasInativo := int(time.Since(ultimaAtividade).Hours() / 24)

	if diasInativo >= 5 {
		return coordenadaGPS{}, false
	}

	partes := strings.Split(c.LogradouroCoordenadasGPS, ",")
	lat := ""
	lon := ""
	if len(partes) > 0 {
		lat = strings.TrimSpace(partes[0])
	}
	if len(partes) > 1 {
		lon = strings.TrimSpace(partes[1])
	}

	icon := ""
	if d, ok := desconexoes[c.ContratoID]; ok && d >= 7 {
		if c.Conexao == "Online" {
			icon = "/assets/ISP/img/ico/exclamacao_laranja.png"
		} else {
			icon = "/assets/ISP/img/ico/exclamacao_vermelho.png"
		}
	} else if c.Conexao == "Online" {
		icon = "/assets/ISP/img/ico/Map-Marker-Flag-4-Right-Chartreuse-icon.png"
	} else {
		if diasInativo >= 1 && diasInativo <= 5 {
			icon = "/assets/ISP/img/ico/triangulo_amarelo.png"
		} else {
			icon = "/assets/ISP/img/ico/Map-Marker-Flag-4-Right-Pink-icon.png"
		}
	}

	orgID := 0
	if c.OrganizacaoID.Valid {
		orgID = int(c.OrganizacaoID.Int64)
	}

	marker := coordenadaGPS{
		Icon:              icon,
		InfoWindowContent: fmt.Sprintf("<strong>%s</strong><br><strong>Endereço:</strong> %s", nomeCliente, logradouro),
		Lat:               lat,
		Lon:               lon,
		Title:             nomeCliente,
		OrganizacaoID:     orgID,
	}

	return marker, true
}
