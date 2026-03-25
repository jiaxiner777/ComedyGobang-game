package main

import (
	"log"
	"net/http"
	"os"

	"codebreaker/internal/api"
)

func main() {
	addr := os.Getenv("CODEBREAKER_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := api.NewServer()
	log.Printf("Code Breaker web server listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}
