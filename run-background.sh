#!/bin/bash

# Скрипт для запуска бота в фоновом режиме с лог-файлом
# Устанавливает переменные окружения и запускает бота в фоне

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

LOG_FILE="${1:-bot.log}"
PID_FILE="bot.pid"

echo -e "${GREEN}=== Запуск Binance Futures Telegram Bot в фоновом режиме ===${NC}"

# Проверка, не запущен ли уже бот (по имени процесса)
EXISTING_PIDS=$(pgrep -f "dorcey|go run main.go" 2>/dev/null)
if [ -n "$EXISTING_PIDS" ]; then
    echo -e "${RED}ОШИБКА: Обнаружены запущенные экземпляры бота!${NC}"
    echo -e "${YELLOW}Найденные процессы:${NC}"
    ps -p $EXISTING_PIDS -o pid,cmd 2>/dev/null || echo "$EXISTING_PIDS"
    echo ""
    echo -e "${YELLOW}Для остановки всех экземпляров используйте:${NC}"
    echo "  ./stop-bot.sh"
    echo "или"
    echo "  pkill -f 'dorcey|go run main.go'"
    exit 1
fi

# Проверка PID файла (на случай, если процесс завершился, но файл остался)
if [ -f "$PID_FILE" ]; then
    OLD_PID=$(cat "$PID_FILE")
    if ps -p "$OLD_PID" > /dev/null 2>&1; then
        echo -e "${YELLOW}Бот уже запущен (PID: $OLD_PID)${NC}"
        echo "Для остановки используйте: kill $OLD_PID"
        exit 1
    else
        echo -e "${YELLOW}Удаляю устаревший PID файл${NC}"
        rm -f "$PID_FILE"
    fi
fi

# Загрузка переменных из .env файла
if [ -f .env ]; then
    echo -e "${YELLOW}Загружаю переменные из .env файла...${NC}"
    export $(cat .env | grep -v '^#' | xargs)
fi

# Проверка обязательных переменных
if [ -z "$TELEGRAM_BOT_TOKEN" ] || [ -z "$BINANCE_API_KEY" ] || [ -z "$BINANCE_SECRET_KEY" ]; then
    echo -e "${RED}ОШИБКА: Не все переменные окружения установлены${NC}"
    echo "Установите: TELEGRAM_BOT_TOKEN, BINANCE_API_KEY, BINANCE_SECRET_KEY"
    exit 1
fi

# Проверка наличия Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}ОШИБКА: Go не установлен${NC}"
    exit 1
fi

# Переход в директорию скрипта
cd "$(dirname "$0")"

# Компиляция (если нужно)
if [ ! -f dorcey ]; then
    echo -e "${YELLOW}Компилирую проект...${NC}"
    go build -o dorcey main.go
    if [ $? -ne 0 ]; then
        echo -e "${RED}ОШИБКА: Ошибка компиляции${NC}"
        exit 1
    fi
fi

# Запуск в фоне
echo -e "${GREEN}Запускаю бота в фоновом режиме...${NC}"
echo -e "${YELLOW}Лог-файл: $LOG_FILE${NC}"
echo -e "${YELLOW}PID файл: $PID_FILE${NC}"
echo ""

# Запуск с перенаправлением вывода в лог-файл
nohup ./dorcey > "$LOG_FILE" 2>&1 &
BOT_PID=$!

# Сохранение PID
echo $BOT_PID > "$PID_FILE"

# Небольшая задержка для проверки, что процесс запустился
sleep 1

if ps -p "$BOT_PID" > /dev/null 2>&1; then
    echo -e "${GREEN}Бот успешно запущен!${NC}"
    echo -e "${GREEN}PID: $BOT_PID${NC}"
    echo -e "${GREEN}Лог-файл: $LOG_FILE${NC}"
    echo ""
    echo "Полезные команды:"
    echo "  Просмотр логов: tail -f $LOG_FILE"
    echo "  Остановка бота: kill $BOT_PID"
    echo "  Проверка статуса: ps -p $BOT_PID"
else
    echo -e "${RED}ОШИБКА: Не удалось запустить бота${NC}"
    echo "Проверьте лог-файл: $LOG_FILE"
    rm -f "$PID_FILE"
    exit 1
fi
