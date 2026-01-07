package main

import (
	"testing"

	"github.com/adshao/go-binance/v2/futures"
)

// Фикстура: симуляция истории ордеров для LSKUSDT (One-way Mode)
// Сценарий: позиция открывалась и закрывалась несколько раз
// Последнее открытие: ордер ID=1006 (BUY 261 в момент времени 1767159730815)
// После открытия: ещё один ордер ID=1007 (BUY 100)
// Ожидаемый результат: время открытия = 1767159730815, количество ордеров = 2
func createTestOrdersLSK_OneWayMode() []*futures.Order {
	return []*futures.Order{
		// Первое открытие позиции (будет закрыто)
		{
			OrderID:          1001,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "127",
			Time:             1766643367148,
			UpdateTime:       1766643367148,
		},
		// Закрытие первой позиции
		{
			OrderID:          1002,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			ExecutedQuantity: "127",
			Time:             1766659210282,
			UpdateTime:       1766659210282,
		},
		// Второе открытие позиции (будет закрыто)
		{
			OrderID:          1003,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "125",
			Time:             1766784671130,
			UpdateTime:       1766784671130,
		},
		// Закрытие второй позиции
		{
			OrderID:          1004,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			ExecutedQuantity: "125",
			Time:             1766831349301,
			UpdateTime:       1766831349301,
		},
		// Ордер, который не исполнен (должен игнорироваться)
		{
			OrderID:          1005,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeCanceled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "100",
			Time:             1767100000000,
			UpdateTime:       1767100000000,
		},
		// ТЕКУЩЕЕ открытие позиции - это время должно быть возвращено
		{
			OrderID:          1006,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "261",
			Time:             1767159730815,
			UpdateTime:       1767159730815,
		},
		// Дополнительный ордер после открытия (усреднение)
		{
			OrderID:          1007,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "100",
			Time:             1767177846634,
			UpdateTime:       1767177846634,
		},
	}
}

// Фикстура: реальные данные LSKUSDT в Hedge Mode
// Воспроизводит баг: показывалось 7 ордеров вместо 1
// Сценарий: многократные открытия/закрытия LONG позиции
// Текущая позиция: LONG 570 LSK (открыт последним BUY ордером)
// Ожидаемый результат: время открытия = 1767783430546, количество ордеров = 1
func createTestOrdersLSK_HedgeMode() []*futures.Order {
	return []*futures.Order{
		// Закрытие предыдущей LONG позиции
		{
			OrderID:          1892681196,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "590",
			Time:             1767203645727,
			UpdateTime:       1767203645727,
		},
		// Открытие LONG, затем закрытие
		{
			OrderID:          1892685332,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "637",
			Time:             1767204130387,
			UpdateTime:       1767204130387,
		},
		{
			OrderID:          1892685352,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "637",
			Time:             1767204131146,
			UpdateTime:       1767204131146,
		},
		// Ещё один цикл открытия/закрытия
		{
			OrderID:          1893604086,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "608",
			Time:             1767363611240,
			UpdateTime:       1767363611240,
		},
		{
			OrderID:          1893604111,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "608",
			Time:             1767363611985,
			UpdateTime:       1767363611985,
		},
		// Отменённый ордер (должен игнорироваться)
		{
			OrderID:          1894095326,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeCanceled,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "0",
			Time:             1767418631518,
			UpdateTime:       1767418631518,
		},
		// Ещё несколько циклов...
		{
			OrderID:          1897544487,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "579",
			Time:             1767747611750,
			UpdateTime:       1767747611750,
		},
		{
			OrderID:          1897544546,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "579",
			Time:             1767747612490,
			UpdateTime:       1767747612490,
		},
		{
			OrderID:          1897761426,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "565",
			Time:             1767765727259,
			UpdateTime:       1767765727259,
		},
		// Закрытие перед последним открытием
		{
			OrderID:          1897913042,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "565",
			Time:             1767782836658,
			UpdateTime:       1767782836658,
		},
		// ТЕКУЩЕЕ ОТКРЫТИЕ ПОЗИЦИИ - BUY 570, это единственный ордер текущей позиции
		{
			OrderID:          1897918427,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "570",
			Time:             1767783430546,
			UpdateTime:       1767783430546,
		},
		// Take-profit ордер (ещё не исполнен)
		{
			OrderID:          1897918464,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeNew,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "0",
			Time:             1767783431535,
			UpdateTime:       1767783431535,
		},
	}
}

// ============================================================================
// Тесты для One-way Mode
// ============================================================================

// TestCalculatePositionOpenTime_LongPosition проверяет определение времени открытия LONG позиции
func TestCalculatePositionOpenTime_LongPosition(t *testing.T) {
	orders := createTestOrdersLSK_OneWayMode()

	openTime := calculatePositionOpenTime(orders, true)

	expectedTime := int64(1767159730815)
	if openTime != expectedTime {
		t.Errorf("Ожидалось время открытия %d, получено %d", expectedTime, openTime)
	}
}

// TestCalculateFilledOrdersCount проверяет подсчёт исполненных ордеров после открытия позиции
func TestCalculateFilledOrdersCount(t *testing.T) {
	orders := createTestOrdersLSK_OneWayMode()
	positionOpenTime := int64(1767159730815)

	// Для LONG позиции считаем только BUY ордера
	filledCount := calculateFilledOrdersCount(orders, positionOpenTime, true)

	expectedCount := 2 // Ордера 1006 и 1007 (оба BUY)
	if filledCount != expectedCount {
		t.Errorf("Ожидалось %d ордеров, получено %d", expectedCount, filledCount)
	}
}

// TestCalculatePositionOpenTime_ShortPosition проверяет определение времени открытия SHORT позиции
func TestCalculatePositionOpenTime_ShortPosition(t *testing.T) {
	orders := []*futures.Order{
		// Открытие SHORT позиции
		{
			OrderID:          2001,
			Symbol:           "BTCUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			ExecutedQuantity: "0.5",
			Time:             1767000000000,
			UpdateTime:       1767000000000,
		},
		// Закрытие SHORT
		{
			OrderID:          2002,
			Symbol:           "BTCUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "0.5",
			Time:             1767100000000,
			UpdateTime:       1767100000000,
		},
		// Новое открытие SHORT - это время должно быть возвращено
		{
			OrderID:          2003,
			Symbol:           "BTCUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			ExecutedQuantity: "0.3",
			Time:             1767200000000,
			UpdateTime:       1767200000000,
		},
	}

	openTime := calculatePositionOpenTime(orders, false)

	expectedTime := int64(1767200000000)
	if openTime != expectedTime {
		t.Errorf("Ожидалось время открытия SHORT %d, получено %d", expectedTime, openTime)
	}
}

// TestCalculatePositionOpenTime_EmptyOrders проверяет поведение при пустом списке ордеров
func TestCalculatePositionOpenTime_EmptyOrders(t *testing.T) {
	orders := []*futures.Order{}

	openTime := calculatePositionOpenTime(orders, true)

	if openTime != 0 {
		t.Errorf("Ожидалось 0 для пустого списка, получено %d", openTime)
	}
}

// TestCalculateFilledOrdersCount_NoMatchingOrders проверяет подсчёт когда нет подходящих ордеров
func TestCalculateFilledOrdersCount_NoMatchingOrders(t *testing.T) {
	orders := []*futures.Order{
		{
			OrderID:          3001,
			Symbol:           "ETHUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "1.0",
			Time:             1767000000000,
			UpdateTime:       1767000000000,
		},
	}

	// Время открытия позже всех ордеров
	filledCount := calculateFilledOrdersCount(orders, 1768000000000, true)

	if filledCount != 0 {
		t.Errorf("Ожидалось 0 ордеров, получено %d", filledCount)
	}
}

// TestCalculatePositionOpenTime_UnsortedOrders проверяет корректную сортировку ордеров
func TestCalculatePositionOpenTime_UnsortedOrders(t *testing.T) {
	// Ордера в неправильном порядке
	orders := []*futures.Order{
		// Второй по времени
		{
			OrderID:          4002,
			Symbol:           "XRPUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			ExecutedQuantity: "100",
			Time:             1767200000000,
			UpdateTime:       1767200000000,
		},
		// Третий по времени - текущее открытие
		{
			OrderID:          4003,
			Symbol:           "XRPUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "150",
			Time:             1767300000000,
			UpdateTime:       1767300000000,
		},
		// Первый по времени
		{
			OrderID:          4001,
			Symbol:           "XRPUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "100",
			Time:             1767100000000,
			UpdateTime:       1767100000000,
		},
	}

	openTime := calculatePositionOpenTime(orders, true)

	// Должен найти последнее открытие LONG (после закрытия SELL)
	expectedTime := int64(1767300000000)
	if openTime != expectedTime {
		t.Errorf("Ожидалось время открытия %d, получено %d", expectedTime, openTime)
	}
}

// TestCalculatePositionOpenTime_FlipFromShortToLong проверяет переворот из SHORT в LONG
func TestCalculatePositionOpenTime_FlipFromShortToLong(t *testing.T) {
	orders := []*futures.Order{
		// Открытие SHORT
		{
			OrderID:          5001,
			Symbol:           "SOLUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			ExecutedQuantity: "10",
			Time:             1767100000000,
			UpdateTime:       1767100000000,
		},
		// BUY больше чем SHORT - переворот в LONG
		{
			OrderID:          5002,
			Symbol:           "SOLUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "20",
			Time:             1767200000000,
			UpdateTime:       1767200000000,
		},
	}

	openTime := calculatePositionOpenTime(orders, true)

	// Время открытия LONG - момент переворота
	expectedTime := int64(1767200000000)
	if openTime != expectedTime {
		t.Errorf("Ожидалось время переворота в LONG %d, получено %d", expectedTime, openTime)
	}
}

// ============================================================================
// Тесты для Hedge Mode
// ============================================================================

// TestHedgeMode_PositionOpenTime проверяет определение времени открытия в Hedge Mode
// Воспроизводит баг: неправильное определение времени открытия при многократных циклах
func TestHedgeMode_PositionOpenTime(t *testing.T) {
	orders := createTestOrdersLSK_HedgeMode()

	openTime := calculatePositionOpenTime(orders, true)

	// Ожидаем время последнего BUY ордера (ID: 1897918427)
	expectedTime := int64(1767783430546)
	if openTime != expectedTime {
		t.Errorf("Ожидалось время открытия %d, получено %d", expectedTime, openTime)
	}
}

// TestHedgeMode_FilledOrdersCount проверяет подсчёт ордеров в Hedge Mode
// Воспроизводит баг: показывалось 7 ордеров вместо 1
func TestHedgeMode_FilledOrdersCount(t *testing.T) {
	orders := createTestOrdersLSK_HedgeMode()

	// Сначала определяем время открытия
	openTime := calculatePositionOpenTime(orders, true)

	// Затем считаем исполненные ордера
	filledCount := calculateFilledOrdersCount(orders, openTime, true)

	// Ожидаем только 1 ордер (BUY 570, ID: 1897918427)
	expectedCount := 1
	if filledCount != expectedCount {
		t.Errorf("Ожидалось %d ордеров, получено %d (баг: должен быть 1, а не 7)", expectedCount, filledCount)
	}
}

// TestHedgeMode_FiltersOrdersByPositionSide проверяет фильтрацию по PositionSide
func TestHedgeMode_FiltersOrdersByPositionSide(t *testing.T) {
	// Создаём ордера для LONG и SHORT позиций одновременно
	orders := []*futures.Order{
		// LONG ордера
		{
			OrderID:          1001,
			Symbol:           "BTCUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "0.1",
			Time:             1767000000000,
			UpdateTime:       1767000000000,
		},
		// SHORT ордера (не должны учитываться для LONG позиции)
		{
			OrderID:          2001,
			Symbol:           "BTCUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeShort,
			ExecutedQuantity: "0.2",
			Time:             1767100000000,
			UpdateTime:       1767100000000,
		},
		// Ещё один LONG BUY (усреднение)
		{
			OrderID:          1002,
			Symbol:           "BTCUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "0.1",
			Time:             1767200000000,
			UpdateTime:       1767200000000,
		},
	}

	// Время открытия LONG
	openTime := calculatePositionOpenTime(orders, true)
	expectedOpenTime := int64(1767000000000)
	if openTime != expectedOpenTime {
		t.Errorf("Ожидалось время открытия LONG %d, получено %d", expectedOpenTime, openTime)
	}

	// Количество ордеров LONG
	filledCount := calculateFilledOrdersCount(orders, openTime, true)
	expectedCount := 2 // Только LONG BUY ордера
	if filledCount != expectedCount {
		t.Errorf("Ожидалось %d LONG ордеров, получено %d", expectedCount, filledCount)
	}

	// Проверяем SHORT позицию
	openTimeShort := calculatePositionOpenTime(orders, false)
	expectedOpenTimeShort := int64(1767100000000)
	if openTimeShort != expectedOpenTimeShort {
		t.Errorf("Ожидалось время открытия SHORT %d, получено %d", expectedOpenTimeShort, openTimeShort)
	}

	filledCountShort := calculateFilledOrdersCount(orders, openTimeShort, false)
	expectedCountShort := 1 // Только SHORT SELL ордер
	if filledCountShort != expectedCountShort {
		t.Errorf("Ожидалось %d SHORT ордеров, получено %d", expectedCountShort, filledCountShort)
	}
}

// TestHedgeMode_IgnoresCanceledOrders проверяет игнорирование отменённых ордеров
func TestHedgeMode_IgnoresCanceledOrders(t *testing.T) {
	orders := []*futures.Order{
		// Открытие LONG
		{
			OrderID:          1001,
			Symbol:           "ETHUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "1.0",
			Time:             1767000000000,
			UpdateTime:       1767000000000,
		},
		// Отменённый ордер (не должен учитываться)
		{
			OrderID:          1002,
			Symbol:           "ETHUSDT",
			Status:           futures.OrderStatusTypeCanceled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "0",
			Time:             1767100000000,
			UpdateTime:       1767100000000,
		},
		// Ещё один отменённый
		{
			OrderID:          1003,
			Symbol:           "ETHUSDT",
			Status:           futures.OrderStatusTypeExpired,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "0",
			Time:             1767200000000,
			UpdateTime:       1767200000000,
		},
	}

	openTime := calculatePositionOpenTime(orders, true)
	filledCount := calculateFilledOrdersCount(orders, openTime, true)

	// Ожидаем только 1 исполненный ордер
	expectedCount := 1
	if filledCount != expectedCount {
		t.Errorf("Ожидалось %d ордеров (отменённые должны игнорироваться), получено %d", expectedCount, filledCount)
	}
}

// TestHedgeMode_OnlyCountsOpeningOrders проверяет, что считаются только открывающие ордера
// Для LONG - только BUY, для SHORT - только SELL
func TestHedgeMode_OnlyCountsOpeningOrders(t *testing.T) {
	orders := []*futures.Order{
		// Открытие LONG
		{
			OrderID:          1001,
			Symbol:           "SOLUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "10",
			Time:             1767000000000,
			UpdateTime:       1767000000000,
		},
		// Усреднение LONG (BUY)
		{
			OrderID:          1002,
			Symbol:           "SOLUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "5",
			Time:             1767100000000,
			UpdateTime:       1767100000000,
		},
		// Частичное закрытие LONG (SELL) - НЕ должен учитываться
		{
			OrderID:          1003,
			Symbol:           "SOLUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			PositionSide:     futures.PositionSideTypeLong,
			ExecutedQuantity: "3",
			Time:             1767200000000,
			UpdateTime:       1767200000000,
		},
	}

	openTime := calculatePositionOpenTime(orders, true)
	filledCount := calculateFilledOrdersCount(orders, openTime, true)

	// Ожидаем только 2 BUY ордера, SELL не учитывается
	expectedCount := 2
	if filledCount != expectedCount {
		t.Errorf("Ожидалось %d ордеров (только BUY для LONG), получено %d", expectedCount, filledCount)
	}
}
