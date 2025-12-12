package src

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func InitDB() (*pgxpool.Pool, error) {
	// tente de charger .env puis db.env
	_ = godotenv.Load("db.env")

	// debug utile si la variable n'est pas trouvée
	wd, _ := os.Getwd()
	log.Printf("working dir: %s", wd)
	if _, err := os.Stat(filepath.Join(wd, ".env")); err == nil {
		log.Println(".env trouvé")
	} else if _, err := os.Stat(filepath.Join(wd, "db.env")); err == nil {
		log.Println("db.env trouvé")
	} else {
		log.Println("pas de .env/db.env trouvé dans le working dir")
	}

	conn := os.Getenv("DATABASE_URL")
	if conn == "" {
		return nil, fmt.Errorf("DATABASE_URL manquant (vérifie .env et exécute depuis la racine du projet)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
