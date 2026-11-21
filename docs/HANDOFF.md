# mailcat — HANDOFF

## Дальше
Все задачи итерации 3 выполнены (RD-1…RD-7 + опции О-1/О-3). Опция О-2
(скруглённые рамки модалов) — делать, только если пользователь выберет.
Вне кодовой сессии: сверка DoD (PLAN-ITER3.md), smoke TESTING.md §5 (TODO
юзера), релиз — /release по явному запросу.

## Сделано
- d969d0b — O-1; ad151f9 — O-3; b0f4eb4 — RD-7; 9c2356e — RD-6; 45e14ec — RD-5;
  8a924bd — RD-4; 1c39486/7941f3a — RD-3; c37695a — RD-2; d26e076 — RD-1; e3e8daf — R2.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7, темы, resize; О-1: разделители на
  живом сервере; О-3: баннер/заливка в Terminal.app и VS Code (█, truecolor).
- НИЗКО: attachments.go ~120 — частичный файл при ошибке Write; Write-операции отложены.

## Не трогать
- О-1: itemRows — единственный маппинг «строка таблицы ↔ s.items» (сырой row колбэков
  в items писать нельзя); разделитель — кол. 0 без Expansion; NotSelectable — tview сам пропускает.
- О-3: заливка «сегодня» — colorTodayBg, НЕ colorAccent; баннер — свой текст, SetWrap(false).
- RD-7: спиннер — SetText, не setStatus (statusGen гасит таймер ошибки); errorShown стопает тик;
  Clear только в rebuildRows по свежему gen.
- RD-6: hint — dynamic, но текст только наш (formatHint), теги — hex ролей. RD-5: EWS-текст —
  только Sanitize + tview.Escape; a.event dynamic=false; превью — FormatMessageHeader.
- RD-4: цвет ячейки — в Style (Style.Decompose()); непрочитанные — accent+bold. RD-3: активная
  папка = открытая (ID), счётчик — mainFolders. RD-2: заголовок — panelHeader, линия textSecondary.
- RD-1: цвета — только роли theme.go; фокус/выделение — accent. UI-9: подсказки→a.hint,
  сообщения→a.status; endLoad один раз; красный — colorError.
- EWS: item:IsRead в FindItem / t:BodyType в GetItem → 500; read-only; tls в internal/ews.
