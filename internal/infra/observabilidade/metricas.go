package observabilidade

import "sync"

type HandlerMetrica struct {
	Total        int     `json:"total"`
	TempoMedioMs float64 `json:"tempo_medio_ms"`
	Erros        int     `json:"erros"`
}

type MetricasSnapshot struct {
	TempoMedioMs   float64                  `json:"tempo_medio_ms"`
	TotalExecucoes int                      `json:"total_execucoes"`
	TotalErros     int                      `json:"total_erros"`
	TaxaSucesso    float64                  `json:"taxa_sucesso"`
	PorHandler     map[string]HandlerMetrica `json:"por_handler"`
}

type Metricas struct {
	mu       sync.RWMutex
	entradas []LogEntry
}

const maxEntradas = 50000

func (m *Metricas) Ingerir(entry LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entradas = append(m.entradas, entry)
	if len(m.entradas) > maxEntradas {
		excesso := len(m.entradas) - maxEntradas
		m.entradas = m.entradas[excesso:]
	}
}

func (m *Metricas) Snapshot() MetricasSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var s MetricasSnapshot
	s.PorHandler = make(map[string]HandlerMetrica)

	totalTempo := int64(0)
	totalComTempo := 0

	for _, e := range m.entradas {
		s.TotalExecucoes++
		if e.Nivel == "erro" {
			s.TotalErros++
		}
		if e.DuracaoMs > 0 {
			totalTempo += e.DuracaoMs
			totalComTempo++
		}

		h := s.PorHandler[e.Tag]
		h.Total++
		if e.Nivel == "erro" {
			h.Erros++
		}
		if h.Total == 1 {
			h.TempoMedioMs = float64(e.DuracaoMs)
		} else if e.DuracaoMs > 0 {
			h.TempoMedioMs = ((h.TempoMedioMs * float64(h.Total-1)) + float64(e.DuracaoMs)) / float64(h.Total)
		}
		s.PorHandler[e.Tag] = h
	}

	if totalComTempo > 0 {
		s.TempoMedioMs = float64(totalTempo) / float64(totalComTempo)
	}
	if s.TotalExecucoes > 0 {
		s.TaxaSucesso = float64(s.TotalExecucoes-s.TotalErros) / float64(s.TotalExecucoes) * 100
	}

	return s
}
