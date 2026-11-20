# mailcat — HANDOFF

## Дальше
Итерация 2 — `docs/PLAN-ITER2.md`: **UI-4** (повестка: длинные URL — в реестр
ссылок, маркер `[1]`, модал `g` — переиспользовать showLinks/openLink; строка
«14:30–15:00 Название [1]»; статусы ✓/~/?/★; заголовок «Повестка: 18.11 —
3 встречи»; мёртвый FormatAgenda переписать и подключить). Флагман — /models.

## Сделано (итерация 2)
- b110fbe — iter2:UI-3: 📎 N вложений в FormatMessageHeader (⚠ нет — Importance нет в EWS-REFERENCE).
- 184bf25 — iter2:UI-2: имя отправителя, компактная дата, капы ширины колонок.
- f7cae6e/9b1d232 — iter2:UI-1: сортировка папок, секция «Ещё ▸», `ui.hidden_folders`.

## Открыто (для пользователя; нужен VPN — TESTING.md §5)
- Smoke S1–S8; UI-2 на живом сервере: имя/дата/жирные непрочитанные, дата при ~40%.
- `item:Importance` в EWS-REFERENCE? (нужно для min_importance и ⚠ карточки).
- Security-ревью этапов 5–6 — security-reviewer по `git diff 8fb6a9f..HEAD` (по желанию).

## Не трогать
- item:IsRead в FindItem AdditionalProperties и t:BodyType в GetItem ItemShape —
  живой Exchange 2019 (15.2.2562) отвечает 500 (EWS-REFERENCE §5–6).
- SortOrder/Restriction с CalendarView — запрет EWS (§8); ответ сам упорядочен по Start.
- Read-only whitelist EWS-операций (золотое правило 1); crypto/tls только в internal/ews.
- Sanitize + tview.Escape для Table/List/Modal (escapeForList) — «показанный URL = открываемому».
- Отбрасывание ответов по поколению (mail.go gen, calendar monthStale).
- Первый прогон watcher'а без уведомлений; тихий backoff до 5 мин.
- Панель папок — только renderFolders (Clear + ребилд); гвард SetChangedFunc на «Ещё ▸/▾».
- Капы ширины колонок списка — только clampMessageColumns (SetDrawFunc, до раскладки tview).
- Отправитель в списке — имя (t:Name) с fallback на локальную часть; полный адрес — в карточке (§4.1).

(История итерации 1 — в git: этапы 0–6 слиты в b53745f, пост-релиз d7b1d0d
a1fe7b1 d25f2bd 3033a9b.)
