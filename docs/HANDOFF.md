# mailcat — HANDOFF

## Дальше
Итерация 2 — `docs/PLAN-ITER2.md`: **D-1** (доки: SECURITY.md §4 — формулировка
«домен ≠ org_domain» → список `org_domains`, изменение 3033a9b). После D-1 —
DoD итерации; merge в main — только по явной просьбе пользователя.

## Сделано (итерация 2)
- c28341f — iter2:T-1: техдолг — reAnchor data-href/некотируемые; CreateUnique
  O_EXCL; каталоги вложений при старте; closeModals при 1/2 (Esc-источник).
- ed80438 — iter2:UI-8: индикатор «Загрузка…» (loads, beginLoad/endLoad).
- 86cc38c — iter2:UI-7: подсказки по странице, красная ошибка, автоочистка.
- e015c2f — iter2:UI-6: рамка активной панели акцентом (paintFocusFrames).
- c4b4140 — iter2:UI-5: сетка месяца — заголовок, «сегодня» ≠ выбранный, «17·2».
- a5ca69d — iter2:UI-4: повестка — маркеры ✓/~/?/★/✗, URL в реестр, модал g.
- b110fbe — iter2:UI-3: 📎 N вложений. 184bf25 — UI-2: имя/дата/капы колонок.
  f7cae6e/9b1d232 — UI-1: сортировка папок, «Ещё ▸», ui.hidden_folders.

## Открыто (для пользователя; нужен VPN — TESTING.md §5)
- Smoke S1–S8 и UI-2..UI-8/T-1 на живом сервере; `item:Importance` в
  EWS-REFERENCE? (нужно для min_importance и ⚠); security-ревью
  `git diff 8fb6a9f..HEAD` — по желанию.

## Не трогать
- Индикатор загрузки: реестр `loads` + гейтинг в showHint; каждая ветка
  горутины зовёт endLoad(id) в QueueUpdateDraw ровно один раз; доступ —
  только главный поток TUI; GetAttachment не участвует (статусы терминальные).
- Статус-бар: красный — только SetTextColor(colorError), без dynamic colors;
  автоочистка — по поколению statusGen; подсказка — только pageHint().
- Рамка фокуса — только paintFocusFrames (before-draw; GetFocus в хуке —
  deadlock); текст ячейки сетки — только FormatDayCell.
- Модалы: showLinks(links, back) — Esc к источнику (страница+фокус); при 1/2
  closeModals снимает links/attachments/confirm и переводит фокус — скрытый
  SwitchToPage'ом модал держит фокус, его Esc уводил на покинутый источник.
- Вложения: каталоги — только prepareAttachmentDirs при старте (attDirErr —
  fail-fast до скачивания); файл — только CreateUnique (O_EXCL, 0600):
  вернуть «путь» и писать WriteFile — гонка вернётся.
- Ссылки: reAnchor — href только после пробела (data-href — трекер), значение
  тремя альтернативами кавычек; «показанный URL = открываемому»;
  Sanitize + tview.Escape (escapeForList); строки повестки — FormatAgenda.
- EWS: item:IsRead в FindItem и t:BodyType в GetItem — живой Exchange даёт
  500; SortOrder/Restriction с CalendarView — запрет; read-only whitelist;
  crypto/tls только в internal/ews.
- Отбрасывание ответов по поколению (mail.go gen, calendar monthStale);
  первый прогон watcher без уведомлений, тихий backoff до 5 мин.
- Папки — только renderFolders, гвард на «Ещё ▸/▾»; капы колонок —
  clampMessageColumns; отправитель — t:Name с fallback на локальную часть.

(История итерации 1 — в git: этапы 0–6 слиты в b53745f, пост-релиз d7b1d0d
a1fe7b1 d25f2bd 3033a9b.)
