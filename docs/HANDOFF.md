# mailcat — HANDOFF

## Дальше
Итерация 2 — `docs/PLAN-ITER2.md`: **UI-3** (карточка письма: 📎 N вложений
и ⚠ при high-importance в `FormatMessageHeader`; Importance — открытый вопрос,
без него ⚠ не делать). После задачи — запись в CHANGELOG.md. Старт: `/stage UI-3`.

## Сделано (итерация 2)
- 184bf25 — iter2:UI-2: имя отправителя, компактная дата, капы ширины колонок.
- f7cae6e/9b1d232 — iter2:UI-1: сортировка папок, секция «Ещё ▸», `ui.hidden_folders`.

## Открыто (для пользователя; нужен VPN — TESTING.md §5)
- Smoke S1–S8; UI-2 на живом сервере: имя/дата/жирные непрочитанные
  (IsRead идёт с Default-shape — инвариант ниже), дата видна при панели ~40%.
- `item:Importance` в EWS-REFERENCE? (нужно для min_importance и UI-3 ⚠).
- Security-ревью этапов 5–6 — security-reviewer по `git diff 8fb6a9f..HEAD`
  (по желанию; было само-ревью, 8c9d501).

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
- Список писем: капы ширины колонок только в clampMessageColumns
  (SetDrawFunc, до раскладки tview) — tview выбрасывает колонки, не влезающие
  целиком; без капа длинная тема выталкивает дату за край (184bf25).
- В списке писем отправитель — имя (t:Name) с fallback на локальную часть;
  полный адрес — в заголовке просмотра/карточке (SECURITY.md §4.1, 184bf25).

(История итерации 1 — в git: этапы 0–6 слиты в b53745f, пост-релиз d7b1d0d
a1fe7b1 d25f2bd 3033a9b.)
