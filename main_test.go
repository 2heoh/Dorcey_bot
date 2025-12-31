package main

import (
	"testing"

	"github.com/adshao/go-binance/v2/futures"
)

// Фикстура: симуляция истории ордеров для LSKUSDT
// Сценарий: позиция открывалась и закрывалась несколько раз
// Последнее открытие: ордер ID=1006 (BUY 261 в момент времени 1767159730815)
// После открытия: ещё один ордер ID=1007 (BUY 100)
// Ожидаемый результат: время открытия = 1767159730815, количество ордеров = 2
func createTestOrdersLSK() []*futures.Order {
	return []*futures.Order{
		// Первое открытие позиции (будет закрыто)
		{
			OrderID:          1001,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "127",
			Time:             1766643367148, // 25 декабря 10:16
			UpdateTime:       1766643367148,
		},
		// Закрытие первой позиции
		{
			OrderID:          1002,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			ExecutedQuantity: "127",
			Time:             1766659210282, // 25 декабря 14:40
			UpdateTime:       1766659210282,
		},
		// Второе открытие позиции (будет закрыто)
		{
			OrderID:          1003,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "125",
			Time:             1766784671130, // 27 декабря 01:31
			UpdateTime:       1766784671130,
		},
		// Закрытие второй позиции
		{
			OrderID:          1004,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeSell,
			ExecutedQuantity: "125",
			Time:             1766831349301, // 27 декабря 14:29
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
			Time:             1767159730815, // 31 декабря 09:42:10
			UpdateTime:       1767159730815,
		},
		// Дополнительный ордер после открытия (усреднение)
		{
			OrderID:          1007,
			Symbol:           "LSKUSDT",
			Status:           futures.OrderStatusTypeFilled,
			Side:             futures.SideTypeBuy,
			ExecutedQuantity: "100",
			Time:             1767177846634, // 31 декабря 14:44
			UpdateTime:       1767177846634,
		},
	}
}

// TestCalculatePositionOpenTime_LongPosition проверяет определение времени открытия LONG позиции
func TestCalculatePositionOpenTime_LongPosition(t *testing.T) {
	orders := createTestOrdersLSK()
	
	openTime := calculatePositionOpenTime(orders, true)
	
	expectedTime := int64(1767159730815) // 31 декабря 09:42:10
	if openTime != expectedTime {
		t.Errorf("Ожидалось время открытия %d, получено %d", expectedTime, openTime)
	}
}

// TestCalculateFilledOrdersCount проверяет подсчёт исполненных ордеров после открытия позиции
func TestCalculateFilledOrdersCount(t *testing.T) {
	orders := createTestOrdersLSK()
	positionOpenTime := int64(1767159730815) // 31 декабря 09:42:10
	
	filledCount := calculateFilledOrdersCount(orders, positionOpenTime)
	
	expectedCount := 2 // Ордера 1006 и 1007
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
	filledCount := calculateFilledOrdersCount(orders, 1768000000000)
	
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
