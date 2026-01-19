package src

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

// ConnectDB opens (or returns) a MySQL connection to the groupietracker database.
// Defaults: host 127.0.0.1, port 3306, user root, no password.
// Values can be overridden via environment variables: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME.
func ConnectDB() (*sql.DB, error) {
	if DB != nil {
		if err := DB.Ping(); err == nil {
			return DB, nil
		}
		// Si la connexion est morte, on la recrée
		DB.Close()
		DB = nil
	}

	host := getEnvOrDefault("DB_HOST", DefaultDBHost)
	port := getEnvOrDefault("DB_PORT", DefaultDBPort)
	user := getEnvOrDefault("DB_USER", DefaultDBUser)
	password := getEnvOrDefault("DB_PASSWORD", DefaultDBPassword)
	name := getEnvOrDefault("DB_NAME", DefaultDBName)

	log.Printf("Tentative de connexion à MySQL: %s@%s:%s/%s", user, host, port, name)

	// D'abord, se connecter sans spécifier la base pour pouvoir la créer si nécessaire
	dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true&charset=utf8mb4&loc=Local", user, password, host, port)
	tempDB, err := sql.Open("mysql", dsnWithoutDB)
	if err != nil {
		return nil, fmt.Errorf("échec ouverture connexion MySQL: %w", err)
	}
	defer tempDB.Close()

	// Vérifier que MySQL répond
	if err := tempDB.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL ne répond pas sur %s:%s - vérifiez que MySQL est démarré: %w", host, port, err)
	}

	// Vérifier si la base existe, sinon la créer
	var dbExists int
	err = tempDB.QueryRow("SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", name).Scan(&dbExists)
	if err != nil {
		return nil, fmt.Errorf("erreur vérification base de données: %w", err)
	}

	if dbExists == 0 {
		log.Printf("La base '%s' n'existe pas, création en cours...", name)
		_, err = tempDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", name))
		if err != nil {
			return nil, fmt.Errorf("impossible de créer la base '%s': %w", name, err)
		}
		log.Printf("Base '%s' créée avec succès", name)
	}

	// Maintenant se connecter à la base spécifique
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local", user, password, host, port, name)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("échec ouverture connexion à la base '%s': %w", name, err)
	}

	// Basic pool tuning; adjust if needed.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("échec ping base '%s': %w", name, err)
	}

	log.Printf("✅ Connecté à MySQL %s:%s (base: %s)", host, port, name)

	DB = db
	return DB, nil
}

// Migrate crée les tables nécessaires si elles n'existent pas encore.
func Migrate(db *sql.DB) error {
	// Table utilisateurs avec gestion de profil
	const usersTable = `
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    pseudo VARCHAR(255) DEFAULT NULL,
    bio TEXT DEFAULT NULL,
    photo_profil VARCHAR(500) DEFAULT NULL,
    role VARCHAR(20) DEFAULT 'user',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`

	if _, err := db.Exec(usersTable); err != nil {
		return fmt.Errorf("création table users: %w", err)
	}

	// Ajouter les colonnes si elles n'existent pas (pour les bases existantes)
	alterQueries := []string{
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS pseudo VARCHAR(255) DEFAULT NULL",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT DEFAULT NULL",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS photo_profil VARCHAR(500) DEFAULT NULL",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user'",
	}

	for _, query := range alterQueries {
		// MySQL n'a pas "IF NOT EXISTS" pour ALTER TABLE, donc on ignore les erreurs
		_, _ = db.Exec(query)
	}

	// Vérifier si la colonne role existe et créer un index si nécessaire
	var columnExists int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'role'").Scan(&columnExists)
	if err == nil && columnExists == 0 {
		// Ajouter la colonne role si elle n'existe pas
		_, _ = db.Exec("ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'user'")
	}

	return nil
}
