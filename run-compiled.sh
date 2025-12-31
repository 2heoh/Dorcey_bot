#!/bin/bash

# Скрипт для запуска скомпилированного бинарника
# Сначала компилирует проект, затем запускает

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Компиляция и запуск Binance Futures Telegram Bot ===${NC}"

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

# Переход в директорию скрипта
cd "$(dirname "$0")"

# Компиляция
echo -e "${YELLOW}Компилирую проект...${NC}"
go build -o dorcey main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}ОШИБКА: Ошибка компиляции${NC}"
    exit 1
fi

echo -e "${GREEN}Компиляция завершена успешно${NC}"

# Запуск
echo -e "${GREEN}Запускаю бота...${NC}"
echo ""
./dorcey
