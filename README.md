# auth-service

Код представляет собой реализацию части сервиса аутентификации с использованием технологий Go, JWT, Fiber, PostgreSQL и bcrypt. Рассмотрим работу каждого блока кода:
К проекту таже приложены uni-тесты: запуск осуществляется с помощью команды ```go test```
---

### Необходимо создать таблицу в базе данных PostgreSQL

```
CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    refresh_token_hash TEXT NOT NULL,
    client_ip TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);
```

### **1. Импорты**
Код использует:
- **`crypto/rand`**, **`crypto/sha512`**: для генерации безопасных случайных данных и хэшей.
- **`database/sql`**, **`github.com/lib/pq`**: для работы с PostgreSQL.
- **`jwt-go`**: для создания и проверки Access токенов.
- **`bcrypt`**: для хэширования и проверки Refresh токенов.
- **`fiber/v2`**: для создания REST API.
- **`validator`**: для валидации входных данных.
- **`godotenv`**: для работы с `.env` файлами.

---

### **2. Инициализация и конфигурация**
1. **Переменные окружения**:
   - Загружаются с помощью `godotenv`.
   - `JWT_SECRET` (секрет для подписи JWT) и `DB_CONN` (строка подключения к БД) извлекаются из `.env`.

2. **Инициализация базы данных**:
   - Функция `initDB` устанавливает соединение с PostgreSQL.

3. **Переменные**:
   - `db` — подключение к базе данных.
   - `jwtSecret` — секретный ключ для подписи JWT.
   - `validate` — объект для проверки входных данных.

---

### **3. Генерация токенов**
1. **Access токен (`GenerateAccessToken`)**:
   - Создается JWT токен с:
     - `user_id` — ID пользователя.
     - `client_ip` — IP адрес клиента.
     - `exp` — время истечения (15 минут).
   - Подписывается с использованием `HS512`.

2. **Refresh токен (`GenerateRefreshToken`)**:
   - Генерируется случайная последовательность байт.
   - Хэшируется с помощью SHA512.
   - Кодируется в base64 для передачи клиенту.

---

### **4. Работа с Refresh токенами**
1. **Сохранение токена (`StoreRefreshToken`)**:
   - Токен обрезается до 72 байт (ограничение bcrypt).
   - Хэшируется с использованием bcrypt.
   - Сохраняется в БД с `user_id`, IP адресом клиента, временем создания.

2. **Проверка токена (`VerifyRefreshToken`)**:
   - Извлекается последний сохраненный токен пользователя из БД.
   - Проверяется соответствие хэша и токена.
   - Проверяется IP адрес клиента:
     - Если изменился, выводится предупреждение (моковая отправка email).
   - Удаляется использованный Refresh токен.

---

### **5. REST API маршруты**
#### **Маршрут `/api/auth/token` (получение токенов)**:
- Принимает JSON с `user_id` (UUID).
- Генерирует Access и Refresh токены.
- Сохраняет Refresh токен в БД.
- Возвращает клиенту оба токена в JSON.

#### **Маршрут `/api/auth/refresh` (обновление токена)**:
- Принимает JSON с `user_id` и Refresh токеном.
- Проверяет Refresh токен:
  - Соответствие хэша.
  - Совпадение IP адреса.
- Удаляет использованный Refresh токен.
- Генерирует новый Access токен.
- Возвращает новый Access токен в JSON.

---

### **6. Валидация и обработка ошибок**
- **`validator`** проверяет корректность входных данных (например, `user_id` должен быть UUID).
- Обработка ошибок реализована через:
  - Логи (`log.Printf`).
  - Возврат статусов HTTP (например, 400, 401, 500).

---

### **7. Основная функция (`main`)**
1. Инициализируется база данных.
2. Настраиваются маршруты API.
3. Запускается HTTP сервер на порту 8080.

---

### **8. Пример работы**
1. **Получение токенов**:
   - Клиент отправляет запрос с `user_id`.
   - Сервер возвращает пару Access/Refresh токенов.

2. **Обновление токенов**:
   - Клиент отправляет запрос с `user_id` и Refresh токеном.
   - Сервер проверяет Refresh токен, удаляет его, выдает новый Access токен.

---

### **Особенности реализации**
1. **Безопасность**:
   - Access токены подписаны и не хранятся в базе.
   - Refresh токены хранятся только в виде хэша.
   - Использован bcrypt для защиты от утечек токенов.
   - Удаление использованных токенов предотвращает повторное использование.

2. **Работа с IP**:
   - Сервер проверяет IP клиента при обновлении токенов.
   - Предупреждает о смене IP через моковую отправку email.

3. **Простота тестирования**:
   - Четко разделенные функции и маршруты.
   - Возможность заменить моковые данные реальными (например, отправку email).

## Тестирование

1. Убедитесь, что база данных PostgreSQL запущена.
2. Выполните запрос с помощью curl:

Пример CURL-запроса:
```bash
curl -X POST http://localhost:8080/api/auth/token \
-H "Content-Type: application/json" \
-d '{"user_id": "550e8400-e29b-41d4-a716-446655440000"}'
```

Привер вывода CURL-запроса: 
```bash
{"access_token":"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaXAiOiIxMjcuMC4wLjEiLCJleHAiOjE3MzQwOTU3NjksInVzZXJfaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ.ZdF9GeQmNOThAJyF9EUJSva_hZNEFxWbOCyHiI4GoaWwEr3tY3465tXU3c0uq01c1J-gGNa15RhXH_nwC97_9w","refresh_token":"NwApCO__vYuy6SayJCSodQsRWy_EQvtfMRTjurPsf4oVLuGhkeALZI0IqdzvkXgsSQHLL0oWircAPn8AEPFsnQ=="}
```

## Это значит что запрос выполнен успешно, и сервер вернул пару токенов:

1. **Access Token**:  
   ```json
   "eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaXAiOiIxMjcuMC4wLjEiLCJleHAiOjE3MzQwOTU3NjksInVzZXJfaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ.ZdF9GeQmNOThAJyF9EUJSva_hZNEFxWbOCyHiI4GoaWwEr3tY3465tXU3c0uq01c1J-gGNa15RhXH_nwC97_9w"
   ```

   Этот токен представляет собой **JWT**, содержащий:
   - `client_ip`: IP-адрес клиента (в данном случае `127.0.0.1`).
   - `exp`: Временную метку истечения срока действия токена.
   - `user_id`: Уникальный идентификатор пользователя.

2. **Refresh Token**:  
   ```json
   "NwApCO__vYuy6SayJCSodQsRWy_EQvtfMRTjurPsf4oVLuGhkeALZI0IqdzvkXgsSQHLL0oWircAPn8AEPFsnQ=="
   ```

   Этот токен используется для обновления Access Token без необходимости повторной аутентификации пользователя.

### Следующие шаги
1. **Для проверки токенов:**
   - Access Token можно проверить с помощью любого JWT-декодера, например, на сайте [jwt.io](https://jwt.io).
   - Refresh Token проверяется сервером, который использует сохраненные значения в базе данных.

2. **Для обновления Access Token:**
   Выполните запрос на `/api/auth/refresh`, передав `user_id` и `refresh_token`.

Пример CURL-запроса, `важная заметка refresh_token каждый раз уникальный`:
```bash
curl -X POST http://localhost:8080/api/auth/refresh \
-H "Content-Type: application/json" \
-d '{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "refresh_token": "NwApCO__vYuy6SayJCSodQsRWy_EQvtfMRTjurPsf4oVLuGhkeALZI0IqdzvkXgsSQHLL0oWircAPn8AEPFsnQ=="
}'
```
Привер вывода CURL-запроса:
```bash
{"access_token":"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaXAiOiIxMjcuMC4wLjEiLCJleHAiOjE3MzQwOTYxMzYsInVzZXJfaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ.eFFfkeh4Bi3gzeoltDQItsjnMWObpvJnAnVp8pW7z7JLDdSX_KTZB63AHeaOuZP0rX8OW9MuftkNzUDcg7JMBw"}
```

## По скольку запрос на обновление токена успешно выполнен! Сервер вернул новый **Access Token**:

### Новый Access Token:
```json
"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJjbGllbnRfaXAiOiIxMjcuMC4wLjEiLCJleHAiOjE3MzQwOTYxMzYsInVzZXJfaWQiOiI1NTBlODQwMC1lMjliLTQxZDQtYTcxNi00NDY2NTU0NDAwMDAifQ.eFFfkeh4Bi3gzeoltDQItsjnMWObpvJnAnVp8pW7z7JLDdSX_KTZB63AHeaOuZP0rX8OW9MuftkNzUDcg7JMBw"
```

### Обновленный Access Token включает:
- **`client_ip`**: `127.0.0.1` — IP клиента.
- **`exp`**: Временная метка истечения нового токена.
- **`user_id`**: `550e8400-e29b-41d4-a716-446655440000` — уникальный идентификатор пользователя.

### Поведение:
- Предыдущий Refresh Token был успешно использован, удален из базы, и новый Access Token был сгенерирован.
- Если вы снова захотите обновить Access Token, потребуется новый Refresh Token.


## Итог:

### Проверим соблюдение требований задания:

---

#### **1. Используемые технологии**
- **Go**: Код написан на языке Go.
- **JWT**: Access токен формируется с использованием библиотеки `github.com/dgrijalva/jwt-go` и алгоритма SHA512.
- **PostgreSQL**: Используется для хранения Refresh токенов в виде bcrypt хэша.

Требования соблюдены.

---

#### **2. Реализация REST маршрутов**
- **Первый маршрут**:
  - Принимает GUID пользователя (`user_id`).
  - Генерирует пару Access/Refresh токенов.
  - Access токен содержит сведения об IP клиента и срок действия.
  - Refresh токен генерируется, хэшируется через bcrypt и сохраняется в базу.

- **Второй маршрут**:
  - Принимает GUID пользователя и Refresh токен.
  - Проверяет Refresh токен, сверяя bcrypt хэш с базой.
  - Проверяет, совпадает ли IP адрес.
  - Удаляет использованный Refresh токен после валидации.
  - Генерирует новый Access токен и возвращает его клиенту.

Оба маршрута реализованы корректно.

---

#### **3. Хранение Access токена**
- Access токен хранится только на стороне клиента и не сохраняется в базу данных.

Требование соблюдено.

---

#### **4. Хранение Refresh токена**
- Refresh токен формируется в формате base64.
- В базе хранится только bcrypt хэш токена.
- Реализована защита от повторного использования путем удаления токена из базы после успешной проверки.

Все требования выполнены.

---

#### **5. Связь Access и Refresh токенов**
- Access и Refresh токены связаны через `user_id` и выдаются в паре.
- Проверка Refresh токена включает:
  - Сравнение хэша токена.
  - Сравнение IP адреса.
  - Удаление использованного токена.

Связь реализована корректно.

---

#### **6. Валидация IP адреса**
- При изменении IP адреса во время Refresh операции:
  - Генерируется предупреждение через `fmt.Printf` (моковая отправка email).

Реализовано, соответствует заданию.

---

