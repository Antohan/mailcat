# mailcat — HANDOFF

## Дальше
Итерация 3.1, волна 4 закрыта. Следующий — **V-8** (календарь: убрать
ASCII-баннер, сетка на панель, Enter → детали, описание встречи). EWS: сначала
спека GetItem для CalendarItem в `EWS-REFERENCE.md §11`, код — после.
Читать `calendar.go`, `calendar_format.go`.

## Сделано
- iter31 V-27 (a1d2d85): →/l/h не прокручивают текст панели чтения по горизонтали.
- iter31 V-26 (176068c): счётчик папок у правого края, длинное имя усекается «…».

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; О-3 в Terminal.app/VS Code; V-25/V-26 на живом терминале и живой почте.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-27: `wrappedLines` считает строки `tview.WordWrap`, НЕ `GetWrappedLineCount` (тот портит индекс TextView: `t.width==0`); →/l/h поглощает только `SetInputCapture` панели чтения и `a.detail`; ← — «к списку» (V-15), `SetScrollable` и полоса прокрутки V-17 не трогаются; `h/l` поглощены и в черновике — тот же TextView.
- V-26: ширина панели папок — только из DrawFunc (`setFoldersWidth`, один `SetItemText`, без Clear/GetFocus); вид строки — `folderRowText` для обоих путей.
- V-25: имя отправителя — только у первого письма серии (`sameSenderRun`: ящик + группа дня); [ВНЕШН] у каждой строки (SECURITY §4).
- V-24/V-23/V-21: `hintFolders` — все сегменты обязательные; панели переключает только cycleFocus (он же зовёт showHint); кэш `a.hintSegs` пишет лишь showHint (V-17); старт — фокус `messages`.
- Выделение — всегда `colorSelectedBg`/`colorSelectedFg`; фокус показывает только accent-рамка (V-22 отменён).
- V-20/V-19/V-18: folderName/accent — только selectFolder; renderFolders помнит папку/«Ещё»; watcher — только refreshFolderRow.
- V-17: пункт — две строки, вторая — paintItemTail; поля — listPadLeft/Right; в Table нет «│»; подсказки — opt снимает formatHintFit, в DrawFunc НЕ звать GetFocus; Enter — readMessage.
