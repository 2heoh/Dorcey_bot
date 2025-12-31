#!/bin/bash

# Скрипт для запуска Binance Futures Telegram Bot
# Устанавливает переменные окружения и запускает бота

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Запуск Binance Futures Telegram Bot ===${NC}"

# Проверка наличия .env файла
if [ -f .env ]; then
    echo -e "${YELLOW}Загружаю переменные из .env файла...${NC}"
    export $(cat .env | grep -v '^#' | xargs)
else
    echo -e "${YELLOW}.env файл не найден, используем переменные из скрипта или окружения${NC}"
fi

# Установка переменных окружения (если не установлены)
# Раскомментируйте и заполните значения, если хотите задать их здесь
# export TELEGRAM_BOT_TOKEN="ваш_telegram_bot_token"
# export BINANCE_API_KEY="ваш_binance_api_key"
# export BINANCE_SECRET_KEY="ваш_binance_secret_key"

# Проверка обязательных переменных
if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
    echo -e "${RED}ОШИБКА: TELEGRAM_BOT_TOKEN не установлен${NC}"
    echo "Установите переменную окружения или создайте .env файл"
    exit 1
fi

if [ -z "$BINANCE_API_KEY" ]; then
    echo -e "${RED}ОШИБКА: BINANCE_API_KEY не установлен${NC}"
    echo "Установите переменную окружения или создайте .env файл"
    exit 1
fi

if [ -z "$BINANCE_SECRET_KEY" ]; then
    echo -e "${RED}ОШИБКА: BINANCE_SECRET_KEY не установлен${NC}"
    echo "Установите переменную окружения или создайте .env файл"
    exit 1
fi

echo -e "${GREEN}Все переменные окружения установлены${NC}"

# Проверка наличия Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}ОШИБКА: Go не установлен${NC}"
    echo "Установите Go 1.17 или выше"
    exit 1
fi

# Проверка версии Go
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}Найден Go версии: $GO_VERSION${NC}"

# Переход в директорию скрипта
cd "$(dirname "$0")"

# Запуск бота
echo -e "${GREEN}Запускаю бота...${NC}"
echo ""

# Проверка аргумента для лог-файла
if [ "$1" = "--log" ] || [ "$1" = "-l" ]; then
    LOG_FILE="${2:-bot.log}"
    echo -e "${YELLOW}Вывод будет сохранен в файл: $LOG_FILE${NC}"
    echo -e "${YELLOW}Для просмотра логов в реальном времени: tail -f $LOG_FILE${NC}"
    echo ""
    
    # Вариант 1: Запуск через go run с лог-файлом
    go run main.go 2>&1 | tee "$LOG_FILE"
    
    # Вариант 2: Запуск скомпилированного бинарника с лог-файлом (раскомментируйте, если используете)
    # if [ -f dorcey ]; then
    #     ./dorcey 2>&1 | tee "$LOG_FILE"
    # else
    #     echo -e "${YELLOW}Бинарник не найден, компилирую...${NC}"
    #     go build -o dorcey main.go
    #     ./dorcey 2>&1 | tee "$LOG_FILE"
    # fi
else
    # Вариант 1: Запуск через go run (для разработки)
    go run main.go
    
    # Вариант 2: Запуск скомпилированного бинарника (раскомментируйте, если используете)
    # if [ -f dorcey ]; then
    #     ./dorcey
    # else
    #     echo -e "${YELLOW}Бинарник не найден, компилирую...${NC}"
    #     go build -o dorcey main.go
    #     ./dorcey
    # fi
fi
