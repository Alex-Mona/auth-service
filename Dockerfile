# Используем официальное изображение Golang
FROM golang:1.23.1

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum и устанавливаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Компилируем бинарник
RUN go build -o auth-service .

# Устанавливаем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./auth-service"]
