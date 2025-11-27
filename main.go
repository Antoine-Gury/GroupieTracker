package main

import (
	"groupietracker/src"
	"log"
)

func main() {
	srv, err := src.NewServer()
	if err != nil {
		log.Fatalf("initialisation impossible: %v", err)
	}
	if err := srv.Start(); err != nil {
		log.Fatalf("serveur arrêté: %v", err)
	}
}
