# How this server is put together 🧩

A simple guide to what each folder does and how they work together.

---

## The big picture

Think of the app like a **restaurant** 🍽️

| Restaurant | Our code | Job |
| --- | --- | --- |
| Front door | `main.go` | Where everyone enters |
| Manager | `cmd/` | Opens the restaurant, tells everyone to start |
| Recipe book | `types/` | The shared "shapes" everyone agrees on |
| Checklist before opening | `config/` | Reads the settings, checks nothing is missing |
| Phone line to the kitchen | `database/` | Connects to the database |
| Floor plan / table map | `router/` | Decides which request goes where |
| Waiters | `handler/` | Actually do the work for each request |

---

## The folder map

```
server/
│
├── main.go            👈 START HERE — just says "go!"
│
├── cmd/
│   └── server.go      🧑‍💼 The boss: sets everything up in order
│
├── config/
│   └── config.go      📋 Reads your .env settings (and complains if one is missing)
│
├── types/             📚 Shared "shapes" — no logic, just definitions
│   ├── config.go         what a Config looks like
│   └── db.go             a nickname for the database pool
│
├── database/
│   └── database.go    🔌 Opens the connection to Postgres
│
├── router/
│   └── router.go      🗺️  The map: "/health goes to the Health waiter"
│
├── handler/
│   └── health.go      🙋 The waiter that answers "/health"
│
└── .env               🔑 Your local settings (passwords, ports…)
```

---

## How a startup happens (top to bottom)

```
  make run
     │
     ▼
  main.go  ──────────────►  "Hey cmd, take over!"
     │
     ▼
  cmd/server.go  (the boss does 4 things, in order)
     │
     ├─ 1️⃣  config.Load()        📋  read .env  →  settings
     │                               (stops here if DB_PASSWORD etc. is missing)
     │
     ├─ 2️⃣  database.New(...)     🔌  open the database connection
     │
     ├─ 3️⃣  router.New(db)        🗺️  build the server + its routes
     │
     └─ 4️⃣  e.Start(":8080")      🚀  start listening for requests
```

---

## What happens when someone visits `/health`

```
  🌐 Browser                                          🗄️ Database
      │                                                    ▲
      │  GET /health                                       │ ping?
      ▼                                                    │
  router 🗺️  ──"that's the Health route"──►  handler 🙋 ───┘
                                                 │
                                    ┌────────────┴────────────┐
                                    ▼                         ▼
                            DB answers ✅              DB silent ❌
                        200 {"status":"ok"}      503 {"status":"down"}
                                    │                         │
                                    └────────────┬────────────┘
                                                 ▼
                                          🌐 Browser gets reply
```

---

## Who is allowed to talk to whom

The arrows only point **downward**. Nothing low ever reaches back up.
This is what keeps the code from getting tangled.

```
            main
              │
              ▼
             cmd
        ┌─────┼───────────┐
        ▼     ▼           ▼
    config  database   router
        │     │           │
        │     │           ▼
        │     │        handler
        │     │           │
        └─────┴─────┬─────┘
                    ▼
                  types     ◄── everyone uses it, it uses no one
```

👉 **Rule of thumb:** `types` is the dictionary everybody reads from.
Because nobody points *back up*, you never get stuck in a loop.

---

## The one-line job of each folder

| Folder | In plain words |
| --- | --- |
| **`main`** | "Start the app." That's literally all. |
| **`cmd`** | The boss that plugs all the pieces together in the right order. |
| **`config`** | Reads your `.env` and yells early if something important is missing. |
| **`types`** | The shared shapes (`Config`, `DBPool`). Just definitions, no actions. |
| **`database`** | Opens and tests the Postgres connection. |
| **`router`** | The map: which URL goes to which handler. |
| **`handler`** | The workers that actually answer each request. |

---

## Why split it like this? 🤔

- **Easy to find things** — need the health logic? It's in `handler/`. Need settings? `config/`.
- **Easy to change one thing** — swap the database later and only `database/` + `types/` care.
- **Fails early & loudly** — if `.env` is missing a password, `config` stops *now* with a clear message, instead of a confusing crash later.
- **`main` stays tiny** — the real setup lives in `cmd`, which is easier to read and test.
