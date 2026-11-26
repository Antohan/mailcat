#!/usr/bin/env python3
"""Отчёт об использовании токенов по локальной базе opencode (только чтение).

Три главные цифры: сессий, оборванных лимитом шагов; сессий на задачу;
цена задачи. Период — с последнего релиза (последний коммит main) или
с --since ГГГГ-ММ-ДД. Нет базы или изменилась схема — «Отчёт недоступен»
и код 0: отчёт не блокирует релиз.
"""
import argparse
import json
import os
import re
import sqlite3
import statistics as st
import subprocess
import sys
from datetime import datetime

DB = os.path.expanduser("~/.local/share/opencode/opencode.db")
AGENT_FILE = ".opencode/agents/go-implementer.md"
AGENT = os.path.basename(AGENT_FILE)[:-3]
# Формулировки агента об остановке по лимиту шагов (из ответов реальных сессий).
LIMIT_HINT = re.compile(
    r"лимит\w* шагов|шагов исчерпан|максимум шагов|max(imum)? steps|steps? (limit|reached|exhausted)",
    re.I)
TASK_ID = re.compile(r"\b([A-Z]{1,5}-\d+)\b")
# Пороги внимания: доля оборванных сессий агента и сессий на задачу.
STOP_PCT, TASK_RATIO = 10.0, 1.5


def unavailable(msg):
    print(f"Отчёт недоступен: {msg}")
    sys.exit(0)


def git(*args):
    return subprocess.run(["git", *args], capture_output=True, text=True).stdout.strip()


def med(xs):
    return st.median(xs) if xs else 0.0


def load(con, since):
    rows = con.execute(
        """SELECT id, agent, title, cost, model FROM session_v2
           WHERE directory = ? AND time_created >= ?""",
        (os.path.realpath("."), since)).fetchall()
    out = [dict(r) for r in rows]
    try:  # шаги и контекст — из сообщений; нет таблицы — считаем без них
        for d in out:
            msgs = con.execute(
                "SELECT data FROM session_message WHERE session_id = ? AND type = 'assistant' ORDER BY seq",
                (d["id"],)).fetchall()
            d["steps"] = len(msgs)
            d["asks"] = con.execute(
                "SELECT COUNT(*) FROM session_message WHERE session_id = ? AND type = 'user'",
                (d["id"],)).fetchone()[0]
            d["limit_stop"] = bool(msgs and LIMIT_HINT.search(msgs[-1]["data"] or ""))
            d["ctx"] = []
            for m in msgs:
                try:
                    t = json.loads(m["data"]).get("tokens") or {}
                except ValueError:
                    continue
                c = t.get("cache") or {}
                d["ctx"].append(t.get("input", 0) + c.get("read", 0) + c.get("write", 0))
    except sqlite3.Error:
        for d in out:
            d["steps"] = d["asks"] = d["limit_stop"] = None
            d["ctx"] = []
    for d in out:
        try:
            m = json.loads(d["model"] or "{}")
            d["model"] = f"{m.get('providerID')}/{m.get('id')}"
        except ValueError:
            d["model"] = "?"
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--since", help="ГГГГ-ММ-ДД — вместо последнего релиза")
    ap.add_argument("--limit", type=int, help="лимит шагов — вместо значения из конфига агента")
    a = ap.parse_args()
    if a.since:
        try:
            since = datetime.strptime(a.since, "%Y-%m-%d").timestamp() * 1000
        except ValueError:
            unavailable("--since ждёт ГГГГ-ММ-ДД")
    else:
        ts = git("log", "main", "-1", "--format=%ct")
        since = int(ts) * 1000 if ts.isdigit() else 0
    if not os.path.exists(DB):
        unavailable(f"нет базы {DB}")

    limit = a.limit
    if limit is None:
        try:
            m = re.search(r"^steps:\s*(\d+)", open(AGENT_FILE, encoding="utf-8").read(), re.M)
            limit = int(m.group(1)) if m else None
        except OSError:
            limit = None

    con = sqlite3.connect(f"file:{DB}?mode=ro", uri=True)
    con.row_factory = sqlite3.Row
    try:
        rows = load(con, since)
    except sqlite3.Error as e:
        unavailable(f"схема базы opencode изменилась ({e})")

    since_txt = datetime.fromtimestamp(since / 1000).strftime("%d.%m %H:%M") if since else "начала"
    cost = sum(r["cost"] for r in rows)
    print(f"Отчёт по токенам: с {since_txt} · сессий {len(rows)} · ${cost:.2f}")
    by_model = {}
    for r in rows:
        n, c = by_model.get(r["model"], (0, 0.0))
        by_model[r["model"]] = (n + 1, c + r["cost"])
    for name, (n, c) in sorted(by_model.items(), key=lambda kv: -kv[1][1]):
        print(f"  {name}: {n} сесс., ${c:.2f}")
    if not rows:
        return

    ag = [r for r in rows if r["agent"] == AGENT and r["steps"] is not None]
    if ag:
        confirmed = [r for r in ag if r["limit_stop"]]
        at_limit = [r for r in ag if limit and r["steps"] == limit and r["asks"] == 1]
        stop_pct = len(confirmed) / len(ag) * 100
        print(f"1. Обрывы по лимиту шагов ({limit if limit is not None else 'не задан'}): "
              f"подтверждено {len(confirmed)} из {len(ag)} сессий агента ({stop_pct:.0f}%); "
              f"ровно на лимите с одним запросом — {len(at_limit)}")
        if stop_pct >= STOP_PCT:
            print(f"   ВНИМАНИЕ: лимит шагов останавливает {stop_pct:.0f}% сессий — поднимите steps в {AGENT_FILE}")

    tasks = {}
    for r in rows:
        m = TASK_ID.search(r["title"] or "")
        if m:
            tasks.setdefault(m.group(1), []).append(r)
    n_task_sessions = sum(len(v) for v in tasks.values())
    ratio = n_task_sessions / len(tasks) if tasks else 0.0
    per_task = (sum(r["cost"] for v in tasks.values() for r in v) / len(tasks)) if tasks else 0.0
    print(f"2. Сессий на задачу: {ratio:.1f} (задач {len(tasks)}, сессий {n_task_sessions} "
          f"с ID в названии из {len(rows)})")
    print(f"3. Цена задачи: ${per_task:.3f}")
    if tasks and ratio >= TASK_RATIO:
        print(f"   ВНИМАНИЕ: {ratio:.1f} сессии на задачу — задачи обрываются или слишком велики")

    grows = [(r["ctx"][-1] - r["ctx"][0]) / (len(r["ctx"]) - 1) for r in rows if len(r["ctx"]) >= 2]
    starts = [r["ctx"][0] for r in rows if len(r["ctx"]) >= 2]
    if starts:
        print(f"   контекст сессии: старт {med(starts) / 1e3:.1f} тыс. токенов, "
              f"+{med(grows) / 1e3:.2f} тыс. на шаг")


if __name__ == "__main__":
    main()
