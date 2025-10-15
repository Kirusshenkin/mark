# 🤖 Crypto Trading Bot - DCA + Grid + AI

**Версия:** 2.0 (Stage 5: Hybrid AI Architecture)

Полнофункциональный Telegram-бот для криптотрейдинга с поддержкой DCA, Grid Trading, Auto-Sell стратегий и **гибридной AI архитектуры** (локальная + облачная), интегрированный с биржей Bybit и PostgreSQL.

## 🎯 Основные возможности

### 📈 DCA (Dollar Cost Averaging)
- Автоматическая покупка криптовалюты по расписанию
- Настраиваемая сумма покупки и интервал
- Расчет средней цены входа
- Отслеживание инвестированной суммы

### 💰 Auto-Sell
- Автоматическая фиксация прибыли при достижении целевого процента
- Настраиваемый процент триггера и объем продажи
- Возможность включения/выключения
- Отслеживание реализованной прибыли

### 🔷 Grid Trading
- Автоматические сетки ордеров (buy low, sell high)
- Настраиваемое количество уровней и spacing
- Отслеживание метрик и P&L
- Поддержка нескольких активов одновременно

### 🤖 Stage 4: Autonomous Trading
- **Policy Engine** с настраиваемыми профилями (conservative/balanced/aggressive)
- **Circuit Breaker** для автоматической остановки при критических ситуациях
- **Kill Switch** для экстренной остановки всей торговли
- **Execution Layer** с валидацией через Policy Engine
- **Orchestrator** с режимами: shadow/pilot/full

### 🧠 Stage 5: Hybrid AI Architecture ⭐ NEW!
- **90% запросов → Локальная модель** (Qwen2.5-Coder-14B через Ollama)
  - ChatAgent для общения в Telegram
  - AnalysisAgent для рыночного анализа
  - **Стоимость: $0** | Латентность: ~0.5s
- **10% запросов → Облачная модель** (Moonshot Kimi K2)
  - DecisionAgent для стратегических решений
  - Автономное управление портфелем
  - **Стоимость: ~$0.003/запрос** | Латентность: ~1.5s
- **Экономия: 80-90%** по сравнению с полностью облачным решением
- **AgentRouter** для умной маршрутизации запросов
- **ActionExecutor** с 27 функциями для выполнения AI команд

### 📲 Telegram Bot
- Полное управление через Telegram
- Команды и естественный язык
- Уведомления о сделках
- Просмотр статуса и истории

### 🔄 Интеграция с Bybit
- Spot API v5
- Получение цен и балансов
- Размещение рыночных ордеров
- Проверка статуса ордеров

### 🗄️ PostgreSQL
- Хранение истории сделок
- Отслеживание балансов
- Динамическая конфигурация
- Логирование событий

## 🏗️ Архитектура (Stage 5: Hybrid AI)

```
┌─────────────────────────────────────────┐
│         Telegram Bot Interface          │ ← Пользовательский интерфейс
└──────────────────┬──────────────────────┘
                   │
    ┌──────────────▼──────────────┐
    │    AgentRouter (Stage 5)    │ ← Умная маршрутизация
    └──┬───────────┬──────────┬───┘
       │           │          │
   ┌───▼───┐   ┌──▼───┐   ┌──▼────────┐
   │ Chat  │   │Analysis│  │ Decision  │
   │ Agent │   │ Agent  │  │  Agent    │
   │(Local)│   │(Local) │  │ (Cloud)   │
   └───┬───┘   └───┬────┘  └───┬───────┘
       │           │           │
       └───────────┴───────────┴─────────┐
                   │                     │
       ┌───────────▼────────────┐        │
       │   ActionExecutor       │ ← 27 AI функций
       └───────────┬────────────┘        │
                   │                     │
    ┌──────────────▼──────────────────┐  │
    │  Orchestrator (Stage 4)         │ ◄─┘
    │  ├─ Policy Engine               │
    │  ├─ Circuit Breaker             │
    │  └─ Kill Switch                 │
    └──────────────┬──────────────────┘
                   │
       ┌───────────┼───────────┐
       │           │           │
   ┌───▼─────┐ ┌──▼───────┐ ┌─▼─────────┐
   │Strategy │ │ Executor │ │  Exchange │
   │  Layer  │ │  Layer   │ │ (Bybit v5)│
   │ DCA/Grid│ │          │ └───────────┘
   └───┬─────┘ └────┬─────┘
       │            │
       └────────┬───┘
                │
       ┌────────▼────────┐
       │   PostgreSQL    │ ← Хранение данных
       └─────────────────┘
```

## 📁 Структура проекта

```
.
├── cmd/
│   └── bot/
│       └── main.go              # Точка входа
├── internal/
│   ├── config/
│   │   └── config.go            # Конфигурация
│   ├── exchange/
│   │   └── bybit.go             # Bybit API клиент
│   ├── strategy/
│   │   ├── dca.go               # DCA стратегия
│   │   └── autosell.go          # Auto-Sell стратегия
│   ├── telegram/
│   │   └── bot.go               # Telegram бот
│   ├── ai/
│   │   └── client.go            # AI интеграция
│   └── storage/
│       ├── models.go            # Модели данных
│       └── postgres.go          # PostgreSQL клиент
├── pkg/
│   └── utils/
│       └── logger.go            # Логирование
├── .env.example                 # Пример конфигурации
├── go.mod
└── README.md
```

## 🚀 Быстрый старт

**Рекомендуемый способ - Docker Compose (всё в одной команде!):**

### Запуск через Docker Compose (рекомендуется)

```bash
# 1. Клонируйте репозиторий
git clone <your-repo>
cd crypto-trading-bot

# 2. Создайте .env файл
cp .env.example .env
# Отредактируйте .env и заполните обязательные параметры

# 3. Запустите всё одной командой
docker-compose up -d

# 4. Проверьте логи
docker-compose logs -f bot
```

**Готово!** PostgreSQL, Ollama и бот автоматически настроены и запущены.

**⭐ Для Stage 5 Hybrid AI:**
1. Установите Ollama локально (см. раздел "Stage 5: Hybrid AI параметры")
2. Добавьте Moonshot API key в `.env`
3. Бот автоматически будет использовать гибридную архитектуру

📖 **Подробные инструкции:**
- [QUICKSTART.md](QUICKSTART.md) - базовый запуск
- `.claude/stage5-integration-complete.md` - Stage 5 AI setup

---

### Альтернативный способ - Ручная установка

<details>
<summary>Нажмите, чтобы развернуть</summary>

#### 1. Требования

- Go 1.24+
- PostgreSQL 12+
- Telegram Bot Token
- Bybit API ключи
- (Опционально) AI API ключи (Qwen/Kimi)

#### 2. Установка

```bash
# Клонируем репозиторий
git clone <your-repo>
cd crypto-trading-bot

# Устанавливаем зависимости
go mod download
```

#### 3. Настройка базы данных

```bash
# Создаем базу данных
createdb crypto_trading_bot

# Миграции выполнятся автоматически при первом запуске
```

#### 4. Конфигурация

Создайте `.env` файл на основе `.env.example`:

```bash
cp .env.example .env
```

Заполните следующие обязательные параметры:

```env
# Telegram
TELEGRAM_BOT_TOKEN=your_bot_token
TELEGRAM_CHAT_ID=your_chat_id

# Bybit
BYBIT_API_KEY=your_api_key
BYBIT_API_SECRET=your_api_secret

# Database
DB_HOST=localhost
DB_PASSWORD=your_password

# AI (опционально)
AI_API_KEY=your_ai_key
AI_BASE_URL=https://api.qwen.ai
```

#### 5. Запуск

```bash
go run cmd/bot/main.go
```

Или соберите бинарник:

```bash
go build -o bot cmd/bot/main.go
./bot
```

📖 Подробная инструкция: [SETUP.md](SETUP.md)

</details>

## 💬 Команды Telegram бота (Production Ready!)

### 📊 Информационные команды

| Команда | Параметры | Описание | Пример |
|---------|-----------|----------|---------|
| `/status` | - | Показывает текущий статус: активные активы, стратегии, автосейл, грид, аптайм | `/status` |
| `/history` | `[SYMBOL] [N]` | История последних N сделок (по умолчанию 10) | `/history BTCUSDT 20` |
| `/config` | - | Текущая конфигурация всех активов и риск-лимитов | `/config` |
| `/price` | `<SYMBOL>` | Текущая цена, средняя цена входа, изменение в % | `/price BTC` |
| `/portfolio` | - | Сводка портфеля: инвестировано, текущая стоимость, P&L, распределение активов | `/portfolio` |
| `/risk` | - | Статус рисков: экспозиция, дневные убытки, лимиты (Admin) | `/risk` |
| `/help` | - | Справка по всем командам | `/help` |

### 💰 Торговые команды

| Команда | Параметры | Описание | Пример |
|---------|-----------|----------|---------|
| `/buy` | `[SYMBOL] [AMOUNT]` | Рыночная покупка на указанную сумму USDT | `/buy BTCUSDT 20` |
| `/sell` | `<PERCENT> [SYMBOL]` | Продать % позиции (1-100%) | `/sell 50 BTC` |

**Валидация:**
- Проверка баланса USDT
- Проверка лимитов (MaxOrderSize, MaxPositionSize)
- Проверка Emergency Stop
- Проверка наличия позиции для продажи

### ⚙️ Auto-Sell команды

| Команда | Параметры | Описание | Пример |
|---------|-----------|----------|---------|
| `/autosellon` | `[SYMBOL]` | Включить Auto-Sell для актива | `/autosellon BTC` |
| `/autoselloff` | `[SYMBOL]` | Выключить Auto-Sell для актива | `/autoselloff BTC` |
| `/autosell` | `[SYMBOL] <TRIGGER_%> <SELL_%>` | Настроить Auto-Sell: триггер и объем продажи | `/autosell BTC 15 50` |

**Пример:** `/autosell BTCUSDT 15 50` = когда прибыль достигнет +15%, продать 50% позиции

### 🔷 Grid Trading команды

| Команда | Параметры | Описание | Пример |
|---------|-----------|----------|---------|
| `/gridinit` | `<SYMBOL> <LEVELS> <SPACING_%> <ORDER_SIZE>` | Инициализировать Grid стратегию | `/gridinit ETHUSDT 10 2.5 100` |
| `/gridstatus` | `<SYMBOL>` | Статус Grid: активные ордера, метрики, P&L | `/gridstatus ETH` |
| `/gridstop` | `<SYMBOL>` | Остановить Grid и отменить все ордера (Admin) | `/gridstop ETH` |

**Пример:** `/gridinit ETHUSDT 10 2.5 100` = 10 уровней с интервалом 2.5%, по 100 USDT на ордер

### 🛡️ Риск-менеджмент (Admin Only)

| Команда | Параметры | Описание | Пример |
|---------|-----------|----------|---------|
| `/risk` | - | Показать текущие лимиты и экспозицию | `/risk` |
| `/panicstop` | `[on\|off]` | Экстренная остановка всей торговли | `/panicstop on` |

### 🧠 Stage 5: Hybrid AI Commands ⭐ NEW!

| Команда | Параметры | Описание | Модель | Пример |
|---------|-----------|----------|--------|---------|
| `/ai_analyze` | `[SYMBOL]` | Рыночный анализ актива | Local Qwen 14B (~0.8s) | `/ai_analyze BTC` |
| `/ai_decision` | - | Стратегическое решение по всему портфелю | Cloud Kimi K2 (~1.5s) | `/ai_decision` |
| `/ai_metrics` | - | Метрики AgentRouter (local vs cloud запросы) | - | `/ai_metrics` |
| `/ai_mode` | `[shadow\|pilot\|full]` | Режим DecisionAgent или показать текущий | - | `/ai_mode pilot` |

**Режимы DecisionAgent:**
- **shadow** - AI анализирует, но не выполняет действия (безопасный тест)
- **pilot** - AI выполняет действия с ограничениями 50% (консервативный)
- **full** - AI полностью автономен (проверенная стратегия)

**Естественный язык (ChatAgent):**

Просто отправьте сообщение боту на **русском или английском языке**:

**English examples:**
- "Buy 20 USDT worth of BTC"
- "Sell 50% of ETH"
- "Set auto-sell at +15%"
- "Show portfolio"
- "What's the current price of BTC?"
- "Initialize Grid for ETHUSDT with 10 levels"

**Русские примеры:**
- "Купи BTC на 20 USDT"
- "Продай 30% позиции"
- "Установи автопродажу на +15%"
- "Покажи портфель"
- "Какая текущая цена BTC?"
- "Инициализируй сетку для ETHUSDT"

**Обработка:** ChatAgent (локальная Qwen 14B) с ToolCalls

### 🔒 Безопасность и права доступа

**Публичные команды** (доступны всем пользователям):
- Все информационные команды
- `/buy`, `/sell` (с подтверждением)
- `/autosellon`, `/autoselloff`, `/autosell`
- `/gridinit`, `/gridstatus`

**Admin-only команды** (требуют прав администратора):
- `/risk` - просмотр лимитов риска
- `/gridstop` - остановка Grid
- `/panicstop` - экстренная остановка

**Опасные команды** (требуют подтверждения через inline кнопки):
- `/sell` - продажа позиции
- `/gridstop` - остановка Grid
- `/panicstop` - экстренная остановка

**Конфигурация:**
```env
TG_ADMINS=123456789,987654321  # ID администраторов (через запятую)
TG_CHAT_WHITELIST=  # Белый список пользователей (пусто = разрешить всем)
```

### 📝 Примечания по использованию

1. **Символы:** Поддерживаются короткие названия (BTC, ETH) и полные (BTCUSDT, ETHUSDT)
2. **Числа:** Можно использовать точку или запятую как десятичный разделитель (10.5 или 10,5)
3. **Проценты:** Можно указывать со знаком % или без него (50% или 50)
4. **Rate limiting:** Максимум 2 команды в секунду на пользователя
5. **Preview Mode:** Установите `PREVIEW_MODE=true` для тестирования без реального исполнения
6. **Языки:** Автоматическое определение языка или установка через `DEFAULT_LANG=ru` или `DEFAULT_LANG=en`

## ⚙️ Конфигурация стратегий

### DCA параметры

```env
DCA_AMOUNT=10              # Сумма в USDT для каждой покупки
DCA_INTERVAL=24h           # Интервал между покупками (24h, 12h, 1w и т.д.)
```

### Auto-Sell параметры

```env
AUTO_SELL_ENABLED=true                  # Включен ли Auto-Sell
AUTO_SELL_TRIGGER_PERCENT=10            # Процент прибыли для активации
AUTO_SELL_AMOUNT_PERCENT=50             # Процент позиции для продажи
PRICE_CHECK_INTERVAL=5m                 # Интервал проверки цены
```

### Stage 5: Hybrid AI параметры ⭐ NEW!

```env
# ===== Local AI (Ollama) =====
LOCAL_AI_ENABLED=true
LOCAL_AI_URL=http://localhost:11434              # или http://host.docker.internal:11434 для Docker
LOCAL_AI_MODEL=qwen2.5-coder:14b

# ===== Cloud AI (Moonshot) =====
CLOUD_AI_ENABLED=true
CLOUD_AI_PROVIDER=moonshot
CLOUD_AI_URL=https://api.moonshot.ai
CLOUD_AI_KEY=sk-your-moonshot-api-key
CLOUD_AI_MODEL=kimi-k2-turbo-preview

# ===== AI Router Configuration =====
AI_USE_LOCAL_FOR_CHAT=true
AI_USE_LOCAL_FOR_ANALYSIS=true
AI_USE_CLOUD_FOR_DECISIONS=true

# ===== Decision Agent =====
AI_DECISION_MODE=pilot                           # shadow | pilot | full
AI_DECISION_INTERVAL=3600                        # секунды (1 час)
```

**Установка Ollama (для M1 Mac):**

```bash
# 1. Установить Ollama
brew install ollama

# 2. Запустить сервер
brew services start ollama

# 3. Загрузить модель (~10GB)
ollama pull qwen2.5-coder:14b

# 4. Проверить
ollama list
```

📖 **Подробная документация:** `.claude/stage5-integration-complete.md`

## 🔒 Безопасность

1. **Никогда** не коммитьте `.env` файл в git
2. Используйте API ключи только с правами на spot trading
3. Установите IP whitelist на Bybit
4. Ограничьте доступ к Telegram боту через `TELEGRAM_CHAT_ID`
5. Используйте отдельные ключи для тестирования

## 📊 База данных

### Таблицы

#### `trades`
Хранит историю всех сделок
- `id`, `symbol`, `side`, `quantity`, `price`, `amount`, `order_id`, `status`, `created_at`

#### `balances`
Текущие балансы по активам
- `id`, `symbol`, `total_quantity`, `available_qty`, `avg_entry_price`, `total_invested`, `total_sold`, `realized_profit`, `updated_at`

#### `config_params`
Динамические параметры конфигурации
- `id`, `key`, `value`, `updated_at`

#### `logs`
Системные события и логи
- `id`, `level`, `message`, `data`, `created_at`

## 🧪 Тестирование

```bash
# Запуск тестов
go test ./...

# С покрытием
go test -cover ./...
```

## 🛠️ Разработка

### Добавление новой стратегии

1. Создайте новый файл в `internal/strategy/`
2. Реализуйте интерфейс стратегии
3. Добавьте в `main.go`
4. Обновите Telegram команды

### Добавление новой биржи

1. Создайте новый файл в `internal/exchange/`
2. Реализуйте методы:
   - `GetPrice()`
   - `GetBalance()`
   - `PlaceOrder()`
3. Обновите конфигурацию

## 📝 TODO / Roadmap

### ✅ Реализовано (v2.0)

**Stage 1-3:**
- [x] DCA стратегия
- [x] Auto-Sell стратегия
- [x] Grid Trading стратегия
- [x] Portfolio Manager
- [x] Risk Manager
- [x] Bybit Spot API v5 интеграция
- [x] PostgreSQL хранилище (8 репозиториев)
- [x] Telegram бот интерфейс (18+ команд)
- [x] Docker контейнеризация

**Stage 4: Autonomous Trading**
- [x] Policy Engine с профилями (conservative/balanced/aggressive)
- [x] Circuit Breaker для автоматической защиты
- [x] Kill Switch для экстренной остановки
- [x] Execution Layer с валидацией
- [x] Orchestrator с режимами (shadow/pilot/full)

**Stage 5: Hybrid AI Architecture** ⭐
- [x] AgentRouter для умной маршрутизации
- [x] ChatAgent (локальная Qwen 14B)
- [x] AnalysisAgent (локальная Qwen 14B)
- [x] DecisionAgent (облачная Kimi K2)
- [x] ActionExecutor с 27 AI функциями
- [x] Интеграция Ollama для M1 Mac
- [x] Экономия 80-90% на AI costs

### 🔄 В разработке

- [ ] Web Search интеграция для новостных сигналов
- [ ] Redis кэширование для частых AI запросов
- [ ] Scheduled DecisionAgent (автоматические решения каждый час)
- [ ] Prometheus + Grafana мониторинг
- [ ] Unit и Integration тесты

### 📋 В планах

**Функциональность:**
- [ ] Поддержка нескольких торговых пар одновременно
- [ ] Веб-интерфейс для мониторинга
- [ ] Интеграция с TradingView сигналами
- [ ] Backtesting функционал
- [ ] Trailing stop-loss
- [ ] Portfolio rebalancing

**Интеграции:**
- [ ] Поддержка других бирж (Binance, OKX)
- [ ] Уведомления через Discord/Slack
- [ ] Twitter sentiment анализ
- [ ] Vision model для анализа графиков

**DevOps:**
- [ ] Kubernetes deployment манифесты
- [ ] Полное покрытие тестами (>80%)
- [ ] CI/CD pipeline

## 📊 Текущее состояние проекта

**Версия:** 2.0 (Stage 5: Hybrid AI Architecture)
**Статус:** ✅ Production Ready

### Ключевые метрики:

| Метрика | Значение |
|---------|----------|
| **Код** | 10,000+ строк Go кода |
| **Архитектура** | Clean Architecture + SOLID |
| **Telegram команды** | 18+ команд |
| **AI Agents** | 4 агента (Chat, Analysis, Decision, Router) |
| **AI Functions** | 27 ActionExecutor функций |
| **Стратегии** | 5 (DCA, Grid, Auto-Sell, Risk, Portfolio) |
| **Репозитории** | 8 (Clean Repository Pattern) |
| **Docker** | ✅ Полная контейнеризация |
| **AI стоимость** | $3-5/мес (было $20-50) |
| **AI латентность** | 0.5s для 90% запросов |

### Документация:

📚 Полная документация в `.claude/`:
- **stage5-integration-complete.md** - Гибридная AI архитектура (быстрый старт)
- **ai-architecture.md** - Архитектура AI (815 строк)
- **ollama-m1-setup.md** - Setup Ollama на M1 Mac (673 строки)
- **ai-quickstart.md** - Быстрый старт AI (542 строки)
- **CLAUDE.md** - Инструкции для разработчиков
- **architecture.md** - Clean Architecture проекта
- **coding-standards.md** - Стандарты кодирования
- **patterns.md** - Паттерны проектирования
- **project-structure.md** - Структура проекта

## 🤝 Contributing

Приветствуются pull requests! Для больших изменений сначала откройте issue для обсуждения.

**Перед началом разработки:**
1. Прочитай `.claude/CLAUDE.md` - инструкции и стандарты
2. Изучи `.claude/architecture.md` - архитектуру проекта
3. Следуй Clean Architecture и SOLID принципам
4. Используй Repository Pattern для работы с данными

## 📄 Лицензия

MIT License

## ⚠️ Disclaimer

Этот бот предоставляется "как есть" без каких-либо гарантий. Торговля криптовалютой сопряжена с рисками. Автор не несет ответственности за финансовые потери.

**Используйте на свой страх и риск!**

## 📞 Поддержка

Если у вас возникли вопросы или проблемы:

1. Проверьте документацию
2. Посмотрите существующие issues
3. Создайте новый issue с подробным описанием проблемы

---

**Создано с ❤️ для автоматизации криптотрейдинга**
# mark
