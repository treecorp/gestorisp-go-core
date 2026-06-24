package logger

import (
	"fmt"
	"time"
)

// Hooks executados apos cada log
var logHooks []func(nivel, tag, msg string)

// AdicionarHook registra um callback executado apos cada chamada de log
func AdicionarHook(fn func(nivel, tag, msg string)) {
	logHooks = append(logHooks, fn)
}

func executarHooks(nivel, tag, msg string) {
	for _, fn := range logHooks {
		fn(nivel, tag, msg)
	}
}

// Cores ANSI
const (
	corResetar  = "\033[0m"
	corAzul     = "\033[1;34m"
	corVerde    = "\033[1;32m"
	corAmarelo  = "\033[1;33m"
	corVermelho = "\033[1;31m"
	corCiano    = "\033[1;36m"
	corMagenta  = "\033[1;35m"
)

// Icones
const (
	iconeInfo    = " ℹ "
	iconeSucesso = " ✔ "
	iconeAviso   = " ⚠ "
	iconeErro    = " ✘ "
	iconeInicio  = " ▶ "
	iconeBanco   = " 🗄 "
	iconeFilas   = " 📨 "
	iconeSeta    = " → "
)

// timestamp retorna o horario atual formatado
func timestamp() string {
	return time.Now().Format("2006/01/02 15:04:05")
}

// Info registra uma mensagem informativa (azul)
func Info(tag, msg string, args ...interface{}) {
	texto := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s[%s]%s %s%s%s\n",
		corCiano, timestamp(), corAzul, tag, corResetar,
		iconeInfo, texto, corResetar)
	executarHooks("info", tag, texto)
}

// Info registra uma mensagem informativa com destaque (ciano)
func Destaque(tag, msg string, args ...interface{}) {
	texto := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s[%s]%s %s%s%s\n",
		corCiano, timestamp(), corMagenta, tag, corResetar,
		iconeInfo, texto, corResetar)
	executarHooks("destaque", tag, texto)
}

// Sucesso registra uma mensagem de sucesso (verde)
func Sucesso(tag, msg string, args ...interface{}) {
	texto := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s[%s]%s %s%s%s\n",
		corCiano, timestamp(), corVerde, tag, corResetar,
		iconeSucesso, texto, corResetar)
	executarHooks("sucesso", tag, texto)
}

// Aviso registra uma mensagem de aviso (amarelo)
func Aviso(tag, msg string, args ...interface{}) {
	texto := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s[%s]%s %s%s%s\n",
		corCiano, timestamp(), corAmarelo, tag, corResetar,
		iconeAviso, texto, corResetar)
	executarHooks("aviso", tag, texto)
}

// Erro registra uma mensagem de erro (vermelho)
func Erro(tag, msg string, args ...interface{}) {
	texto := fmt.Sprintf(msg, args...)
	fmt.Printf("%s %s %s[%s]%s %s%s%s\n",
		corCiano, timestamp(), corVermelho, tag, corResetar,
		iconeErro, texto, corResetar)
	executarHooks("erro", tag, texto)
}

// Inicio registra o inicio de uma execucao (magenta)
func Inicio(tag, msg string, args ...interface{}) {
	texto := fmt.Sprintf(msg, args...)
	fmt.Printf("\n%s %s %s[%s]%s %s%s%s\n",
		corCiano, timestamp(), corMagenta, tag, corResetar,
		iconeInicio, texto, corResetar)
	executarHooks("inicio", tag, texto)
}
