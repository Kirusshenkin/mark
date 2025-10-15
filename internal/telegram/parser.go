package telegram

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CommandArgs представляет распарсенные аргументы команды
type CommandArgs struct {
	Command string
	Symbol  string
	Amount  float64
	Percent float64
	Count   int
	Levels  int
	Spacing float64
	Trigger float64
	Action  string // on/off для autosell, panic
	Raw     []string
}

// CommandType представляет тип команды
type CommandType string

const (
	// Info commands
	CmdStatus    CommandType = "status"
	CmdHistory   CommandType = "history"
	CmdConfig    CommandType = "config"
	CmdPrice     CommandType = "price"
	CmdPortfolio CommandType = "portfolio"
	CmdRisk      CommandType = "risk"
	CmdHelp      CommandType = "help"

	// Trading commands
	CmdBuy  CommandType = "buy"
	CmdSell CommandType = "sell"

	// Auto-Sell commands
	CmdAutoSellOn  CommandType = "autosellon"
	CmdAutoSellOff CommandType = "autoselloff"
	CmdAutoSell    CommandType = "autosell"

	// Grid commands
	CmdGridInit   CommandType = "gridinit"
	CmdGridStatus CommandType = "gridstatus"
	CmdGridStop   CommandType = "gridstop"

	// Admin commands
	CmdPanicStop CommandType = "panicstop"

	// AI commands
	CmdAnalysis CommandType = "analysis"
)

// ParseCommand парсит команду и аргументы
func ParseCommand(text string) (*CommandArgs, error) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return nil, fmt.Errorf("not a command")
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	cmd := strings.ToLower(strings.TrimPrefix(parts[0], "/"))
	args := &CommandArgs{
		Command: cmd,
		Raw:     parts[1:],
	}

	// Парсим в зависимости от команды
	switch cmd {
	case "status", "help", "start", "portfolio", "risk", "config":
		// Команды без параметров
		return args, nil

	case "price":
		// /price BTCUSDT
		if len(parts) < 2 {
			return nil, fmt.Errorf("usage: /price SYMBOL")
		}
		args.Symbol = normalizeSymbol(parts[1])
		return args, nil

	case "history":
		// /history [SYMBOL] [N]
		if len(parts) >= 2 {
			// Проверяем, это символ или число
			if isNumber(parts[1]) {
				args.Count = parseInt(parts[1], 10)
			} else {
				args.Symbol = normalizeSymbol(parts[1])
				if len(parts) >= 3 && isNumber(parts[2]) {
					args.Count = parseInt(parts[2], 10)
				}
			}
		}
		if args.Count == 0 {
			args.Count = 10
		}
		return args, nil

	case "buy":
		// /buy [SYMBOL] [AMOUNT]
		if len(parts) == 1 {
			// /buy без параметров - используем дефолтный DCA
			return args, nil
		}
		if len(parts) == 2 {
			// Может быть /buy BTCUSDT или /buy 20
			if isNumber(parts[1]) {
				args.Amount = parseFloat(parts[1])
			} else {
				args.Symbol = normalizeSymbol(parts[1])
			}
		} else if len(parts) >= 3 {
			args.Symbol = normalizeSymbol(parts[1])
			args.Amount = parseFloat(parts[2])
		}
		return args, nil

	case "sell":
		// /sell <PERCENT> [SYMBOL]
		if len(parts) < 2 {
			return nil, fmt.Errorf("usage: /sell <PERCENT> [SYMBOL]")
		}
		args.Percent = parseFloat(parts[1])
		if len(parts) >= 3 {
			args.Symbol = normalizeSymbol(parts[2])
		}
		if args.Percent <= 0 || args.Percent > 100 {
			return nil, fmt.Errorf("percent must be between 1 and 100")
		}
		return args, nil

	case "autosellon", "autosell_on":
		// /autosellon [SYMBOL]
		if len(parts) >= 2 {
			args.Symbol = normalizeSymbol(parts[1])
		}
		args.Action = "on"
		return args, nil

	case "autoselloff", "autosell_off":
		// /autoselloff [SYMBOL]
		if len(parts) >= 2 {
			args.Symbol = normalizeSymbol(parts[1])
		}
		args.Action = "off"
		return args, nil

	case "autosell":
		// /autosell [SYMBOL] <TRIGGER_%> <SELL_%>
		if len(parts) < 3 {
			return nil, fmt.Errorf("usage: /autosell [SYMBOL] <TRIGGER_%%> <SELL_%%>")
		}

		// Если 3 параметра: SYMBOL TRIGGER SELL
		// Если 2 параметра: TRIGGER SELL (используем дефолтный символ)
		if len(parts) == 3 {
			args.Trigger = parseFloat(parts[1])
			args.Percent = parseFloat(parts[2])
		} else {
			args.Symbol = normalizeSymbol(parts[1])
			args.Trigger = parseFloat(parts[2])
			args.Percent = parseFloat(parts[3])
		}

		if args.Trigger <= 0 {
			return nil, fmt.Errorf("trigger percent must be positive")
		}
		if args.Percent <= 0 || args.Percent > 100 {
			return nil, fmt.Errorf("sell percent must be between 1 and 100")
		}
		return args, nil

	case "gridinit", "grid_init":
		// /gridinit <SYMBOL> <LEVELS> <SPACING_%> <ORDER_SIZE>
		if len(parts) < 5 {
			return nil, fmt.Errorf("usage: /gridinit <SYMBOL> <LEVELS> <SPACING_%%> <ORDER_SIZE_USDT>")
		}
		args.Symbol = normalizeSymbol(parts[1])
		args.Levels = parseInt(parts[2], 10)
		args.Spacing = parseFloat(parts[3])
		args.Amount = parseFloat(parts[4])

		if args.Levels <= 0 || args.Levels > 100 {
			return nil, fmt.Errorf("levels must be between 1 and 100")
		}
		if args.Spacing <= 0 || args.Spacing > 50 {
			return nil, fmt.Errorf("spacing must be between 0.1 and 50")
		}
		if args.Amount <= 0 {
			return nil, fmt.Errorf("order size must be positive")
		}
		return args, nil

	case "gridstatus", "grid_status":
		// /gridstatus <SYMBOL>
		if len(parts) < 2 {
			return nil, fmt.Errorf("usage: /gridstatus <SYMBOL>")
		}
		args.Symbol = normalizeSymbol(parts[1])
		return args, nil

	case "gridstop", "grid_stop":
		// /gridstop <SYMBOL>
		if len(parts) < 2 {
			return nil, fmt.Errorf("usage: /gridstop <SYMBOL>")
		}
		args.Symbol = normalizeSymbol(parts[1])
		return args, nil

	case "panicstop":
		// /panicstop [on|off]
		if len(parts) >= 2 {
			args.Action = strings.ToLower(parts[1])
			if args.Action != "on" && args.Action != "off" {
				return nil, fmt.Errorf("usage: /panicstop [on|off]")
			}
		} else {
			// Без параметра - показать статус
			args.Action = "status"
		}
		return args, nil

	case "analysis":
		// /analysis [SYMBOL]
		if len(parts) >= 2 {
			args.Symbol = normalizeSymbol(parts[1])
		}
		return args, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}
}

// normalizeSymbol приводит символ к стандартному виду
func normalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// Если не указан USDT в конце, добавляем
	if !strings.HasSuffix(symbol, "USDT") && !strings.HasSuffix(symbol, "USDC") {
		// Проверяем популярные тикеры
		if symbol == "BTC" || symbol == "ETH" || symbol == "SOL" ||
		   symbol == "BNB" || symbol == "XRP" || symbol == "ADA" ||
		   symbol == "DOGE" || symbol == "MATIC" || symbol == "DOT" ||
		   symbol == "AVAX" || symbol == "LINK" || symbol == "UNI" ||
		   symbol == "TON" || symbol == "ARB" || symbol == "OP" ||
		   symbol == "SHIB" || symbol == "PEPE" || symbol == "LTC" {
			symbol = symbol + "USDT"
		}
	}

	return symbol
}

// normalizeCommand нормализует команду (поддержка русского языка)
func normalizeCommand(cmd string) string {
	cmd = strings.ToLower(strings.TrimSpace(cmd))

	// Маппинг русских команд на английские
	ruToEn := map[string]string{
		"статус":      "status",
		"история":     "history",
		"конфиг":      "config",
		"цена":        "price",
		"портфель":    "portfolio",
		"риск":        "risk",
		"помощь":      "help",
		"купить":      "buy",
		"продать":     "sell",
		"автопродажа": "autosell",
		"сетка":       "gridinit",
		"стоп":        "gridstop",
		"анализ":      "analysis",
	}

	if enCmd, ok := ruToEn[cmd]; ok {
		return enCmd
	}

	return cmd
}

// normalizeAction нормализует действие (on/off)
func normalizeAction(action string) string {
	action = strings.ToLower(strings.TrimSpace(action))

	actionMap := map[string]string{
		"вкл":      "on",
		"включить": "on",
		"да":       "on",
		"yes":      "on",
		"выкл":     "off",
		"выключить": "off",
		"нет":      "off",
		"no":       "off",
	}

	if normalized, ok := actionMap[action]; ok {
		return normalized
	}

	return action
}

// ExtractNumbers извлекает числа из текста (для NLU)
func ExtractNumbers(text string) []float64 {
	// Регулярка для чисел (включая с запятыми и точками)
	re := regexp.MustCompile(`[-+]?[0-9]*[.,]?[0-9]+`)
	matches := re.FindAllString(text, -1)

	numbers := make([]float64, 0, len(matches))
	for _, match := range matches {
		if num := parseFloat(match); num != 0 {
			numbers = append(numbers, num)
		}
	}

	return numbers
}

// ExtractSymbols извлекает символы криптовалют из текста (для NLU)
func ExtractSymbols(text string) []string {
	text = strings.ToUpper(text)

	// Популярные тикеры
	tickers := []string{"BTC", "ETH", "SOL", "BNB", "XRP", "ADA", "DOGE",
		"MATIC", "DOT", "AVAX", "LINK", "UNI", "TON", "ARB", "OP", "SHIB", "PEPE", "LTC",
		"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"}

	// Используем map для уникальности
	found := make(map[string]bool)
	symbols := []string{}

	for _, ticker := range tickers {
		if strings.Contains(text, ticker) {
			normalized := normalizeSymbol(ticker)
			if !found[normalized] {
				found[normalized] = true
				symbols = append(symbols, normalized)
			}
		}
	}

	return symbols
}

// isNumber проверяет, является ли строка числом
func isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// parseFloat безопасно парсит float с поддержкой запятой
func parseFloat(s string) float64 {
	// Убираем процент если есть
	s = strings.TrimSuffix(s, "%")

	// Заменяем запятую на точку (русская локаль)
	s = strings.Replace(s, ",", ".", 1)

	// Убираем пробелы (тысячные разделители)
	s = strings.ReplaceAll(s, " ", "")

	val, _ := strconv.ParseFloat(s, 64)
	return val
}

// parseInt безопасно парсит int
func parseInt(s string, defaultVal int) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}
