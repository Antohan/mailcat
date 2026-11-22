# mailcat — HANDOFF

## Дальше
**V-36** (волна 5, `docs/PLAN-ITER3.1.md`): детали встречи — шапка
`FormatEventHeader`, описание GetItem Body с кэшем и тем же конвейером, что
письмо. Перед кодом — `EWS-REFERENCE.md §11` (ews-specialist). Далее V-9.

## Сделано
- iter31 V-35 (751857c): фокус/клавиши/подсказки календаря по панели.
- iter31 V-34 (2b5cd11): повестка — список в стиле писем, бар, пустой день.
- iter31 V-33 (367a751): палитра и типографика сетки — без скобок `[N]`.
- iter31 V-32 (4202cf3): сетка календаря заполняет панель — `monthGrid`.

## Открыто
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-35 на живом терминале; «ок» скриншотов V-35 (`/tmp/v35_before.txt`, `/tmp/v35_grid.txt`, `/tmp/v35_agenda.txt`) до старта V-36.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-35: подсказки календаря — два набора по фокусу (`pageHintSegs`, GetFocus вне DrawFunc); `hintSegs` пишет только `showHint`; Enter на повестке — `SetSelectedFunc`, не capture; 1/2 — `ensureFocus` в сетку, фокус повестки сохраняется.
- V-34: повестка — `SetItemText` при смене курсора (не Clear); changed идёт до сдвига и на AddItem — зажим по `GetItemCount` в `restyleAgenda`; бар — колонка за рамкой.
- V-33: стили ячейки — только `cellStyles` (порядок выходной→сегодня→выбор); счётчик — отдельный стиль, на акценте `colorSelectedFg`; `FormatDayCell` без скобок; шапка/счётчик — `colorTextSecondary`, только theme-роли (RD-1).
- V-32: сетка — только `monthGrid` (Box+Draw), не `tview.Table`; геометрия — чистые `dayCellRect`/`dayAt` на каждый Draw; смена выбора — только `selectCell`; текст ячейки — `gridText`, не `tview.Print`.
- Волна 5: панели — theme-роли, `colorSelectedBg/Fg`, фокус — только accent-рамка (V-22 отменён).
- V-29: разделители папок не выбираемы — `SetInputCapture`+`foldersMouse`; после `SetCurrentItem` — `applyActiveFolderStyle`.
- V-28/V-25: `formatBody` — Sanitize + tview.Escape ПОСТРОЧНО (SECURITY §3); `[ВНЕШН]` у каждой строки; имя отправителя — только у первого письма серии.
