package auth

import (
"database/sql"
"errors"
"log"
"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	DB        *sql.DB
	SecretKey string
}

func NewAuthService(db *sql.DB, secretKey string) *AuthService {
	return &AuthService{
		DB:        db,
		SecretKey: secretKey,
	}
}

func (s *AuthService) Authenticate(username, password string) (*LoginResponse, error) {
	user, err := s.getUserByUsername(username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}
	
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid username or password")
	}
	
	token, expiresAt, err := GenerateToken(user, s.SecretKey)
	if err != nil {
		return nil, err
	}
	
	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
		},
	}, nil
}

func (s *AuthService) getUserByUsername(username string) (*User, error) {
	user := &User{}
	query := `SELECT id, username, email, password, role FROM users WHERE username = $1 AND is_active = true`
	
	err := s.DB.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role)
	if err != nil {
		return nil, err
	}
	
	return user, nil
}

func (s *AuthService) CreateUser(username, email, password, role string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS users (
id SERIAL PRIMARY KEY,
username VARCHAR(50) UNIQUE NOT NULL,
email VARCHAR(100) UNIQUE NOT NULL,
password VARCHAR(255) NOT NULL,
role VARCHAR(20) DEFAULT 'user',
is_active BOOLEAN DEFAULT true,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
	`
	
	if _, err := s.DB.Exec(createTableQuery); err != nil {
		return err
	}
	
	insertQuery := `INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4)`
	_, err = s.DB.Exec(insertQuery, username, email, string(hashedPassword), role)
	if err != nil {
		return err
	}
	
	log.Printf("[AUTH] User created: %s (%s)", username, role)
	return nil
}

func (s *AuthService) GetUserByID(userID int) (*UserInfo, error) {
	user := &UserInfo{}
	query := `SELECT id, username, email, role FROM users WHERE id = $1 AND is_active = true`
	
	err := s.DB.QueryRow(query, userID).Scan(&user.ID, &user.Username, &user.Email, &user.Role)
	if err != nil {
		return nil, err
	}
	
	return user, nil
}
