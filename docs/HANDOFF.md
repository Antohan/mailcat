# mailcat — HANDOFF

## Дальше
RD-7: микро-UX — braille-спиннер (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏) в строке сообщений; обновление
списка (u, смена папки) сохраняет выделение по ItemId и скролл; пустая папка —
muted-заглушка. Читать docs/PLAN-ITER3.md (RD-7), internal/ui/app.go
(beginLoad/endLoad), internal/ui/mail.go.

## Сделано
- 9c2356e — RD-6: нижняя панель — клавиши accent жирным, описания
  textSecondary (hintSeg + formatHint); 45e14ec — RD-5: просмотр письма;
  8a924bd — RD-4: список; 1c39486/7941f3a — RD-3: папки; c37695a — RD-2:
  каркас; d26e076 — RD-1: палитра; e3e8daf — Release 2.

## Открыто
- TODO пользователя (TESTING.md §5): smoke S3–S6 (превью/детали RD-5, нижняя
  панель RD-6), темы, resize; визуал Terminal.app/VS Code.
- НИЗКО: attachments.go ~120 — частичный файл при ошибке Write; Write-операции
  (mark-as-read/delete) отложены — решения у пользователя.

## Не трогать
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
