# mailcat — HANDOFF

## Сделано (этапы 0–6, ветка dev)
79881c3 (0+1) · 889a3cd f123125 f97477a 5b32e3e 10fb8a7 8fb6a9f (3) ·
54d6df0 0012b0d (4) · 3bbff28 2507572 31b956e 309332b (5) · 8f1a1f0 1026960
7a160c3 47b3c52 002a7e1 (6). Suite: 8 пакетов зелёные, policy_test включён.

## Дальше
Этапов в PLAN.md больше нет. Следующая сессия: итоговое security-ревью
этапов 5–6 (оно ещё не проводилось) → fixes → «Rulings»-строки в инварианты.
Читать: docs/SECURITY.md (чек-лист), `git log --oneline 309332b..HEAD`.

## Открыто (для пользователя, нужен VPN — TESTING.md §5)
- S1 doctor (негативные: VPN off / кривой ca_path / кривой пароль)
- S2 folders · S3 чтение+пагинация · S4 вложения (PDF, .exe, лимит)
- S5 ссылки (⚠, IDN, модал g) · S6 календарь (повторы, MyResponseType)
- S7 разрыв VPN → u · S8 уведомления (звук, фильтры, show_subject=false,
  enabled=false — тишина)
- Вопрос: добавить ли item:Importance в EWS-REFERENCE (для min_importance;
  сейчас важность считается normal — см. notify/rules.go).
- Deferred minors (по согласованию): reAnchor data-href/unquoted;
  UniquePath O_EXCL; каталоги вложений при старте; Esc-модала источник;
  FormatAgenda — мёртвый код (удалить/подключить).

## Не трогать
- Read-only whitelist EWS-операций — зачем: золотое правило 1.
- crypto/tls только в internal/ews — зачем: policy_test, единая точка TLS.
- Sanitize + tview.Escape для Table/List/Modal (escapeForList) — зачем:
  теги интерпретируются безусловно; «показанный URL = открываемому».
- Отбрасывание ответов по поколению (mail.go gen, calendar monthStale) —
  зачем: поздний ответ старой папки/месяца перезапишет список.
- Первый прогон watcher'а без уведомлений; тихий backoff до 5 мин.
