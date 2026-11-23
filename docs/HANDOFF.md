# mailcat — HANDOFF

## Дальше
**V-9** (`docs/PLAN-ITER3.1.md`): README (скриншоты/структура) + инвариант
«цвета и стили — только theme.go/UI-6»; затем DoD итерации 3.1.

## Сделано
- V-48 (d1aa4d8): разделители — полосы `colorHairline`, заливка вплотную;
  V-47 (261c341): `t`/`T` — сегодня.

## Открыто
- «ок» скриншотов: V-48 (`/private/tmp/v48_band/`: `b_d25/b_d16/b_220/b_dark.png`,
  `.txt/.tsv`), V-47, V-35…V-47.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3; V-25–V-48
  на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено:
  item:Importance, write, Catppuccin, Preview; скругления — tview только прямоугольник.

## Не трогать
- V-48: `gridMargins` 0,0; разделители — `drawGridBands` фоном `colorHairline`
  (не глифы: глиф центрируется и отрывает заливку); `paintSelFrame` — рамка у
  выбора; роль `colorGridLine` удалена, сетка — `colorHairline`.
- V-47: `t`/`T` — только `jumpToToday`; чужой месяц — `loadMonth`.
- V-46: `InMonth` — признак своего дня; счётчики только in-month.
- V-45: «сегодня» — accent+bold на `colorPanelBg`; V-44: текст +1,+1 при `ch>1`.
