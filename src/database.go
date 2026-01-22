package src

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDB() (*sql.DB, error) {
	if DB != nil {
		if err := DB.Ping(); err == nil {
			return DB, nil
		}
		DB.Close()
		DB = nil
	}

	host := getEnvOrDefault("DB_HOST", DefaultDBHost)
	port := getEnvOrDefault("DB_PORT", DefaultDBPort)
	user := getEnvOrDefault("DB_USER", DefaultDBUser)
	password := getEnvOrDefault("DB_PASSWORD", DefaultDBPassword)
	name := getEnvOrDefault("DB_NAME", DefaultDBName)

	dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true&charset=utf8mb4&loc=Local", user, password, host, port)
	tempDB, err := sql.Open("mysql", dsnWithoutDB)
	if err != nil {
		return nil, fmt.Errorf("échec ouverture connexion MySQL: %w", err)
	}
	defer tempDB.Close()

	tempDB.SetConnMaxLifetime(5 * time.Second)
	if err := tempDB.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL ne répond pas sur %s:%s - %w", host, port, err)
	}

	var dbExists int
	err = tempDB.QueryRow("SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", name).Scan(&dbExists)
	if err != nil {
		return nil, fmt.Errorf("erreur vérification base de données: %w", err)
	}

	if dbExists == 0 {
		_, err = tempDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", name))
		if err != nil {
			return nil, fmt.Errorf("impossible de créer la base '%s': %w", name, err)
		}
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local", user, password, host, port, name)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("échec ouverture connexion à la base '%s': %w", name, err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("échec ping base '%s': %w", name, err)
	}

	DB = db
	return DB, nil
}

func Migrate(db *sql.DB) error {
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

	alterQueries := []string{
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS pseudo VARCHAR(255) DEFAULT NULL",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT DEFAULT NULL",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS photo_profil VARCHAR(500) DEFAULT NULL",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at DATETIME DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP",
		"ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'user'",
	}

	for _, query := range alterQueries {
		_, _ = db.Exec(query)
	}

	var columnExists int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'users' AND COLUMN_NAME = 'role'").Scan(&columnExists)
	if err == nil && columnExists == 0 {
		_, _ = db.Exec("ALTER TABLE users ADD COLUMN role VARCHAR(20) DEFAULT 'user'")
	}

	return nil
}
