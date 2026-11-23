# mailcat — HANDOFF

## Дальше
**V-37** (волна 6, `docs/PLAN-ITER3.1.md`): одинаковые ячейки сетки —
`calendar_grid.go` (`dayCellRect`/`dayAt`); далее V-38…V-42 по плану.

## Сделано
- iter31 V-36 (efb0b8c): детали встречи — `FormatEventHeader` + описание GetItem с кэшем.
- iter31 V-35 (751857c): фокус/клавиши/подсказки календаря по панели.
- iter31 V-34 (2b5cd11): повестка — список в стиле писем, бар, пустой день.

## Открыто
- «ок» скриншотов: V-35 (`/tmp/v35_*.txt`) и V-36 (`/tmp/v36_before.txt`, `/tmp/v36_after.txt`).
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-36 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-36: кэш описания — по ItemId (`eventBodies`), `eventLoading` гасит двойной Enter; шапка и тело — один реестр ссылок (маркеры [N]); тело только через `formatBody` (Sanitize+Escape построчно); GetItem — §11, без `BodyType`, разбор `t:CalendarItem`.
- V-35: подсказки календаря — два набора по фокусу (`pageHintSegs`, GetFocus вне DrawFunc); `hintSegs` пишет только `showHint`; Enter на повестке — `SetSelectedFunc`, не capture; 1/2 — `ensureFocus` в сетку, фокус повестки сохраняется.
- V-34: повестка — `SetItemText` при смене курсора (не Clear); changed идёт до сдвига и на AddItem — зажим по `GetItemCount` в `restyleAgenda`; бар — колонка за рамкой.
- V-33/V-32: стили ячейки — только `cellStyles` (выходной→сегодня→выбор); счётчик — отдельный стиль, на акценте `colorSelectedFg`; сетка — только `monthGrid` (Box+Draw), геометрия — чистые `dayCellRect`/`dayAt` на каждый Draw; смена выбора — только `selectCell`.
- Волна 5: панели — theme-роли, `colorSelectedBg/Fg`, фокус — только accent-рамка (V-22 отменён).
- V-29: разделители папок не выбираемы — `SetInputCapture`+`foldersMouse`; после `SetCurrentItem` — `applyActiveFolderStyle`.
- V-28/V-25: `formatBody` — Sanitize + tview.Escape ПОСТРОЧНО (SECURITY §3); `[ВНЕШН]` у каждой строки; имя отправителя — только у первого письма серии.
