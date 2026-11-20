# mailcat — HANDOFF

## Дальше
Итерация 2 — `docs/PLAN-ITER2.md`: **UI-7** (контекстный статус-бар: подсказки
по экрану — почта/календарь свои наборы клавиш, см. спеку; ошибки — красной
строкой `showError` с автоочисткой по таймеру; вход — app.go `setStatus`/
`showError`, статус-бар `a.status`).

## Сделано (итерация 2)
- e015c2f — iter2:UI-6: рамка активной панели акцентом (бирюза, ≠ синего
  выделения) — before-draw хук paintFocusFrames, 5 панелей почты/календаря.
- c4b4140 — iter2:UI-5: сетка месяца — заголовок «Ноябрь 2026», «сегодня» [скобки+жирный] ≠ выбранный (инверсия), «17·2» — число событий.
- a5ca69d — iter2:UI-4: повестка — маркеры ✓/~/?/★/✗, URL «Места» в реестр (модал g), заголовок «N встреч».
- b110fbe — iter2:UI-3: 📎 N вложений в FormatMessageHeader (⚠ нет — Importance нет в EWS-REFERENCE).
- 184bf25 — iter2:UI-2: имя отправителя, компактная дата, капы ширины колонок.
- f7cae6e/9b1d232 — iter2:UI-1: сортировка папок, секция «Ещё ▸», `ui.hidden_folders`.

## Открыто (для пользователя; нужен VPN — TESTING.md §5)
- Smoke S1–S8; UI-2/UI-4/UI-6 на живом сервере: имя/дата/жирные непрочитанные;
  маркеры повестки и модал g по реальным Zoom/Teams-ссылкам; акцентная рамка
  при Tab/←→/клике мыши.
- `item:Importance` в EWS-REFERENCE? (нужно для min_importance и ⚠ карточки).
- Security-ревью этапов 5–6 — security-reviewer по `git diff 8fb6a9f..HEAD` (по желанию).

## Не трогать
- Рамка активной панели — только paintFocusFrames (before-draw): фокус меняют
  Tab/←→/мышь/модалы, хук видит их все; GetFocus внутри хука — deadlock.
  Акцент — бирюза, не синий: «активная панель» ≠ «выбранная строка».
- Текст ячейки сетки месяца — только FormatDayCell; маркер «сегодня» —
  текстовые скобки, не цвет: переживают инверсию выделения и не спорят с
  красным Сб/Вс и серым соседних месяцев.
- item:IsRead в FindItem AdditionalProperties и t:BodyType в GetItem ItemShape —
  живой Exchange 2019 (15.2.2562) отвечает 500 (EWS-REFERENCE §5–6).
- SortOrder/Restriction с CalendarView — запрет EWS (§8); ответ сам упорядочен по Start.
- Read-only whitelist EWS-операций (золотое правило 1); crypto/tls только в internal/ews.
- Sanitize + tview.Escape для Table/List/Modal (escapeForList) — «показанный URL = открываемому».
- Строки повестки — только FormatAgenda (escapeForList целиком, маркер [N] в
  экранированной форме); почему: List интерпретирует теги [red] безусловно.
- showLinks(links, back): back — колбэк «страница + фокус» из места вызова;
  Esc из модала ссылок возвращается к источнику (письмо/встреча/календарь).
- Отбрасывание ответов по поколению (mail.go gen, calendar monthStale).
- Первый прогон watcher'а без уведомлений; тихий backoff до 5 мин.
- Панель папок — только renderFolders (Clear + ребилд); гвард SetChangedFunc на «Ещё ▸/▾».
- Капы ширины колонок списка — только clampMessageColumns (SetDrawFunc, до раскладки tview).
- Отправитель в списке — имя (t:Name) с fallback на локальную часть; полный адрес — в карточке (§4.1).

(История итерации 1 — в git: этапы 0–6 слиты в b53745f, пост-релиз d7b1d0d
a1fe7b1 d25f2bd 3033a9b.)
