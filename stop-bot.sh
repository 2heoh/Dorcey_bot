#!/bin/bash

# Скрипт для остановки всех запущенных экземпляров бота

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

PID_FILE="bot.pid"

echo -e "${YELLOW}=== Остановка Binance Futures Telegram Bot ===${NC}"

# Поиск всех процессов бота
PIDS=$(pgrep -f "dorcey|go run main.go" 2>/dev/null)

if [ -z "$PIDS" ]; then
    echo -e "${GREEN}Бот не запущен${NC}"
    
    # Удаляем PID файл, если он есть
    if [ -f "$PID_FILE" ]; then
        rm -f "$PID_FILE"
        echo -e "${YELLOW}Удален устаревший PID файл${NC}"
    fi
    exit 0
fi

echo -e "${YELLOW}Найдены запущенные процессы:${NC}"
ps -p $PIDS -o pid,cmd,etime 2>/dev/null || echo "$PIDS"

# Остановка процессов
echo ""
echo -e "${YELLOW}Останавливаю процессы...${NC}"

for PID in $PIDS; do
    if ps -p "$PID" > /dev/null 2>&1; then
        echo -e "  Останавливаю процесс $PID..."
        kill "$PID" 2>/dev/null
        
        # Ждем завершения процесса (максимум 5 секунд)
        for i in {1..5}; do
            if ! ps -p "$PID" > /dev/null 2>&1; then
                echo -e "  ${GREEN}Процесс $PID остановлен${NC}"
                break
            fi
            sleep 1
        done
        
        # Если процесс все еще работает, принудительно завершаем
        if ps -p "$PID" > /dev/null 2>&1; then
            echo -e "  ${YELLOW}Принудительное завершение процесса $PID...${NC}"
            kill -9 "$PID" 2>/dev/null
            sleep 1
            if ! ps -p "$PID" > /dev/null 2>&1; then
                echo -e "  ${GREEN}Процесс $PID принудительно завершен${NC}"
            fi
        fi
    fi
done

# Удаляем PID файл
if [ -f "$PID_FILE" ]; then
    rm -f "$PID_FILE"
    echo -e "${GREEN}Удален PID файл${NC}"
fi

# Финальная проверка
REMAINING=$(pgrep -f "dorcey|go run main.go" 2>/dev/null)
if [ -z "$REMAINING" ]; then
    echo -e "${GREEN}Все экземпляры бота успешно остановлены${NC}"
else
    echo -e "${RED}ВНИМАНИЕ: Некоторые процессы все еще запущены:${NC}"
    ps -p $REMAINING -o pid,cmd 2>/dev/null
    echo -e "${YELLOW}Попробуйте принудительно: pkill -9 -f 'dorcey|go run main.go'${NC}"
    exit 1
fi
