package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/common"
	"github.com/adshao/go-binance/v2/futures"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	telegramBot   *tgbotapi.BotAPI
	binanceClient *futures.Client
}

func NewBot(telegramToken, binanceAPIKey, binanceSecretKey string) (*Bot, error) {
	log.Println("[DEBUG] Инициализация Telegram бота...")
	// Инициализация Telegram бота
	bot, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		log.Printf("[ERROR] Ошибка при создании Telegram бота: %v", err)
		return nil, fmt.Errorf("не удалось создать Telegram бота: %w", err)
	}
	log.Printf("[DEBUG] Telegram бот успешно создан: %s", bot.Self.UserName)

	log.Println("[DEBUG] Инициализация Binance Futures клиента...")
	// Инициализация Binance Futures клиента
	binanceClient := futures.NewClient(binanceAPIKey, binanceSecretKey)
	log.Println("[DEBUG] Binance Futures клиент успешно создан")

	return &Bot{
		telegramBot:   bot,
		binanceClient: binanceClient,
	}, nil
}

func (b *Bot) formatAPIError(err error) string {
	if apiErr, ok := err.(*common.APIError); ok {
		switch apiErr.Code {
		case -2015:
			return fmt.Sprintf("❌ Ошибка авторизации API (код %d):\n\n"+
				"Возможные причины:\n"+
				"1. Неверный API ключ или Secret Key\n"+
				"2. IP адрес не добавлен в whitelist (если включен IP whitelist)\n"+
				"3. API ключ не имеет прав на чтение Futures данных\n\n"+
				"Решение:\n"+
				"• Проверьте правильность API ключа и Secret Key\n"+
				"• Убедитесь, что IP whitelist отключен или ваш IP добавлен в whitelist\n"+
				"• В настройках API ключа включите права на 'Enable Reading' для Futures\n"+
				"• Убедитесь, что используете Futures API ключ, а не Spot API ключ\n"+
				"• Перейдите в Binance → API Management и проверьте настройки ключа\n\n"+
				"Сообщение от Binance: %s", 
				apiErr.Code, apiErr.Message)
		case -1022:
			return fmt.Sprintf("❌ Ошибка подписи (код %d):\n\nНеверный Secret Key или проблема с подписью запроса.\n\nСообщение: %s", 
				apiErr.Code, apiErr.Message)
		case -2010:
			return fmt.Sprintf("❌ Ошибка прав доступа (код %d):\n\nAPI ключ не имеет необходимых прав для выполнения операции.\n\nСообщение: %s", 
				apiErr.Code, apiErr.Message)
		default:
			return fmt.Sprintf("❌ Ошибка API Binance (код %d):\n\n%s", apiErr.Code, apiErr.Message)
		}
	}
	return fmt.Sprintf("❌ Ошибка при получении позиций: %v", err)
}


func (b *Bot) getOpenPositions() ([]*futures.PositionRisk, error) {
	log.Println("[DEBUG] Начинаю получение позиций из Binance API...")
	ctx := context.Background()
	
	// Получаем открытые позиции на futures
	positions, err := b.binanceClient.NewGetPositionRiskService().
		Do(ctx)
	
	if err != nil {
		log.Printf("[ERROR] Ошибка при запросе к Binance API: %v", err)
		return nil, err
	}

	log.Printf("[DEBUG] Получено позиций от Binance: %d", len(positions))

	// Фильтруем только открытые позиции (positionAmt != 0)
	var openPositions []*futures.PositionRisk
	for _, pos := range positions {
		// Нормализуем строку (убираем пробелы)
		positionAmtStr := strings.TrimSpace(pos.PositionAmt)
		
		// Быстрая проверка строки: если пустая или начинается с "0" (но не "0.") - пропускаем
		if positionAmtStr == "" {
			continue
		}
		
		// Убираем знак минус для проверки
		checkStr := strings.TrimPrefix(positionAmtStr, "-")
		checkStr = strings.TrimPrefix(checkStr, "+")
		
		// Проверяем, не является ли строка нулем в различных форматах
		if checkStr == "0" || checkStr == "0.0" || checkStr == "0.00" || checkStr == "0.000" || 
		   checkStr == "0.0000" || checkStr == "0.00000" || checkStr == "0.000000" ||
		   checkStr == "0.0000000" || checkStr == "0.00000000" {
			log.Printf("[DEBUG] Пропущена закрытая позиция (строка): %s, размер: %s", pos.Symbol, positionAmtStr)
			continue
		}
		
		// Парсим размер позиции как число для точной проверки
		positionAmt, err := strconv.ParseFloat(positionAmtStr, 64)
		if err != nil {
			log.Printf("[WARN] Не удалось распарсить размер позиции для %s: %s, ошибка: %v", pos.Symbol, positionAmtStr, err)
			continue
		}
		
		// Позиция считается открытой, если её абсолютное значение больше очень маленького числа (epsilon)
		// Это позволяет избежать проблем с точностью float
		const epsilon = 1e-10
		absPositionAmt := math.Abs(positionAmt)
		
		// Дополнительная проверка: цена входа должна быть больше нуля
		entryPriceStr := strings.TrimSpace(pos.EntryPrice)
		entryPrice, err2 := strconv.ParseFloat(entryPriceStr, 64)
		if err2 != nil {
			log.Printf("[DEBUG] ✗ Пропущена позиция (неверная цена входа): %s, размер: %s, цена входа: %s", 
				pos.Symbol, positionAmtStr, entryPriceStr)
			continue
		}
		
		// Позиция считается открытой только если:
		// 1. Размер позиции не равен нулю (с учетом погрешности)
		// 2. Цена входа больше нуля (позиция действительно была открыта)
		if absPositionAmt > epsilon && entryPrice > epsilon {
			openPositions = append(openPositions, pos)
			log.Printf("[DEBUG] ✓ Открытая позиция: %s, размер: %s, цена входа: %s", pos.Symbol, positionAmtStr, entryPriceStr)
		} else {
			log.Printf("[DEBUG] ✗ Пропущена закрытая позиция: %s, размер: %s (%.10f), цена входа: %s (%.10f)", 
				pos.Symbol, positionAmtStr, positionAmt, entryPriceStr, entryPrice)
		}
	}

	log.Printf("[DEBUG] ===== ИТОГО: Отфильтровано открытых позиций: %d из %d =====", len(openPositions), len(positions))
	return openPositions, nil
}

func (b *Bot) formatPositionTime(updateTime int64) string {
	now := time.Now().UnixMilli()
	duration := time.Duration(now-updateTime) * time.Millisecond
	
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	
	return fmt.Sprintf("%d ч %d мин", hours, minutes)
}

func (b *Bot) getPositionOpenTime(symbol string) (int64, error) {
	ctx := context.Background()
	
	// Получаем историю ордеров для определения времени открытия позиции
	// Используем последний выполненный ордер как приблизительное время открытия
	log.Printf("[DEBUG] Получаю время открытия позиции для %s...", symbol)
	orders, err := b.binanceClient.NewListOrdersService().
		Symbol(symbol).
		Limit(10).
		Do(ctx)
	
	if err != nil {
		log.Printf("[WARN] Не удалось получить историю ордеров для %s: %v", symbol, err)
		return time.Now().UnixMilli(), nil
	}
	
	if len(orders) == 0 {
		log.Printf("[DEBUG] Нет истории ордеров для %s, использую текущее время", symbol)
		return time.Now().UnixMilli(), nil
	}
	
	// Находим последний выполненный ордер (FILLED)
	var lastFilledTime int64 = 0
	for _, order := range orders {
		if order.Status == futures.OrderStatusTypeFilled && order.UpdateTime > lastFilledTime {
			lastFilledTime = order.UpdateTime
		}
	}
	
	if lastFilledTime > 0 {
		log.Printf("[DEBUG] Найдено время открытия для %s: %d", symbol, lastFilledTime)
		return lastFilledTime, nil
	}
	
	// Если нет выполненных ордеров, используем время последнего обновления
	if len(orders) > 0 {
		log.Printf("[DEBUG] Использую время последнего обновления для %s: %d", symbol, orders[0].UpdateTime)
		return orders[0].UpdateTime, nil
	}
	
	return time.Now().UnixMilli(), nil
}

func (b *Bot) formatPositionsMessage(positions []*futures.PositionRisk) string {
	log.Printf("[DEBUG] Форматирую сообщение для %d позиций", len(positions))
	if len(positions) == 0 {
		return "У вас нет открытых позиций на futures."
	}

	message := "📊 Открытые позиции на Futures:\n\n"
	
	for i, pos := range positions {
		log.Printf("[DEBUG] Обрабатываю позицию %d/%d: %s", i+1, len(positions), pos.Symbol)
		// Получаем время открытия позиции
		openTime, _ := b.getPositionOpenTime(pos.Symbol)
		timeStr := b.formatPositionTime(openTime)
		
		side := "LONG"
		if len(pos.PositionAmt) > 0 && pos.PositionAmt[0] == '-' {
			side = "SHORT"
		}
		
		message += fmt.Sprintf("%d. %s %s\n", i+1, pos.Symbol, side)
		message += fmt.Sprintf("   Размер: %s\n", pos.PositionAmt)
		message += fmt.Sprintf("   Цена входа: %s\n", pos.EntryPrice)
		
		// Проверяем маржу (может быть IsolatedMargin или Notional)
		margin := pos.IsolatedMargin
		if margin == "" || margin == "0" || margin == "0.0" {
			margin = pos.Notional
		}
		if margin != "" && margin != "0" && margin != "0.0" {
			message += fmt.Sprintf("   Маржа: %s\n", margin)
		}
		
		// Отображаем PnL только если он не равен нулю
		if pos.UnRealizedProfit != "" && pos.UnRealizedProfit != "0" && pos.UnRealizedProfit != "0.0" {
			message += fmt.Sprintf("   PnL: %s\n", pos.UnRealizedProfit)
		} else {
			message += fmt.Sprintf("   PnL: 0.00\n")
		}
		
		message += fmt.Sprintf("   Время сделки: %s назад\n\n", timeStr)
	}

	log.Printf("[DEBUG] Сообщение сформировано, длина: %d символов", len(message))
	return message
}

// sendLongMessage разбивает длинное сообщение на части и отправляет их по отдельности
// Telegram имеет лимит 4096 символов на сообщение
func (b *Bot) sendLongMessage(chatID int64, message string, parseMode string) error {
	const maxMessageLength = 4096
	const headerLength = 50 // Резерв для заголовка "Часть X из Y"
	const safeLength = maxMessageLength - headerLength
	
	if len(message) <= maxMessageLength {
		// Сообщение короткое, отправляем как есть
		msg := tgbotapi.NewMessage(chatID, message)
		if parseMode != "" {
			msg.ParseMode = parseMode
		}
		_, err := b.telegramBot.Send(msg)
		return err
	}
	
	// Разбиваем сообщение на строки
	lines := []string{}
	currentLine := ""
	for _, char := range message {
		if char == '\n' {
			lines = append(lines, currentLine+"\n")
			currentLine = ""
		} else {
			currentLine += string(char)
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	// Группируем строки в части
	parts := []string{}
	currentPart := ""
	
	for _, line := range lines {
		// Если одна строка слишком длинная, разбиваем её
		if len(line) > safeLength {
			// Сохраняем текущую часть, если она не пустая
			if currentPart != "" {
				parts = append(parts, currentPart)
				currentPart = ""
			}
			// Разбиваем длинную строку
			for len(line) > safeLength {
				parts = append(parts, line[:safeLength])
				line = line[safeLength:]
			}
			if line != "" {
				currentPart = line
			}
		} else if len(currentPart)+len(line) <= safeLength {
			// Строка помещается в текущую часть
			currentPart += line
		} else {
			// Строка не помещается, сохраняем текущую часть и начинаем новую
			if currentPart != "" {
				parts = append(parts, currentPart)
			}
			currentPart = line
		}
	}
	
	// Добавляем последнюю часть
	if currentPart != "" {
		parts = append(parts, currentPart)
	}
	
	// Отправляем все части
	log.Printf("[DEBUG] Сообщение разбито на %d частей", len(parts))
	for i, part := range parts {
		msg := tgbotapi.NewMessage(chatID, part)
		if parseMode != "" {
			msg.ParseMode = parseMode
		}
		// Добавляем номер части, если сообщение разбито на несколько
		if len(parts) > 1 {
			header := fmt.Sprintf("📄 Часть %d из %d\n\n", i+1, len(parts))
			msg.Text = header + part
		}
		
		sentMsg, err := b.telegramBot.Send(msg)
		if err != nil {
			log.Printf("[ERROR] Ошибка при отправке части %d из %d: %v", i+1, len(parts), err)
			return err
		}
		log.Printf("[DEBUG] Часть %d из %d отправлена успешно (message ID: %d)", i+1, len(parts), sentMsg.MessageID)
		
		// Небольшая задержка между отправками, чтобы не превысить лимиты API
		if i < len(parts)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	return nil
}

// showTyping показывает пользователю, что бот печатает сообщение
func (b *Bot) showTyping(chatID int64) {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, err := b.telegramBot.Send(action)
	if err != nil {
		log.Printf("[WARN] Не удалось отправить действие 'печатает': %v", err)
	}
}

func (b *Bot) handlePositionsCommand(update tgbotapi.Update) {
	log.Printf("[INFO] Получена команда /positions от пользователя %d (chat ID: %d)", 
		update.Message.From.ID, update.Message.Chat.ID)
	
	// Показываем, что бот печатает
	b.showTyping(update.Message.Chat.ID)
	
	// Запускаем горутину для периодического обновления индикатора печати
	// (в Telegram индикатор показывается только ~5 секунд)
	stopTyping := make(chan bool)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				b.showTyping(update.Message.Chat.ID)
			case <-stopTyping:
				return
			}
		}
	}()
	
	positions, err := b.getOpenPositions()
	if err != nil {
		stopTyping <- true
		log.Printf("[ERROR] Ошибка при получении позиций: %v", err)
		errorMsg := b.formatAPIError(err)
		log.Printf("[DEBUG] Отправляю сообщение об ошибке пользователю")
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, errorMsg)
		sentMsg, sendErr := b.telegramBot.Send(msg)
		if sendErr != nil {
			log.Printf("[ERROR] Ошибка при отправке сообщения об ошибке: %v", sendErr)
		} else {
			log.Printf("[DEBUG] Сообщение об ошибке отправлено успешно (message ID: %d)", sentMsg.MessageID)
		}
		return
	}

	log.Printf("[DEBUG] Успешно получены позиции, начинаю форматирование сообщения")
	message := b.formatPositionsMessage(positions)
	
	// Останавливаем индикатор печати перед отправкой сообщения
	stopTyping <- true
	
	log.Printf("[DEBUG] Отправляю сообщение с позициями пользователю (длина: %d символов)", len(message))
	sendErr := b.sendLongMessage(update.Message.Chat.ID, message, "HTML")
	if sendErr != nil {
		log.Printf("[ERROR] Ошибка при отправке сообщения с позициями: %v", sendErr)
	} else {
		log.Printf("[INFO] Сообщение с позициями отправлено успешно")
	}
}

func (b *Bot) Start() {
	log.Printf("[INFO] Бот запущен. Авторизован как %s (ID: %d)", 
		b.telegramBot.Self.UserName, b.telegramBot.Self.ID)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	log.Println("[INFO] Начинаю получение обновлений от Telegram...")
	updates := b.telegramBot.GetUpdatesChan(u)

	for update := range updates {
		log.Printf("[DEBUG] Получено обновление: UpdateID=%d", update.UpdateID)
		
		if update.Message == nil {
			log.Printf("[DEBUG] Обновление не содержит сообщения, пропускаю")
			continue
		}

		log.Printf("[DEBUG] Получено сообщение от пользователя %s (ID: %d) в чате %d: %s",
			update.Message.From.UserName, update.Message.From.ID, update.Message.Chat.ID, update.Message.Text)

		// Обработка команды /positions
		if update.Message.IsCommand() {
			command := update.Message.Command()
			log.Printf("[INFO] Распознана команда: /%s", command)
			
			switch command {
			case "start":
				log.Printf("[DEBUG] Обрабатываю команду /start")
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, 
					"Привет! Я бот для отслеживания открытых позиций на Binance Futures.\n\n"+
					"Используйте команду /positions для просмотра открытых позиций.")
				sentMsg, err := b.telegramBot.Send(msg)
				if err != nil {
					log.Printf("[ERROR] Ошибка при отправке ответа на /start: %v", err)
				} else {
					log.Printf("[DEBUG] Ответ на /start отправлен (message ID: %d)", sentMsg.MessageID)
				}
			case "positions":
				log.Printf("[DEBUG] Обрабатываю команду /positions")
				b.handlePositionsCommand(update)
			default:
				log.Printf("[DEBUG] Неизвестная команда: /%s", command)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, 
					"Неизвестная команда. Используйте /positions для просмотра позиций.")
				sentMsg, err := b.telegramBot.Send(msg)
				if err != nil {
					log.Printf("[ERROR] Ошибка при отправке ответа на неизвестную команду: %v", err)
				} else {
					log.Printf("[DEBUG] Ответ на неизвестную команду отправлен (message ID: %d)", sentMsg.MessageID)
				}
			}
		} else {
			log.Printf("[DEBUG] Сообщение не является командой, пропускаю")
		}
	}
}

func main() {
	log.Println("[INFO] Запуск бота...")
	
	// Получаем переменные окружения
	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	binanceAPIKey := os.Getenv("BINANCE_API_KEY")
	binanceSecretKey := os.Getenv("BINANCE_SECRET_KEY")

	if telegramToken == "" {
		log.Fatal("[FATAL] TELEGRAM_BOT_TOKEN не установлен")
	}
	log.Println("[DEBUG] TELEGRAM_BOT_TOKEN установлен")
	
	if binanceAPIKey == "" {
		log.Fatal("[FATAL] BINANCE_API_KEY не установлен")
	}
	log.Printf("[DEBUG] BINANCE_API_KEY установлен (первые 10 символов: %s...)", 
		binanceAPIKey[:min(10, len(binanceAPIKey))])
	
	if binanceSecretKey == "" {
		log.Fatal("[FATAL] BINANCE_SECRET_KEY не установлен")
	}
	log.Println("[DEBUG] BINANCE_SECRET_KEY установлен")

	log.Println("[INFO] Инициализация бота...")
	bot, err := NewBot(telegramToken, binanceAPIKey, binanceSecretKey)
	if err != nil {
		log.Fatalf("[FATAL] Ошибка создания бота: %v", err)
	}
	log.Println("[INFO] Бот успешно инициализирован")

	log.Println("[INFO] Запуск основного цикла бота...")
	bot.Start()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
