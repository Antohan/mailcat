# mailcat — HANDOFF

## Дальше
**V-38** (волна 6, `docs/PLAN-ITER3.1.md`): разделители ячеек сетки —
`calendar_grid.go` (`Draw`, внутренний прямоугольник `dayCellRect`), линии
`tview.Borders.Vertical/Horizontal` цветом `colorTextSecondary`; далее V-39…V-42.

## Сделано
- iter31 V-37 (7553acc): ячейки сетки одного размера, остаток — внешние поля.
- iter31 V-36 (efb0b8c): детали встречи — `FormatEventHeader` + описание GetItem с кэшем.
- iter31 V-35 (751857c): фокус/клавиши/подсказки календаря по панели.

## Открыто
- «ок» скриншотов: V-37 (`/tmp/v37_{before,after}_{100,220}.txt`), V-36, V-35.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-37 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-37: геометрия сетки — только `gridMargins`/`dayCellRect` (`cw=w/7`, `ch=(h-1)/6`, поля по центру); шапка — на `my`, недели — с `my+1`; остаток не отдавать последней колонке/неделе.
- V-36: кэш описания — по ItemId (`eventBodies`), `eventLoading` гасит двойной Enter; шапка и тело — один реестр ссылок (маркеры [N]); тело только через `formatBody` (Sanitize+Escape построчно); GetItem — §11, без `BodyType`, разбор `t:CalendarItem`.
- V-35: подсказки календаря — два набора по фокусу (`pageHintSegs`, GetFocus вне DrawFunc); `hintSegs` пишет только `showHint`; Enter на повестке — `SetSelectedFunc`, не capture; 1/2 — `ensureFocus` в сетку, фокус повестки сохраняется.
- V-34: повестка — `SetItemText` при смене курсора (не Clear); changed идёт до сдвига и на AddItem — зажим по `GetItemCount` в `restyleAgenda`; бар — колонка за рамкой.
- V-33/V-32: стили ячейки — только `cellStyles` (выходной→сегодня→выбор); счётчик — отдельный стиль, на акценте `colorSelectedFg`; сетка — только `monthGrid` (Box+Draw); смена выбора — только `selectCell`.
- Волна 5: панели — theme-роли, `colorSelectedBg/Fg`, фокус — только accent-рамка (V-22 отменён).
- V-29: разделители папок не выбираемы — `SetInputCapture`+`foldersMouse`; после `SetCurrentItem` — `applyActiveFolderStyle`.
