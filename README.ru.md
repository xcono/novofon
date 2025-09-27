# Novofon Documentation Generator

Этот набор скриптов автоматически генерирует документацию и OpenAPI спецификации из HTML документации Novofon API.

## Компоненты

### 1. `enhanced_html_parser.py`
Парсер HTML документации, который извлекает информацию об API endpoints и генерирует OpenAPI спецификации.

**Функциональность:**
- Парсит HTML файлы документации
- Извлекает параметры запросов и ответов
- Генерирует OpenAPI 3.0 спецификации
- Создает структурированные данные об API endpoints

**Использование:**
```bash
python scripts/enhanced_html_parser.py \
  --input temp-html/data_api \
  --output . \
  --api-type data
```

### 2. `html_to_markdown_converter.py`
Конвертер HTML в Markdown для создания читаемой документации.

**Функциональность:**
- Конвертирует HTML в чистый Markdown
- Сохраняет структуру таблиц
- Обрабатывает код блоки
- Удаляет навигационные элементы

**Использование:**
```bash
python scripts/html_to_markdown_converter.py \
  --input temp-html/data_api \
  --output . \
  --api-type data
```

## GitHub Action

Workflow `.github/workflows/novofon.yaml` автоматически:

1. **Клонирует документацию** из `novofon/novofon.github.io`
2. **Генерирует OpenAPI specs** для Data и Call API
3. **Конвертирует HTML в Markdown**
4. **Создает структуру файлов**
5. **Коммитит изменения** в репозиторий

## Структура выходных файлов

```
openai/
├── data/               # Markdown документация Data API
└── calls/              # Markdown документация Call API
docs/
├── data/               # OpenAPI спецификации Data API
└── calls/              # OpenAPI спецификации Call API
```

## Установка зависимостей

```bash
pip install -r scripts/requirements.txt
```

## Локальный запуск

```bash
# Генерация OpenAPI specs
python scripts/enhanced_html_parser.py \
  --input temp-html/data_api \
  --output . \
  --api-type data

# Конвертация в Markdown
python scripts/html_to_markdown_converter.py \
  --input temp-html/data_api \
  --output . \
  --api-type data
```

## Поддерживаемые API типы

- `data` - Data API документация
- `calls` - Call API документация

## Особенности парсера

- **Автоматическое извлечение** параметров из HTML таблиц
- **Поддержка JSON-RPC 2.0** формата
- **Валидация** обязательных параметров
- **Обработка** различных типов данных
- **Сохранение** структуры и иерархии документации
