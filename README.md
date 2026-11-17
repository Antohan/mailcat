# mailcat — пакет для AI-разработки

Read-only клиент корпоративной почты и календаря (on-prem Exchange) на Go + TUI.
**Этот репозиторий — стартовый пакет**: план разработки и инструкции для
AI-агента (opencode + GLM). Код появится в результате работы агента по этапам.

## Что здесь лежит

| Файл/каталог | Назначение |
|---|---|
| `AGENTS.md` | инструкции, которые opencode читает автоматически |
| `docs/PLAN.md` | поэтапный план (этапы 0–6, Definition of Done) |
| `docs/ARCHITECTURE.md` | архитектура будущего приложения |
| `docs/EWS-REFERENCE.md` | точные SOAP-шаблоны Exchange — источник правды по протоколу |
| `docs/SECURITY.md` | обязательный чек-лист безопасности |
| `docs/TESTING.md` | тесты + ручные smoke-сценарии |
| `config.example.toml` | формат конфига приложения |
| `opencode.json` | права агента и настройки |
| `.opencode/agents/` | субагенты: go-implementer, ews-specialist, security-reviewer |
| `.opencode/skills/ews-reference/` | скил с шаблонами EWS (анти-галлюцинации) |
| `.opencode/commands/` | `/stage`, `/check` |

## Быстрый старт

1. **Скопируйте пакет в корень нового репозитория** (файлы — как есть,
   включая скрытый `.opencode/`).
2. **Настройте opencode** (если ещё не настроен): установка
   `curl -fsSL https://opencode.ai/install | bash`, затем `opencode auth login`
   или блок `providers` из примера ниже.
3. **Подготовьте данные для приложения** (понадобятся на этапе дымовых тестов):
   - файл корпоративного root-CA в PEM → `~/.config/mailcat/company-root-ca.pem`
   - пароль в Keychain: `security add-generic-password -a "$USER" -s mailcat -w`
   - `cp config.example.toml ~/.config/mailcat/config.toml` и заполнить
4. **Запустите сессию разработки**: `opencode`, затем команда
   `/stage 0` — агент начнёт с этапа 0 плана и пойдёт по этапам.
5. После каждой задачи агент прогоняет `/check`; этапы завершаются
   smoke-сценариями из `docs/TESTING.md §5` (нужен VPN и живой сервер).

## Подключение модели GLM (если не настроено глобально)

В `~/.config/opencode/opencode.json` (глобально) или в `opencode.json` проекта:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "zai-glm": {
      "npm": "@ai-sdk/openai-compatible",
      "options": { "baseURL": "https://api.z.ai/api/anthropic" },
      "models": {
        "glm-5.3": {}
      }
    }
  }
}
```

Ключ — в переменной окружения (см. документацию вашего плана GLM / z.ai).
Если модель уже настроена глобально — блок не нужен; агенты наследуют модель
сессии. (Проверьте актуальное имя модели в вашем тарифе.)

## Рекомендации по окружению VS Code

- **Go** (gopls) — официальный плагин: форматирование, диагностика, тесты
- **Error Lens** — инлайн-подсветка ошибок компиляции
- **Even Better TOML** — подсветка/схема для `config.toml`
- Терминал: iTerm2 или kitty желательны (лучший рендер TUI; для inline-картинок
  в будущем — только они)

## Правила безопасности пакета

- Не коммитьте: `config.toml` c реальными значениями, файлы CA, экспорты
  Keychain. Пример уже в `.gitignore`-паттернах: `config.toml`, `*.pem`,
  `mailcat.log`.
- Пароль приложения живёт ТОЛЬКО в Keychain (сервис `mailcat`).
- Корпоративный CA не добавлять в системное хранилище — приложение читает его
  из файла в изолированный TLS-пул (см. `docs/SECURITY.md §1`).

## Почему такие решения (кратко)

| Решение | Причина |
|---|---|
| Go + TUI (tview) | нет браузерного движка → меньше поверхность атаки; HTML писем → текст |
| Свой EWS-клиент | готовые Go-обёртки EWS заброшены; 4 операции дешевле написать по шаблонам |
| Basic → NTLM | типично для on-prem; go-ntlmssp поддерживается Microsoft/Azure |
| Изолированный CA-пул | корп. сертификат доверителен только внутри приложения |
| Read-only whitelist | mailsafe by construction: клиент не может отправить/удалить |
