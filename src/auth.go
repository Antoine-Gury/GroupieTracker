package src

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int
	Username     string
	Email        string
	PasswordHash string
	Pseudo       sql.NullString
	Bio          sql.NullString
	PhotoProfil  sql.NullString
	Role         string
	CreatedAt    time.Time
	UpdatedAt    sql.NullTime
}

func hashPassword(password string) (string, error) {
	const cost = 12
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// checkPassword compare un mot de passe en clair au hash stocké.
func checkPassword(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func CreateUser(db *sql.DB, email, password string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return errors.New("email requis")
	}
	if len(password) < 8 {
		return errors.New("mot de passe trop court (8 caractères min)")
	}

	username := email
	if idx := strings.Index(email, "@"); idx > 0 {
		username = email[:idx]
	}

	hashed, err := hashPassword(password)
	if err != nil {
		return err
	}

	const query = `INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`
	if _, err := db.Exec(query, username, email, hashed); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return fmt.Errorf("un compte existe déjà avec cet email")
		}
		return fmt.Errorf("création utilisateur: %w", err)
	}
	return nil
}

func GetUserByEmail(db *sql.DB, email string) (User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var u User
	var role sql.NullString
	const query = `SELECT id, username, email, password_hash, pseudo, bio, photo_profil, role, created_at, updated_at FROM users WHERE email = ? LIMIT 1`
	if err := db.QueryRow(query, email).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Pseudo, &u.Bio, &u.PhotoProfil, &role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, fmt.Errorf("utilisateur introuvable")
		}
		return User{}, fmt.Errorf("lecture utilisateur: %w", err)
	}
	if role.Valid {
		u.Role = role.String
	} else {
		u.Role = "user"
	}
	return u, nil
}

func GetUserByID(db *sql.DB, id int) (User, error) {
	var u User
	var role sql.NullString
	const query = `SELECT id, username, email, password_hash, pseudo, bio, photo_profil, role, created_at, updated_at FROM users WHERE id = ? LIMIT 1`
	if err := db.QueryRow(query, id).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Pseudo, &u.Bio, &u.PhotoProfil, &role, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, fmt.Errorf("utilisateur introuvable")
		}
		return User{}, fmt.Errorf("lecture utilisateur: %w", err)
	}
	if role.Valid {
		u.Role = role.String
	} else {
		u.Role = "user"
	}
	return u, nil
}

func UpdateUserProfile(db *sql.DB, userID int, pseudo, bio, photoProfil string) error {
	const query = `UPDATE users SET pseudo = ?, bio = ?, photo_profil = ?, updated_at = NOW() WHERE id = ?`
	_, err := db.Exec(query, pseudo, bio, photoProfil, userID)
	if err != nil {
		return fmt.Errorf("mise à jour profil: %w", err)
	}
	return nil
}

func GetAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`SELECT id, username, email, password_hash, pseudo, bio, photo_profil, role, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("liste utilisateurs: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var role sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Pseudo, &u.Bio, &u.PhotoProfil, &role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan utilisateur: %w", err)
		}
		if role.Valid {
			u.Role = role.String
		} else {
			u.Role = "user"
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func UpdateUserRole(db *sql.DB, userID int, role string) error {
	if role != "user" && role != "admin" {
		return fmt.Errorf("rôle invalide: %s", role)
	}
	const query = `UPDATE users SET role = ?, updated_at = NOW() WHERE id = ?`
	_, err := db.Exec(query, role, userID)
	if err != nil {
		return fmt.Errorf("mise à jour rôle: %w", err)
	}
	return nil
}

func DeleteUser(db *sql.DB, userID int) error {
	const query = `DELETE FROM users WHERE id = ?`
	_, err := db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("suppression utilisateur: %w", err)
	}
	return nil
}


