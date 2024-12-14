package main

import (
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Пара токенов: Access и Refresh
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Структура для хранения записи Refresh токена
type RefreshTokenEntry struct {
	UserID           string
	RefreshTokenHash string
	ClientIP         string
	CreatedAt        time.Time
}

var db *sql.DB
var jwtSecret = []byte("supersecret")
var validate = validator.New()

func init() {
	// Загрузка переменных окружения
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
}

// Инициализация подключения к базе данных
func initDB() (*sql.DB, error) {
	connStr := os.Getenv("DB_CONN")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, db.Ping()
}

// Генерация Access токена (JWT)
func GenerateAccessToken(userID, clientIP string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":   userID,
		"client_ip": clientIP,
		"exp":       time.Now().Add(time.Minute * 15).Unix(), // Токен действителен 15 минут
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	return token.SignedString(jwtSecret)
}

// Генерация Refresh токена (случайные данные с хешированием SHA512)
func GenerateRefreshToken() (string, error) {
	raw := make([]byte, 64)
	_, err := rand.Read(raw) // Используйте rand для генерации случайных данных
	if err != nil {
		return "", err
	}
	hash := sha512.Sum512(raw)
	return base64.URLEncoding.EncodeToString(hash[:]), nil
}

// Сохранение Refresh токена в базе данных
func StoreRefreshToken(userID, token, clientIP string) error {
	// Обрезаем длину токена до 72 символов для безопасности
	trimmedToken := token[:72]

	hash, err := bcrypt.GenerateFromPassword([]byte(trimmedToken), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing refresh token: %v", err)
		return err
	}

	log.Printf("Storing refresh token for user_id=%s, client_ip=%s", userID, clientIP)
	_, err = db.Exec("INSERT INTO refresh_tokens (user_id, refresh_token_hash, client_ip, created_at) VALUES ($1, $2, $3, $4)", userID, hash, clientIP, time.Now())
	if err != nil {
		log.Printf("Error executing INSERT: %v", err)
		return err
	}

	log.Println("Refresh token successfully stored")
	return nil
}

// Проверка Refresh токена и удаление после использования
func VerifyRefreshToken(userID, token, clientIP string) (bool, error) {
	var entry RefreshTokenEntry
	// Получаем последнюю запись токена для пользователя
	err := db.QueryRow("SELECT refresh_token_hash, client_ip FROM refresh_tokens WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1", userID).Scan(&entry.RefreshTokenHash, &entry.ClientIP)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, errors.New("refresh token not found")
		}
		return false, err
	}
	// Проверка хешированного токена
	if err := bcrypt.CompareHashAndPassword([]byte(entry.RefreshTokenHash), []byte(token)); err != nil {
		return false, errors.New("invalid refresh token")
	}
	// Проверка изменения IP-адреса
	if entry.ClientIP != clientIP {
		// Mock email пример логики предупреждения об изменении IP
		fmt.Printf("Warning: IP address changed for user %s from %s to %s\n", userID, entry.ClientIP, clientIP)
	}

	// Удаляем использованный токен
	_, err = db.Exec("DELETE FROM refresh_tokens WHERE user_id = $1 AND refresh_token_hash = $2", userID, entry.RefreshTokenHash)
	if err != nil {
		return false, err
	}

	return true, nil
}

func main() {
	var err error
	db, err = initDB() // Подключение к базе данных
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	app := fiber.New()
	// Маршрут для получения пары токенов
	app.Post("/api/auth/token", func(c *fiber.Ctx) error {
		type Request struct {
			UserID string `json:"user_id" validate:"required,uuid4"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
		}

		if err := validate.Struct(req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed"})
		}

		clientIP, _, _ := net.SplitHostPort(c.Context().RemoteAddr().String())
		accessToken, err := GenerateAccessToken(req.UserID, clientIP)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate access token"})
		}
		refreshToken, err := GenerateRefreshToken()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate refresh token"})
		}
		if err := StoreRefreshToken(req.UserID, refreshToken, clientIP); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to store refresh token"})
		}
		return c.JSON(TokenPair{AccessToken: accessToken, RefreshToken: refreshToken})
	})
	// Маршрут для обновления Access токена
	app.Post("/api/auth/refresh", func(c *fiber.Ctx) error {
		type Request struct {
			UserID       string `json:"user_id" validate:"required,uuid4"`
			RefreshToken string `json:"refresh_token" validate:"required"`
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
		}

		if err := validate.Struct(req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Validation failed"})
		}

		clientIP, _, _ := net.SplitHostPort(c.Context().RemoteAddr().String())
		valid, err := VerifyRefreshToken(req.UserID, req.RefreshToken, clientIP)
		if err != nil || !valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid refresh token"})
		}
		accessToken, err := GenerateAccessToken(req.UserID, clientIP)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate access token"})
		}
		return c.JSON(fiber.Map{"access_token": accessToken})
	})
	// Запуск сервера на порту 8080
	app.Listen(":8080")
}
