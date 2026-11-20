# mailcat — HANDOFF

## Дальше
Итерация 2 — `docs/PLAN-ITER2.md`: **UI-2** (колонки списка писем: отправитель —
`t:Name` с fallback на локальную часть адреса; дата «сегодня HH:MM / Чт /
ДД.ММ»; непрочитанные жирным — IsRead, см. «Не трогать»). После каждой задачи —
запись в CHANGELOG.md (формат в плане). Старт: `/stage UI-2`.

## Сделано (итерация 2)
- 9b1d232 — iter2:UI-1: сортировка папок, секция «Ещё ▸», `ui.hidden_folders`.

## Открыто (для пользователя; нужен VPN — TESTING.md §5)
- Smoke S1–S8; security-ревью этапов 5–6 — security-reviewer по
  `git diff 8fb6a9f..HEAD` (по желанию; было само-ревью, 8c9d501).
- `item:Importance` в EWS-REFERENCE? (нужно для min_importance и UI-3 ⚠).
- На живом сервере (UI-1): порядок папок, «Ещё ▸», hidden_folders.

## Не трогать
- item:IsRead в AdditionalProperties FindItem и t:BodyType в ItemShape GetItem
  — живой Exchange 2019 (15.2.2562) отвечает 500 ErrorInvalidRequest
  (EWS-REFERENCE §5–6, a1fe7b1 + a1d68ee).
- SortOrder/Restriction с CalendarView — запрет EWS, ответ сам упорядочен по
  Start (§8, d25f2bd).
- Read-only whitelist EWS-операций — золотое правило 1.
- crypto/tls только в internal/ews — policy_test, единая точка TLS.
- Sanitize + tview.Escape для Table/List/Modal (escapeForList) — «показанный
  URL = открываемому».
- Отбрасывание ответов по поколению (mail.go gen, calendar monthStale) —
  поздний ответ старой папки/месяца перезапишет список.
- Первый прогон watcher'а без уведомлений; тихий backoff до 5 мин.
- Панель папок строится только renderFolders (Clear + полный ребилд); гвард
  SetChangedFunc на «Ещё ▸/▾» — иначе пункт секции подменял бы folderName.

(История итерации 1 — в git: этапы 0–6 слиты в b53745f, пост-релиз d7b1d0d
a1fe7b1 d25f2bd 3033a9b.)
