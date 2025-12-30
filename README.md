# Binance Futures Telegram Bot

Телеграм-бот на Golang для отслеживания открытых позиций на Binance Futures.

## Возможности

- Просмотр открытых позиций на Binance Futures
- Отображение времени сделки в часах и минутах
- Информация о размере позиции, цене входа, марже и PnL

## Требования

- Go 1.21 или выше
- Telegram Bot Token (получить у [@BotFather](https://t.me/BotFather))
- Binance API Key и Secret Key (создать на [Binance API Management](https://www.binance.com/en/my/settings/api-management))

## Установка

1. Клонируйте репозиторий или скачайте файлы проекта

2. Установите зависимости:
```bash
go mod download
```

3. Установите переменные окружения:

### Как получить Telegram Bot Token:

1. Откройте Telegram и найдите бота [@BotFather](https://t.me/BotFather)
2. Отправьте команду `/newbot` или `/start`
3. Укажите имя вашего бота (например: "Binance Futures Tracker")
4. Укажите username бота (должен заканчиваться на `bot`, например: `my_binance_bot`)
5. BotFather пришлет вам токен вида `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`
6. Скопируйте этот токен и используйте его в переменной окружения

```bash
export TELEGRAM_BOT_TOKEN="ваш_telegram_bot_token"
export BINANCE_API_KEY="ваш_binance_api_key"
export BINANCE_SECRET_KEY="ваш_binance_secret_key"
```

Или создайте файл `.env` (не забудьте добавить его в `.gitignore`):
```bash
TELEGRAM_BOT_TOKEN=ваш_telegram_bot_token
BINANCE_API_KEY=ваш_binance_api_key
BINANCE_SECRET_KEY=ваш_binance_secret_key
```

## Запуск

```bash
go run main.go
```

Или скомпилируйте и запустите:
```bash
go build -o binance-bot
./binance-bot
```

## Использование

1. Найдите вашего бота в Telegram по имени, которое вы указали при создании у @BotFather
2. Отправьте команду `/start` для начала работы
3. Используйте команду `/positions` для просмотра открытых позиций

## Команды

- `/start` - Начать работу с ботом
- `/positions` - Показать список открытых позиций на Futures

## Настройка Binance API

Для работы бота необходимо:

1. Создать API ключ на Binance:
   - Перейдите на [Binance API Management](https://www.binance.com/en/my/settings/api-management)
   - Нажмите "Create API"
   - Выберите "System generated" (рекомендуется) или "Self-generated"
   - Подтвердите создание через email и 2FA

2. Настройте права доступа:
   - **Обязательно**: Включите "Enable Reading" для Futures
   - **Важно**: Убедитесь, что вы создаете ключ для **Futures**, а не для Spot
   - Для чтения позиций достаточно прав на чтение (Enable Reading)

3. Настройка IP whitelist (опционально, но рекомендуется):
   - Если включен IP whitelist, добавьте IP адрес вашего сервера
   - Если не уверены в IP, временно отключите whitelist для тестирования
   - Ваш текущий IP можно узнать из сообщения об ошибке

## Решение проблем

### Ошибка: "Invalid API-key, IP, or permissions for action" (код -2015)

Эта ошибка означает проблему с авторизацией. Проверьте:

1. **Правильность ключей**:
   - Убедитесь, что `BINANCE_API_KEY` и `BINANCE_SECRET_KEY` скопированы правильно
   - Проверьте, что нет лишних пробелов или символов

2. **Права доступа**:
   - Перейдите в настройки API ключа на Binance
   - Убедитесь, что включено "Enable Reading" для **Futures**
   - Проверьте, что ключ создан для Futures, а не для Spot

3. **IP whitelist**:
   - Если включен IP whitelist, добавьте ваш IP адрес
   - Или временно отключите whitelist для тестирования
   - IP адрес указан в сообщении об ошибке

4. **Тип API ключа**:
   - Убедитесь, что используете Futures API ключ
   - Spot API ключи не работают с Futures API

## Структура проекта

```
.
├── main.go          # Основной файл с логикой бота
├── go.mod           # Файл зависимостей Go
├── Requirements.md  # Требования к проекту
└── README.md        # Документация
```

## Зависимости

- `github.com/adshao/go-binance/v2` - Клиент для Binance API
- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Клиент для Telegram Bot API

## Безопасность

⚠️ **Важно**: Никогда не публикуйте ваши API ключи в публичных репозиториях. Используйте переменные окружения или файлы конфигурации, которые добавлены в `.gitignore`.
