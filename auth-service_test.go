package main

import (
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"testing"
)

func TestGenerateAccessToken(t *testing.T) {
	userID := "test-user"
	clientIP := "127.0.0.1"

	token, err := GenerateAccessToken(userID, clientIP)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Проверим, что токен можно разобрать и claims совпадают
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	assert.NoError(t, err)
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, userID, claims["user_id"])
	assert.Equal(t, clientIP, claims["client_ip"])
}

func TestGenerateRefreshToken(t *testing.T) {
	token1, err := GenerateRefreshToken()
	assert.NoError(t, err)
	assert.Len(t, token1, 88) // Длина base64(512 бит)

	token2, err := GenerateRefreshToken()
	assert.NoError(t, err)

	assert.NotEqual(t, token1, token2) // Токены должны быть уникальны
}

func TestStoreRefreshToken(t *testing.T) {
	// Создаем mock для sql.DB
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	db = mockDB // Подменяем глобальную переменную db на mock

	userID := "test-user"
	// Генерируем токен длиной 88 символов
	token, err := GenerateRefreshToken()
	assert.NoError(t, err)
	clientIP := "127.0.0.1"

	// Ожидание SQL запроса
	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs(userID, sqlmock.AnyArg(), clientIP, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Выполнение функции
	err = StoreRefreshToken(userID, token, clientIP)
	assert.NoError(t, err)

	// Проверка всех ожиданий
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyRefreshToken(t *testing.T) {
	// Создаем mock для sql.DB
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	db = mockDB // Подменяем глобальную переменную db

	userID := "test-user"
	clientIP := "127.0.0.1"

	// Генерируем корректный токен длиной не менее 72 символов
	token, err := GenerateRefreshToken()
	assert.NoError(t, err)

	trimmedToken := token[:72]
	hashedToken, err := bcrypt.GenerateFromPassword([]byte(trimmedToken), bcrypt.DefaultCost)
	assert.NoError(t, err)

	// Ожидание запроса SELECT
	mock.ExpectQuery("SELECT refresh_token_hash, client_ip FROM refresh_tokens").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"refresh_token_hash", "client_ip"}).
			AddRow(hashedToken, clientIP))

	// Ожидание DELETE запроса
	mock.ExpectExec("DELETE FROM refresh_tokens").
		WithArgs(userID, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Выполнение функции
	valid, err := VerifyRefreshToken(userID, token, clientIP)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Проверка всех ожиданий
	assert.NoError(t, mock.ExpectationsWereMet())
}
