# mailcat — HANDOFF

## Дальше
Итерация 2 — `docs/PLAN-ITER2.md`: задачи UI-1…UI-8 + T-1 (техдолг) + D-1
(доки). Обязательно после каждой задачи — запись в CHANGELOG.md (формат —
в плане, «Журнал решений»; по нему пользователь проверяет ход работ).
Старт: `/stage UI-1`.

## Сделано (итерация 1, этапы 0–6; слито в main: b53745f)
0–6: 79881c3 · 889a3cd f123125 f97477a 5b32e3e 10fb8a7 8fb6a9f · 54d6df0
0012b0d · 3bbff28 2507572 31b956e 309332b · 8f1a1f0 1026960 7a160c3 47b3c52
002a7e1 · 8c9d501 (само-ревью: проброс логгера). Suite зелёная, policy_test.
Пост-релиз на живом сервере: d7b1d0d светлая тема · a1fe7b1 FindItem 500 ·
d25f2bd CalendarView · 3033a9b org_domains.

## Открыто (для пользователя; нужен VPN — TESTING.md §5)
- Smoke S1–S8; формальное security-ревью этапов 5–6 — security-reviewer по
  `git diff 8fb6a9f..HEAD` (по желанию; было само-ревью, 8c9d501).
- Вопрос: добавлять ли `item:Importance` в EWS-REFERENCE (нужно для
  min_importance уведомлений и UI-3 ⚠; сейчас важность = normal —
  notify/rules.go). Техдолг из старого «Открыто» перенесён в T-1 (ITER2).

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
