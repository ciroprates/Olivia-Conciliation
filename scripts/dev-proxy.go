//go:build ignore

// dev-proxy: proxy reverso de desenvolvimento que replica o papel do nginx
// localmente, servindo tudo em uma única origem (same-origin).
//
// Por que existe: o frontend usa caminhos relativos (`/api`, `/executions` —
// veja frontend/constants.js) e o backend Go não emite headers CORS. Servir o
// frontend numa origem diferente do backend quebraria o fetch com credenciais.
// Este proxy serve os estáticos do frontend e encaminha /api e /executions,
// deixando tudo em http://localhost:3001 (o APP_ORIGIN do dev local).
//
// Uso (a partir da raiz do repo, com o backend rodando em :8080):
//
//	go run scripts/dev-proxy.go
//
// Flags:
//
//	-port      porta de escuta (padrão 3001)
//	-frontend  diretório dos estáticos (padrão ./frontend)
//	-backend   alvo do /api (padrão http://localhost:8080)
//	-exec      alvo do /executions (padrão http://localhost:3000)
//
// Para apontar o /executions para produção (que exige um cookie de sessão
// válido — mesmo JWT_SECRET), passe o host de prod:
//
//	go run scripts/dev-proxy.go -exec https://console.olivinha.online
package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// singleHostProxy encaminha para target, ajustando também o header Host — o
// director padrão não sobrescreve req.Host, o que quebra roteamento por
// server_name e SNI de TLS ao apontar para um host remoto (ex.: prod).
func singleHostProxy(target string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("alvo inválido %q: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	director := proxy.Director
	proxy.Director = func(r *http.Request) {
		director(r)
		r.Host = u.Host
	}
	return proxy
}

func main() {
	port := flag.String("port", "3001", "porta de escuta")
	frontend := flag.String("frontend", "./frontend", "diretório dos estáticos do frontend")
	backend := flag.String("backend", "http://localhost:8080", "alvo do /api")
	exec := flag.String("exec", "http://localhost:3000", "alvo do /executions")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/api/", singleHostProxy(*backend))
	mux.Handle("/executions/", singleHostProxy(*exec))
	mux.Handle("/", http.FileServer(http.Dir(*frontend)))

	addr := ":" + *port
	log.Printf("dev-proxy em http://localhost%s  (/api -> %s, /executions -> %s, estáticos <- %s)",
		addr, *backend, *exec, *frontend)
	log.Fatal(http.ListenAndServe(addr, mux))
}
