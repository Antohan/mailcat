# mailcat — HANDOFF

## Дальше
Задачи RD-1…RD-7 итерации 3 выполнены (чекбоксы PLAN-ITER3.md). Дальше — вне
кодовой сессии: сверка DoD («Definition of Done» PLAN-ITER3.md) и решение
пользователя по опциональным О-1…О-3; smoke по TESTING.md §5 — TODO юзера.

## Сделано
- b0f4eb4 — RD-7: спиннер + сохранение выделения/скролла; 9c2356e — RD-6;
  45e14ec — RD-5; 8a924bd — RD-4; 1c39486/7941f3a — RD-3; c37695a — RD-2;
  d26e076 — RD-1; e3e8daf — Release 2.

## Открыто
- TODO пользователя (TESTING.md §5): smoke S3–S7 (в т.ч. RD-7: спиннер при
  тормозящем VPN, u возвращает курсор на то же письмо, пустая папка), темы,
  resize; визуал Terminal.app/VS Code.
- НИЗКО: attachments.go ~120 — частичный файл при ошибке Write; Write-операции
  (mark-as-read/delete) отложены — решения у пользователя.

## Не трогать
- RD-7: спиннер — прямой SetText, не setStatus (statusGen гасит таймер
  ошибки); errorShown стопает тик; Clear только в rebuildRows по свежему gen.
- RD-6: hint — SetDynamicColors(true), но текст только наш (formatHint из
  hintSeg); серверный текст не писать; теги — hex ролей (Color.Hex()).
- RD-5: preview/details — EWS-текст только Sanitize + tview.Escape; a.event
  dynamic=false; превью — FormatMessageHeader от MessageSummary.
- RD-4: цвет ячейки — в Style (тесты — Style.Decompose()); непрочитанные — accent.
- RD-3: активная папка = открытая (ID, до переключения — имя); счётчик — mainFolders.
- RD-2: заголовок — panelHeader; линия textSecondary; рамка неактивной colorPanelBg.
- RD-1: цвета — только роли theme.go; фокус/выделение — accent.
- UI-9: подсказки→a.hint, сообщения→a.status; endLoad один раз; красный — colorError.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500; read-only; tls в internal/ews.
