package main

import (
	"groupietracker/src"
	"log"
)

func main() {
	// Initialiser la connexion à la DB
	pool, err := src.InitDB()
	if err != nil {
		log.Fatalf("initialisation DB impossible: %v", err)
	}
	defer pool.Close()
	log.Println("Connexion DB OK")

	// Démarrer le serveur
	srv, err := src.NewServer()
	if err != nil {
		log.Fatalf("initialisation impossible: %v", err)
	}
	if err := srv.Start(); err != nil {
		log.Fatalf("serveur arrêté: %v", err)
	}
}
