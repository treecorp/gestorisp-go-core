package fuso

import "time"

var LocalBR = func() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic("fuso: falha ao carregar America/Sao_Paulo: " + err.Error())
	}
	return loc
}()

func Agora() time.Time {
	return time.Now().In(LocalBR)
}
