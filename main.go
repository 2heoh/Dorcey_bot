package main

import (
	"context"
	"encoding/json"
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

type Limit struct {
	Coin string `json:"coin"`
	Time string `json:"time"`
}

type LimitsStorage struct {
	Limits        []Limit `json:"limits"`
	CheckInterval string  `json:"check_interval,omitempty"` // Интервал проверки в формате "5m", "10m" и т.д.
}

type Bot struct {
	telegramBot   *tgbotapi.BotAPI
	binanceClient *futures.Client
	limitsFile    string
	chatID        int64     // ID чата для отправки уведомлений
	stopChecker   chan bool // Канал для остановки проверки
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
		limitsFile:    "limits.json",
		chatID:        0, // Будет установлен при первом сообщении
		stopChecker:   make(chan bool),
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

// calculatePositionOpenTime вычисляет время открытия текущей позиции по списку ордеров
// Отслеживает баланс позиции и находит момент последнего открытия
// (когда позиция перешла из 0 или противоположного направления в текущее)
// isLong: true для LONG позиции, false для SHORT
func calculatePositionOpenTime(orders []*futures.Order, isLong bool) int64 {
	if len(orders) == 0 {
		return 0
	}

	// Сортируем ордера по времени (от старых к новым)
	sortedOrders := make([]*futures.Order, len(orders))
	copy(sortedOrders, orders)
	for i := 0; i < len(sortedOrders)-1; i++ {
		for j := i + 1; j < len(sortedOrders); j++ {
			timeI := sortedOrders[i].Time
			if timeI == 0 {
				timeI = sortedOrders[i].UpdateTime
			}
			timeJ := sortedOrders[j].Time
			if timeJ == 0 {
				timeJ = sortedOrders[j].UpdateTime
			}
			if timeI > timeJ {
				sortedOrders[i], sortedOrders[j] = sortedOrders[j], sortedOrders[i]
			}
		}
	}

	// Отслеживаем баланс позиции
	// Положительный баланс = LONG, отрицательный = SHORT
	var positionBalance float64 = 0
	var lastOpenTime int64 = 0

	for _, order := range sortedOrders {
		if order.Status != futures.OrderStatusTypeFilled {
			continue
		}

		// Парсим количество исполненного ордера
		executedQty, err := strconv.ParseFloat(order.ExecutedQuantity, 64)
		if err != nil {
			continue
		}

		orderTime := order.Time
		if orderTime == 0 {
			orderTime = order.UpdateTime
		}

		prevBalance := positionBalance

		// BUY увеличивает позицию, SELL уменьшает
		if order.Side == futures.SideTypeBuy {
			positionBalance += executedQty
		} else {
			positionBalance -= executedQty
		}

		// Определяем направление позиции до и после ордера
		wasLong := prevBalance > 0.0000001 // Небольшой порог для float сравнения
		wasShort := prevBalance < -0.0000001
		wasZero := !wasLong && !wasShort

		nowLong := positionBalance > 0.0000001
		nowShort := positionBalance < -0.0000001

		// Позиция открылась, если:
		// 1. Была нулевой и стала ненулевой в нужном направлении
		// 2. Была в противоположном направлении и стала в нужном
		positionOpened := false
		if isLong {
			positionOpened = nowLong && (wasZero || wasShort)
		} else {
			positionOpened = nowShort && (wasZero || wasLong)
		}

		if positionOpened {
			lastOpenTime = orderTime
		}
	}

	if lastOpenTime > 0 {
		return lastOpenTime
	}

	// Если не нашли момент открытия, используем время самого старого ордера
	if len(sortedOrders) > 0 {
		oldestTime := sortedOrders[0].Time
		if oldestTime == 0 {
			oldestTime = sortedOrders[0].UpdateTime
		}
		return oldestTime
	}

	return 0
}

// getPositionOpenTime получает время открытия текущей позиции
// isLong: true для LONG позиции, false для SHORT
func (b *Bot) getPositionOpenTime(symbol string, isLong bool) (int64, error) {
	ctx := context.Background()

	log.Printf("[DEBUG] Получаю время открытия позиции для %s (направление: %v)...", symbol, isLong)
	orders, err := b.binanceClient.NewListOrdersService().
		Symbol(symbol).
		Limit(1000).
		Do(ctx)

	if err != nil {
		log.Printf("[WARN] Не удалось получить историю ордеров для %s: %v", symbol, err)
		return time.Now().UnixMilli(), nil
	}

	if len(orders) == 0 {
		log.Printf("[DEBUG] Нет истории ордеров для %s, использую текущее время", symbol)
		return time.Now().UnixMilli(), nil
	}

	openTime := calculatePositionOpenTime(orders, isLong)
	if openTime == 0 {
		return time.Now().UnixMilli(), nil
	}

	log.Printf("[DEBUG] Найдено время открытия для %s: %d", symbol, openTime)
	return openTime, nil
}

// calculateFilledOrdersCount подсчитывает исполненные ордера после времени открытия позиции
func calculateFilledOrdersCount(orders []*futures.Order, positionOpenTime int64) int {
	filledCount := 0
	for _, order := range orders {
		if order.Status == futures.OrderStatusTypeFilled {
			orderTime := order.Time
			if orderTime == 0 {
				orderTime = order.UpdateTime
			}
			if orderTime >= positionOpenTime {
				filledCount++
			}
		}
	}
	return filledCount
}

// getFilledOrdersCount получает количество исполненных ордеров для символа
// учитывая только ордера, открытые после времени открытия позиции
func (b *Bot) getFilledOrdersCount(symbol string, positionOpenTime int64) (int, error) {
	ctx := context.Background()

	log.Printf("[DEBUG] Получаю количество исполненных ордеров для %s (после времени открытия: %d)...", symbol, positionOpenTime)

	// Получаем все ордера (максимум 1000 для Binance Futures API)
	orders, err := b.binanceClient.NewListOrdersService().
		Symbol(symbol).
		Limit(1000). // Максимальный лимит для Binance Futures API
		Do(ctx)

	if err != nil {
		log.Printf("[WARN] Не удалось получить ордера для %s: %v", symbol, err)
		return 0, err
	}

	filledCount := calculateFilledOrdersCount(orders, positionOpenTime)
	log.Printf("[DEBUG] Найдено исполненных ордеров для %s (после открытия позиции): %d из %d", symbol, filledCount, len(orders))
	return filledCount, nil
}

func (b *Bot) formatPositionsMessage(positions []*futures.PositionRisk) string {
	log.Printf("[DEBUG] Форматирую сообщение для %d позиций", len(positions))
	if len(positions) == 0 {
		return "У вас нет открытых позиций на futures."
	}

	// Загружаем лимиты для проверки превышения
	storage, err := b.loadLimits()
	if err != nil {
		log.Printf("[WARN] Не удалось загрузить лимиты: %v", err)
		storage = &LimitsStorage{Limits: make([]Limit, 0)}
	}

	// Создаем карту лимитов для быстрого поиска
	limitsMap := make(map[string]time.Duration)
	limitsStrMap := make(map[string]string)
	for _, limit := range storage.Limits {
		duration, err := parseTime(limit.Time)
		if err != nil {
			log.Printf("[WARN] Не удалось распарсить лимит для %s: %v", limit.Coin, err)
			continue
		}
		coinUpper := strings.ToUpper(limit.Coin)
		limitsMap[coinUpper] = duration
		limitsStrMap[coinUpper] = limit.Time
	}

	message := "📊 Открытые позиции на Futures:\n\n"

	for i, pos := range positions {
		log.Printf("[DEBUG] Обрабатываю позицию %d/%d: %s", i+1, len(positions), pos.Symbol)
		// Определяем направление позиции
		isLong := true
		if len(pos.PositionAmt) > 0 && pos.PositionAmt[0] == '-' {
			isLong = false
		}
		// Получаем время открытия позиции
		openTime, _ := b.getPositionOpenTime(pos.Symbol, isLong)
		timeStr := b.formatPositionTime(openTime)

		// Получаем количество исполненных ордеров (только после времени открытия позиции)
		filledOrdersCount, err := b.getFilledOrdersCount(pos.Symbol, openTime)
		if err != nil {
			log.Printf("[WARN] Не удалось получить количество исполненных ордеров для %s: %v", pos.Symbol, err)
			filledOrdersCount = 0
		}

		side := "LONG"
		if len(pos.PositionAmt) > 0 && pos.PositionAmt[0] == '-' {
			side = "SHORT"
		}

		message += fmt.Sprintf("%d. %s %s\n", i+1, pos.Symbol, side)
		message += fmt.Sprintf("   Размер: %s\n", pos.PositionAmt)
		message += fmt.Sprintf("   Цена входа: %s\n", pos.EntryPrice)

		// Отображаем PnL только если он не равен нулю
		if pos.UnRealizedProfit != "" && pos.UnRealizedProfit != "0" && pos.UnRealizedProfit != "0.0" {
			message += fmt.Sprintf("   PnL: %s\n", pos.UnRealizedProfit)
		} else {
			message += fmt.Sprintf("   PnL: 0.00\n")
		}

		message += fmt.Sprintf("   Исполненных ордеров: %d\n", filledOrdersCount)
		message += fmt.Sprintf("   Время сделки: %s назад\n", timeStr)

		// Проверяем превышение лимита
		symbol := pos.Symbol
		coin := symbol
		commonSuffixes := []string{"USDT", "BUSD", "USDC", "BTC", "ETH", "BNB"}
		for _, suffix := range commonSuffixes {
			if strings.HasSuffix(symbol, suffix) {
				coin = strings.TrimSuffix(symbol, suffix)
				break
			}
		}
		coinUpper := strings.ToUpper(coin)

		if limitDuration, exists := limitsMap[coinUpper]; exists {
			now := time.Now().UnixMilli()
			positionAge := time.Duration(now-openTime) * time.Millisecond

			if positionAge > limitDuration {
				exceeded := positionAge - limitDuration
				exceededHours := int(exceeded.Hours())
				exceededMinutes := int(exceeded.Minutes()) % 60
				message += fmt.Sprintf("   ⚠️ Лимит %s превышен на %d ч %d мин\n", limitsStrMap[coinUpper], exceededHours, exceededMinutes)
			} else {
				remaining := limitDuration - positionAge
				remainingHours := int(remaining.Hours())
				remainingMinutes := int(remaining.Minutes()) % 60
				message += fmt.Sprintf("   ⏱ Лимит %s: осталось %d ч %d мин\n", limitsStrMap[coinUpper], remainingHours, remainingMinutes)
			}
		}

		message += "\n"
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
	// Для ChatAction Telegram API возвращает true (boolean), а не Message
	// Игнорируем ошибку парсинга, так как ChatAction успешно отправляется
	_, err := b.telegramBot.Send(action)
	if err != nil {
		// Проверяем, не связана ли ошибка с парсингом bool в Message
		// Если да, то игнорируем её, так как ChatAction успешно отправлен
		errStr := err.Error()
		if strings.Contains(errStr, "cannot unmarshal bool") || strings.Contains(errStr, "unmarshal") {
			// Ошибка парсинга, но ChatAction успешно отправлен - игнорируем
			return
		}
		log.Printf("[WARN] Не удалось отправить действие 'печатает': %v", err)
	}
}

// loadLimits загружает лимиты из JSON файла
func (b *Bot) loadLimits() (*LimitsStorage, error) {
	storage := &LimitsStorage{
		Limits:        make([]Limit, 0),
		CheckInterval: "5m", // Значение по умолчанию
	}

	// Проверяем, существует ли файл
	if _, err := os.Stat(b.limitsFile); os.IsNotExist(err) {
		log.Printf("[DEBUG] Файл лимитов не существует, создаю новый")
		return storage, nil
	}

	// Читаем файл
	data, err := os.ReadFile(b.limitsFile)
	if err != nil {
		log.Printf("[WARN] Ошибка при чтении файла лимитов: %v", err)
		return storage, nil
	}

	// Парсим JSON
	if len(data) == 0 {
		log.Printf("[DEBUG] Файл лимитов пуст")
		return storage, nil
	}

	if err := json.Unmarshal(data, storage); err != nil {
		log.Printf("[WARN] Ошибка при парсинге JSON лимитов: %v", err)
		return storage, nil
	}

	log.Printf("[DEBUG] Загружено лимитов: %d", len(storage.Limits))
	if storage.CheckInterval == "" {
		storage.CheckInterval = "5m" // Значение по умолчанию, если не указано
	}
	log.Printf("[DEBUG] Интервал проверки: %s", storage.CheckInterval)
	return storage, nil
}

// saveLimits сохраняет лимиты в JSON файл
func (b *Bot) saveLimits(storage *LimitsStorage) error {
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при сериализации лимитов: %w", err)
	}

	if err := os.WriteFile(b.limitsFile, data, 0644); err != nil {
		return fmt.Errorf("ошибка при записи файла лимитов: %w", err)
	}

	log.Printf("[DEBUG] Сохранено лимитов: %d", len(storage.Limits))
	return nil
}

// parseTime парсит строку времени в формате "12h", "30m", "1d" и т.д.
func parseTime(timeStr string) (time.Duration, error) {
	timeStr = strings.TrimSpace(timeStr)
	if len(timeStr) == 0 {
		return 0, fmt.Errorf("пустая строка времени")
	}

	// Определяем единицу измерения (последний символ)
	unit := timeStr[len(timeStr)-1:]
	valueStr := timeStr[:len(timeStr)-1]

	// Парсим числовое значение
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("неверный формат числа: %s", valueStr)
	}

	// Преобразуем в Duration в зависимости от единицы
	var duration time.Duration
	switch strings.ToLower(unit) {
	case "s", "S":
		duration = time.Duration(value) * time.Second
	case "m", "M":
		duration = time.Duration(value) * time.Minute
	case "h", "H":
		duration = time.Duration(value) * time.Hour
	case "d", "D":
		duration = time.Duration(value) * 24 * time.Hour
	default:
		return 0, fmt.Errorf("неизвестная единица времени: %s (используйте s, m, h или d)", unit)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("время должно быть больше нуля")
	}

	return duration, nil
}

// handleAddLimitCommand обрабатывает команду /add_limit
func (b *Bot) handleAddLimitCommand(update tgbotapi.Update) {
	log.Printf("[INFO] Получена команда /add_limit от пользователя %d (chat ID: %d)",
		update.Message.From.ID, update.Message.Chat.ID)

	// Получаем аргументы команды
	args := update.Message.CommandArguments()
	parts := strings.Fields(args)

	if len(parts) < 2 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Неверный формат команды.\n\n"+
				"Использование: /add_limit (или /l) <coin> <time>\n\n"+
				"Примеры:\n"+
				"/l LSK 12h\n"+
				"/l BTC 30m\n"+
				"/l ETH 1d\n\n"+
				"Единицы времени: s (секунды), m (минуты), h (часы), d (дни)")
		b.telegramBot.Send(msg)
		return
	}

	coin := strings.ToUpper(strings.TrimSpace(parts[0]))
	timeStr := strings.TrimSpace(parts[1])

	// Парсим время
	duration, err := parseTime(timeStr)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("❌ Ошибка при парсинге времени: %s\n\n"+
				"Используйте формат: число + единица (s, m, h, d)\n"+
				"Примеры: 12h, 30m, 1d", err.Error()))
		b.telegramBot.Send(msg)
		return
	}

	// Загружаем существующие лимиты
	storage, err := b.loadLimits()
	if err != nil {
		log.Printf("[ERROR] Ошибка при загрузке лимитов: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Ошибка при загрузке лимитов. Попробуйте позже.")
		b.telegramBot.Send(msg)
		return
	}

	// Проверяем, существует ли уже лимит для этой монеты
	for i, limit := range storage.Limits {
		if strings.ToUpper(limit.Coin) == coin {
			// Обновляем существующий лимит
			storage.Limits[i].Time = timeStr
			log.Printf("[DEBUG] Обновлен лимит для %s: %s", coin, timeStr)

			if err := b.saveLimits(storage); err != nil {
				log.Printf("[ERROR] Ошибка при сохранении лимитов: %v", err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"❌ Ошибка при сохранении лимитов. Попробуйте позже.")
				b.telegramBot.Send(msg)
				return
			}

			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				fmt.Sprintf("✅ Лимит для %s обновлен: %s (%.0f минут)",
					coin, timeStr, duration.Minutes()))
			b.telegramBot.Send(msg)
			return
		}
	}

	// Добавляем новый лимит
	newLimit := Limit{
		Coin: coin,
		Time: timeStr,
	}
	storage.Limits = append(storage.Limits, newLimit)

	// Сохраняем лимиты
	if err := b.saveLimits(storage); err != nil {
		log.Printf("[ERROR] Ошибка при сохранении лимитов: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Ошибка при сохранении лимитов. Попробуйте позже.")
		b.telegramBot.Send(msg)
		return
	}

	log.Printf("[INFO] Добавлен новый лимит: %s - %s", coin, timeStr)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Лимит добавлен:\n\n"+
			"Монета: %s\n"+
			"Время: %s (%.0f минут)",
			coin, timeStr, duration.Minutes()))
	b.telegramBot.Send(msg)
}

// handleLimitsCommand обрабатывает команду /limits
func (b *Bot) handleLimitsCommand(update tgbotapi.Update) {
	log.Printf("[INFO] Получена команда /limits от пользователя %d (chat ID: %d)",
		update.Message.From.ID, update.Message.Chat.ID)

	// Загружаем лимиты
	storage, err := b.loadLimits()
	if err != nil {
		log.Printf("[ERROR] Ошибка при загрузке лимитов: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Ошибка при загрузке лимитов. Попробуйте позже.")
		b.telegramBot.Send(msg)
		return
	}

	// Проверяем, есть ли лимиты
	if len(storage.Limits) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"📋 Установленных лимитов нет.\n\n"+
				"Используйте команду /add_limit для добавления лимитов.\n\n"+
				"Пример: /add_limit LSK 12h")
		b.telegramBot.Send(msg)
		return
	}

	// Формируем сообщение со списком лимитов
	message := "📋 Установленные лимиты:\n\n"

	for i, limit := range storage.Limits {
		// Парсим время для отображения в минутах
		duration, err := parseTime(limit.Time)
		var timeDisplay string
		if err != nil {
			timeDisplay = limit.Time
		} else {
			minutes := duration.Minutes()
			if minutes < 60 {
				timeDisplay = fmt.Sprintf("%s (%.0f мин)", limit.Time, minutes)
			} else if minutes < 1440 {
				hours := minutes / 60
				timeDisplay = fmt.Sprintf("%s (%.1f ч)", limit.Time, hours)
			} else {
				days := minutes / 1440
				timeDisplay = fmt.Sprintf("%s (%.1f дн)", limit.Time, days)
			}
		}

		message += fmt.Sprintf("%d. %s - %s\n", i+1, limit.Coin, timeDisplay)
	}

	message += "\n💡 Используйте /add_limit для добавления или изменения лимитов."

	// Добавляем информацию об интервале проверки
	checkInterval := storage.CheckInterval
	if checkInterval == "" {
		checkInterval = "5m (по умолчанию)"
	}
	message += fmt.Sprintf("\n\n⏱ Интервал проверки позиций: %s", checkInterval)
	message += "\n💡 Используйте /set_check_interval для изменения интервала."

	// Отправляем сообщение
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, message)
	b.telegramBot.Send(msg)
}

// handleSetCheckIntervalCommand обрабатывает команду /set_check_interval
func (b *Bot) handleSetCheckIntervalCommand(update tgbotapi.Update) {
	log.Printf("[INFO] Получена команда /set_check_interval от пользователя %d (chat ID: %d)",
		update.Message.From.ID, update.Message.Chat.ID)

	// Получаем аргументы команды
	args := update.Message.CommandArguments()
	args = strings.TrimSpace(args)

	if args == "" {
		// Показываем текущий интервал
		storage, err := b.loadLimits()
		if err != nil {
			log.Printf("[ERROR] Ошибка при загрузке настроек: %v", err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"❌ Ошибка при загрузке настроек. Попробуйте позже.")
			b.telegramBot.Send(msg)
			return
		}

		checkInterval := storage.CheckInterval
		if checkInterval == "" {
			checkInterval = "5m (по умолчанию)"
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("⏱ Текущий интервал проверки: %s\n\n"+
				"Использование: /set_check_interval <interval>\n\n"+
				"Примеры:\n"+
				"/set_check_interval 5m\n"+
				"/set_check_interval 10m\n"+
				"/set_check_interval 1h\n\n"+
				"Единицы времени: s (секунды), m (минуты), h (часы), d (дни)",
				checkInterval))
		b.telegramBot.Send(msg)
		return
	}

	// Парсим интервал
	intervalDuration, err := parseTime(args)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("❌ Ошибка при парсинге интервала: %s\n\n"+
				"Используйте формат: число + единица (s, m, h, d)\n"+
				"Примеры: 5m, 10m, 1h", err.Error()))
		b.telegramBot.Send(msg)
		return
	}

	// Загружаем существующие настройки
	storage, err := b.loadLimits()
	if err != nil {
		log.Printf("[ERROR] Ошибка при загрузке настроек: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Ошибка при загрузке настроек. Попробуйте позже.")
		b.telegramBot.Send(msg)
		return
	}

	// Обновляем интервал проверки
	storage.CheckInterval = args

	// Сохраняем настройки
	if err := b.saveLimits(storage); err != nil {
		log.Printf("[ERROR] Ошибка при сохранении настроек: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"❌ Ошибка при сохранении настроек. Попробуйте позже.")
		b.telegramBot.Send(msg)
		return
	}

	log.Printf("[INFO] Интервал проверки обновлен: %s", args)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("✅ Интервал проверки обновлен: %s (%.0f минут)\n\n"+
			"⚠️ Для применения изменений перезапустите бота.",
			args, intervalDuration.Minutes()))
	b.telegramBot.Send(msg)
}

// checkPositionsForLimits проверяет открытые позиции на превышение лимитов
func (b *Bot) checkPositionsForLimits() {
	if b.chatID == 0 {
		log.Printf("[DEBUG] ChatID не установлен, пропускаю проверку позиций")
		return
	}

	log.Printf("[DEBUG] Начинаю проверку позиций на превышение лимитов...")

	// Загружаем лимиты
	storage, err := b.loadLimits()
	if err != nil {
		log.Printf("[ERROR] Ошибка при загрузке лимитов для проверки: %v", err)
		return
	}

	// Если нет лимитов, нечего проверять
	if len(storage.Limits) == 0 {
		log.Printf("[DEBUG] Нет установленных лимитов, пропускаю проверку")
		return
	}

	// Получаем открытые позиции
	positions, err := b.getOpenPositions()
	if err != nil {
		log.Printf("[ERROR] Ошибка при получении позиций для проверки: %v", err)
		return
	}

	if len(positions) == 0 {
		log.Printf("[DEBUG] Нет открытых позиций для проверки")
		return
	}

	// Создаем карту лимитов для быстрого поиска
	limitsMap := make(map[string]time.Duration)
	for _, limit := range storage.Limits {
		duration, err := parseTime(limit.Time)
		if err != nil {
			log.Printf("[WARN] Не удалось распарсить лимит для %s: %v", limit.Coin, err)
			continue
		}
		limitsMap[strings.ToUpper(limit.Coin)] = duration
	}

	// Проверяем каждую позицию
	var exceededPositions []*futures.PositionRisk
	for _, pos := range positions {
		// Извлекаем базовую монету из символа (например, BTCUSDT -> BTC)
		symbol := pos.Symbol
		coin := symbol

		// Пытаемся определить базовую монету
		// Обычно это первая часть до USDT, BUSD и т.д.
		commonSuffixes := []string{"USDT", "BUSD", "USDC", "BTC", "ETH", "BNB"}
		for _, suffix := range commonSuffixes {
			if strings.HasSuffix(symbol, suffix) {
				coin = strings.TrimSuffix(symbol, suffix)
				break
			}
		}

		coinUpper := strings.ToUpper(coin)
		limitDuration, exists := limitsMap[coinUpper]

		if !exists {
			log.Printf("[DEBUG] Лимит для %s (%s) не найден, пропускаю", symbol, coinUpper)
			continue
		}

		// Определяем направление позиции
		isLong := true
		if len(pos.PositionAmt) > 0 && pos.PositionAmt[0] == '-' {
			isLong = false
		}

		// Получаем время открытия позиции
		openTime, err := b.getPositionOpenTime(symbol, isLong)
		if err != nil {
			log.Printf("[WARN] Не удалось получить время открытия для %s: %v", symbol, err)
			continue
		}

		// Вычисляем время жизни позиции
		now := time.Now().UnixMilli()
		positionAge := time.Duration(now-openTime) * time.Millisecond

		// Проверяем, превышает ли время жизни лимит
		if positionAge > limitDuration {
			log.Printf("[INFO] Позиция %s превысила лимит: возраст %v, лимит %v",
				symbol, positionAge, limitDuration)
			exceededPositions = append(exceededPositions, pos)
		}
	}

	// Отправляем уведомления о позициях, превысивших лимит
	if len(exceededPositions) > 0 {
		b.sendLimitExceededNotifications(exceededPositions, storage)
	} else {
		log.Printf("[DEBUG] Все позиции в пределах лимитов")
	}
}

// sendLimitExceededNotifications отправляет уведомления о позициях, превысивших лимит
func (b *Bot) sendLimitExceededNotifications(positions []*futures.PositionRisk, storage *LimitsStorage) {
	log.Printf("[INFO] Отправляю уведомления о %d позициях, превысивших лимит", len(positions))

	message := "⚠️ <b>ВНИМАНИЕ: Позиции превысили установленные лимиты!</b>\n\n"

	for _, pos := range positions {
		// Извлекаем базовую монету
		symbol := pos.Symbol
		coin := symbol
		commonSuffixes := []string{"USDT", "BUSD", "USDC", "BTC", "ETH", "BNB"}
		for _, suffix := range commonSuffixes {
			if strings.HasSuffix(symbol, suffix) {
				coin = strings.TrimSuffix(symbol, suffix)
				break
			}
		}
		coinUpper := strings.ToUpper(coin)

		// Находим лимит для этой монеты
		var limitDuration time.Duration
		var limitStr string
		for _, limit := range storage.Limits {
			if strings.ToUpper(limit.Coin) == coinUpper {
				limitStr = limit.Time
				var err error
				limitDuration, err = parseTime(limit.Time)
				if err != nil {
					log.Printf("[WARN] Ошибка парсинга лимита для %s: %v", coinUpper, err)
				}
				break
			}
		}

		// Определяем направление позиции
		side := "LONG"
		isLong := true
		if len(pos.PositionAmt) > 0 && pos.PositionAmt[0] == '-' {
			side = "SHORT"
			isLong = false
		}

		// Получаем время открытия и вычисляем возраст
		openTime, _ := b.getPositionOpenTime(symbol, isLong)
		now := time.Now().UnixMilli()
		positionAge := time.Duration(now-openTime) * time.Millisecond
		ageStr := b.formatPositionTime(openTime)

		message += fmt.Sprintf("🔴 <b>%s %s</b>\n", symbol, side)
		message += fmt.Sprintf("   Размер: %s\n", pos.PositionAmt)
		message += fmt.Sprintf("   Цена входа: %s\n", pos.EntryPrice)

		// Отображаем PnL
		if pos.UnRealizedProfit != "" && pos.UnRealizedProfit != "0" && pos.UnRealizedProfit != "0.0" {
			message += fmt.Sprintf("   PnL: %s\n", pos.UnRealizedProfit)
		}

		message += fmt.Sprintf("   Время жизни: %s (лимит: %s)\n", ageStr, limitStr)
		message += fmt.Sprintf("   ⚠️ Превышение: %v\n\n", positionAge-limitDuration)
	}

	message += "💡 <i>Рекомендуется закрыть позиции, превысившие лимиты.</i>"

	// Отправляем сообщение
	err := b.sendLongMessage(b.chatID, message, "HTML")
	if err != nil {
		log.Printf("[ERROR] Ошибка при отправке уведомления о превышении лимитов: %v", err)
	} else {
		log.Printf("[INFO] Уведомление о превышении лимитов отправлено успешно")
	}
}

// startPositionChecker запускает фоновую горутину для периодической проверки позиций
func (b *Bot) startPositionChecker() {
	log.Printf("[INFO] Запуск фоновой проверки позиций...")

	// Загружаем настройки для получения интервала проверки
	storage, err := b.loadLimits()
	if err != nil {
		log.Printf("[ERROR] Ошибка при загрузке настроек для проверки: %v", err)
		return
	}

	// Парсим интервал проверки
	checkInterval := storage.CheckInterval
	if checkInterval == "" {
		checkInterval = "5m" // Значение по умолчанию
	}

	intervalDuration, err := parseTime(checkInterval)
	if err != nil {
		log.Printf("[ERROR] Ошибка при парсинге интервала проверки '%s': %v, использую 5 минут", checkInterval, err)
		intervalDuration = 5 * time.Minute
	}

	log.Printf("[INFO] Интервал проверки позиций: %v", intervalDuration)

	// Запускаем горутину
	go func() {
		ticker := time.NewTicker(intervalDuration)
		defer ticker.Stop()

		// Выполняем первую проверку сразу при запуске (опционально)
		// Можно закомментировать, если не нужно проверять сразу
		// b.checkPositionsForLimits()

		for {
			select {
			case <-ticker.C:
				b.checkPositionsForLimits()
			case <-b.stopChecker:
				log.Printf("[INFO] Остановка фоновой проверки позиций")
				return
			}
		}
	}()
}

func (b *Bot) handlePositionsCommand(update tgbotapi.Update) {
	log.Printf("[INFO] Получена команда /positions или /ps от пользователя %d (chat ID: %d)",
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

	// Запускаем фоновую проверку позиций
	b.startPositionChecker()

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

		// Сохраняем chatID при первом сообщении (если еще не установлен)
		if b.chatID == 0 {
			b.chatID = update.Message.Chat.ID
			log.Printf("[INFO] Установлен chatID для уведомлений: %d", b.chatID)
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
						"Доступные команды:\n"+
						"/positions или /ps - просмотр открытых позиций\n"+
						"/add_limit или /l - добавление лимитов\n"+
						"/limits или /ls - просмотр установленных лимитов\n"+
						"/set_check_interval - установка интервала проверки позиций")
				sentMsg, err := b.telegramBot.Send(msg)
				if err != nil {
					log.Printf("[ERROR] Ошибка при отправке ответа на /start: %v", err)
				} else {
					log.Printf("[DEBUG] Ответ на /start отправлен (message ID: %d)", sentMsg.MessageID)
				}
			case "positions", "ps":
				log.Printf("[DEBUG] Обрабатываю команду /%s", command)
				b.handlePositionsCommand(update)
			case "add_limit", "l":
				log.Printf("[DEBUG] Обрабатываю команду /%s", command)
				b.handleAddLimitCommand(update)
			case "limits", "ls":
				log.Printf("[DEBUG] Обрабатываю команду /%s", command)
				b.handleLimitsCommand(update)
			case "set_check_interval":
				log.Printf("[DEBUG] Обрабатываю команду /set_check_interval")
				b.handleSetCheckIntervalCommand(update)
			default:
				log.Printf("[DEBUG] Неизвестная команда: /%s", command)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"Неизвестная команда. Используйте:\n"+
						"/positions или /ps - для просмотра позиций\n"+
						"/add_limit или /l - для добавления лимитов\n"+
						"/limits или /ls - для просмотра установленных лимитов\n"+
						"/set_check_interval - для установки интервала проверки")
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
