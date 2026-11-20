# mailcat — HANDOFF

## Дальше
Итерация 2 — `docs/PLAN-ITER2.md`: **T-1** (техдолг: reAnchor data-href и
некотируемые атрибуты — `internal/mail/links.go` reAnchor; UniquePath через
O_EXCL вместо «проверил-записал» — `internal/security/filesafe.go` UniquePath;
каталоги вложений при старте — `internal/ui/attachments.go` saveAttachment;
Esc-источник модала — `internal/ui/links.go` showLinks/back).

## Сделано (итерация 2)
- ed80438 — iter2:UI-8: индикатор «Загрузка…» на время GetItem/FindItem
  (beginLoad/endLoad, реестр in-flight), переживает смену страницы и
  автоочистку ошибки.
- 86cc38c — iter2:UI-7: подсказки статус-бара по странице (pageHint/showHint),
  ошибка красным с автоочисткой 8 с (statusGen).
- e015c2f — iter2:UI-6: рамка активной панели акцентом (paintFocusFrames).
- c4b4140 — iter2:UI-5: сетка месяца — заголовок, «сегодня» ≠ выбранный, «17·2».
- a5ca69d — iter2:UI-4: повестка — маркеры ✓/~/?/★/✗, URL в реестр, модал g.
- b110fbe — iter2:UI-3: 📎 N вложений в FormatMessageHeader.
- 184bf25 — iter2:UI-2: имя отправителя, компактная дата, капы ширины колонок.
- f7cae6e/9b1d232 — iter2:UI-1: сортировка папок, секция «Ещё ▸», ui.hidden_folders.

## Открыто (для пользователя; нужен VPN — TESTING.md §5)
- Smoke S1–S8; UI-2/UI-4/UI-6/UI-7/UI-8 на живом сервере: имя/дата/жирные
  непрочитанные; маркеры повестки и модал g; акцентная рамка; подсказки
  статус-бара, красная ошибка и её автоочистка (выключить VPN); «Загрузка…»
  держится при смене страницы 1/2 во время медленного запроса.
- `item:Importance` в EWS-REFERENCE? (нужно для min_importance и ⚠ карточки).
- Security-ревью этапов 5–6 — security-reviewer по `git diff 8fb6a9f..HEAD` (по желанию).

## Не трогать
- Индикатор загрузки — реестр `loads` + гейтинг в showHint (UI-8): не гасить
  «Загрузка…» в точках смены страниц; каждая ветка горутины (успех/ошибка/
  устаревший ответ) зовёт endLoad(id) ровно один раз внутри QueueUpdateDraw;
  доступ к loads — только главный поток TUI, без мьютекса. GetAttachment
  (вложения) в реестре не участвует — его статусы терминальные, не подсказка.
- Статус-бар без dynamic colors: красный — только SetTextColor(colorError),
  обычный — SetTextColor(PrimaryTextColor) в setStatus. Тег [red] включил бы
  интерпретацию серверного текста — запрещено (§3).
- Автоочистка ошибки — по поколению statusGen (showError/setStatus меняют,
  таймер зовёт clearError(gen)); не проверять время — статус мог смениться.
- Подсказка — только pageHint() по GetFrontPage после смены страницы/загрузки;
  не хардкодить строки в точках вызова.
- Рамка активной панели — только paintFocusFrames (before-draw); GetFocus
  внутри хука — deadlock. Акцент — бирюза ≠ синее выделение строки.
- Текст ячейки сетки месяца — только FormatDayCell; маркер «сегодня» — скобки.
- item:IsRead в FindItem AdditionalProperties и t:BodyType в GetItem ItemShape —
  живой Exchange 2019 отвечает 500 (EWS-REFERENCE §5–6).
- SortOrder/Restriction с CalendarView — запрет EWS (§8).
- Read-only whitelist EWS-операций; crypto/tls только в internal/ews.
- Sanitize + tview.Escape для Table/List/Modal (escapeForList) — «показанный
  URL = открываемому»; строки повестки — только FormatAgenda.
- showLinks(links, back): back — «страница + фокус», Esc идёт назад к источнику.
- Отбрасывание ответов по поколению (mail.go gen, calendar monthStale).
- Первый прогон watcher'а без уведомлений; тихий backoff до 5 мин.
- Панель папок — только renderFolders; гвард SetChangedFunc на «Ещё ▸/▾».
- Капы ширины колонок — только clampMessageColumns (SetDrawFunc).
- Отправитель в списке — имя (t:Name) с fallback на локальную часть (§4.1).

(История итерации 1 — в git: этапы 0–6 слиты в b53745f, пост-релиз d7b1d0d
a1fe7b1 d25f2bd 3033a9b.)
