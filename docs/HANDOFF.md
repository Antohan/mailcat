# mailcat — HANDOFF

## Дальше
**V-9** (`docs/PLAN-ITER3.1.md`): доки — README (скриншоты/структура), инвариант
в HANDOFF «цвета и стили — только через theme.go/константы UI-6»; policy_test
не задет (новых EWS-операций нет). Волна 7 закрыта.

## Сделано
- iter31 V-47 (261c341): `t`/`T` — сегодня, переход на текущий месяц, «t сегодня» в подсказках.
- iter31 V-46: 42 даты, соседние месяцы muted без счётчиков, выбор только in-month.
- iter31 V-45 (1828d04): «сегодня» — accent+bold на фоне панели, `colorTodayBg` удалён.
- iter31 V-44, V-43 (dc883bd): заливка до линий/отступ 1; линии `colorGridLine`.

## Открыто
- «ок» скриншотов: V-47 (`/private/tmp/v47_{before,after}_{light,dark}_{100x30,220x60}.txt` + `.ans`), V-35…V-46.
- TODO юзера (TESTING.md §5): smoke S3–S7; V-28 на живой почте; О-3 в Terminal.app/VS Code; V-25/V-26/V-29/V-31–V-47 на живом терминале.
- НИЗКО (security): attachments.go ~120 — частичный файл при Write; отложено: item:Importance, write, Catppuccin, item:Preview; скругления — tview только прямоугольник.

## Не трогать
- V-47: `t`/`T` — только `jumpToToday`; чужой месяц — `loadMonth`, выбор делает `render` (не дублировать `SelectDay`); «t сегодня» — opt-сегмент.
- V-46: `InMonth` — единственный признак своего дня (выбор/клик/`cellOf`/`maxDay`); `Day` у соседних — настоящий номер; счётчики только in-month; muted без bold.
- V-45: «сегодня» — accent+bold на `colorPanelBg`, счётчик `colorTextSecondary`; выбор — сплошной акцент и при совпадении побеждает.
- V-44: заливка — весь `dayCellRect` до линий (не под текстом); текст +1 вправо, +1 вниз при `ch>1`; счётчик вплотную после числа.
- V-43: линии — только `colorGridLine` (между hairline и textSecondary); пустые/соседние — те же линии.
