package main

import (
	"groupietracker/src"
	"log"
)

func main() {
	db, err := src.ConnectDB()
	if err != nil {
		log.Fatalf("connexion base de données impossible: %v", err)
	}
	if err := src.Migrate(db); err != nil {
		log.Fatalf("migration base de données impossible: %v", err)
	}
	defer db.Close()

	srv, err := src.NewServer()
	if err != nil {
		log.Fatalf("initialisation impossible: %v", err)
	}
	if err := srv.Start(); err != nil {
		log.Fatalf("serveur arrêté: %v", err)
	}
}
