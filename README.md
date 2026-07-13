# Xeneon Claude Code Companion

Turn a **CORSAIR Xeneon Edge** touchscreen into a live monitor for your **Claude Code**
sessions — 5‑hour and weekly rate‑limit usage, context‑window fill, session cost, and
elapsed time, updating in real time while you work in a plain terminal.

The package has two parts:

| Part | What it is | Where |
|---|---|---|
| **`xeneon-bridge`** | A single Go binary. Runs a tiny HTTP server on `127.0.0.1`, receives usage from Claude Code's statusline, and serves it to the widget. Runs portable (`serve`) or as a Windows service. | [`companion/`](companion/) |
| **Claude Usage widget** | An iCUE HTML widget for the Xeneon Edge. Polls the bridge and draws the display. The bridge hooks in through Claude Code's statusline. | [`claudeusage/`](claudeusage/) |

---

## Screenshots

The **medium** (840×696) layout — the full stack of 5‑hour and weekly limits, context window, and
session cost — shown with a single session and with the multi‑session switcher (`‹ folder n/m ›`;
tap the arrows or swipe to flip between sessions):

| One session | Multiple sessions |
|:---:|:---:|
| ![Medium layout, single session](screenshots/medium-one-session.png) | ![Medium layout with the multi-session switcher](screenshots/medium-multi-session.png) |

The **small** (840×344) layout rearranges the same data into a wide strip:

![Small layout, single session](screenshots/small-one-session.png)

---

## Prerequisites

- **Windows 10/11** (the bridge's `serve` mode is cross‑platform, but the Windows *service* commands and the widget target Windows).
- **[Go](https://go.dev/dl/) 1.26+** — to build the bridge.
- **[iCUE](https://www.corsair.com/icue) 5.44 or later** — to install the widget.
- **iCUE Widget CLI** (`icuewidget`) — to package the widget. (Only needed if you want to build the `.icuewidget` yourself)
- A **CORSAIR Xeneon Edge** to display on.
- **Claude Code**, signed in on a Pro/Max plan.

---

## Repository layout

```
.
├── companion/        Go module `xeneoncc` — the xeneon-bridge binary
│   ├── main.go               command dispatch: serve | service | statusline | hook
│   └── internal/…            server, config, store, Claude Code adapters
├── claudeusage/      The iCUE widget
│   ├── index.html           markup + CSS + polling/render logic
│   ├── manifest.json        widget metadata (interactive: true)
│   ├── translation.json
│   └── resources/icon.svg
└── docs/, references/, skill.md, evals/
                      CORSAIR iCUE widget-SDK reference material used while building.
                      Not part of the shippable package — you can leave these out of a
                      public repo if you only want to publish the bridge + widget.
```

---

## Part 1 — Build the bridge

From the `companion/` directory:

```powershell
cd companion
go build -o xeneon-bridge.exe .
```

That produces a single self‑contained `xeneon-bridge.exe`. The one dependency
(`golang.org/x/sys`, pinned in `go.mod`) is fetched automatically on first build, so the
initial build needs network access; after that it builds offline.

Put the binary somewhere stable that you won't move — e.g. `C:\Tools\xeneon-bridge\` — because
Claude Code's config and (optionally) the Windows service will reference it by path.

Check it runs:

```powershell
.\xeneon-bridge.exe
# usage: xeneon-bridge <serve|service|statusline|notify|hook>
```

---

## Part 2 — Run the bridge

You have two ways to run it. **Portable mode** is simplest for trying it out; **service mode**
is best for a permanent, always‑on setup.

### Option A — Portable mode (`serve`)

Runs in the foreground until you press **Ctrl‑C**:

```powershell
.\xeneon-bridge.exe serve
# listening on http://127.0.0.1:8787
# xeneon-bridge ready on port 8787 (token in C:\Users\<you>\.xeneon-bridge\config.json)
```

On first run it creates a config file with a **random token** and port `8787`:

```
%USERPROFILE%\.xeneon-bridge\config.json
```
```json
{ "port": 8787, "token": "…64-hex-chars…" }
```

You'll paste that `token` into the widget later.

**To keep it running after you close the terminal** (without a service), drop a shortcut to
`xeneon-bridge.exe serve` into your Startup folder (`shell:startup`), or create a Task Scheduler
task "at log on." For a hands‑off, boots‑before‑login setup, use service mode instead.

### Option B — Windows service mode

Installs `xeneon-bridge` as a proper Windows service (**`XeneonBridge`**, LocalSystem, automatic
start), so it starts on boot and runs in the background. **Run these from an elevated
(Administrator) PowerShell.**

```powershell
# Install (creates the service + machine-wide config, prints the token)
.\xeneon-bridge.exe service install

# Control
.\xeneon-bridge.exe service start
.\xeneon-bridge.exe service status
.\xeneon-bridge.exe service stop

# Remove (stops first, then deletes). Add --purge to also delete the config dir.
.\xeneon-bridge.exe service uninstall
.\xeneon-bridge.exe service uninstall --purge
```

In service mode the config lives machine‑wide at:

```
C:\ProgramData\xeneon-bridge\config.json      ← token + port
C:\ProgramData\xeneon-bridge\bridge.log       ← service log
```

`service install` **migrates an existing portable token** (from `%USERPROFILE%\.xeneon-bridge`)
into `C:\ProgramData\xeneon-bridge` if one is there, so the widget keeps working without a
re‑paste. Once the service config exists, every other subcommand (including `statusline`) reads
that same machine‑wide config automatically — so the statusline and the service always agree on
the token.

Both modes bind loopback only (`127.0.0.1` **and** `[::1]`) on port `8787`. Run **one** of the
two modes at a time — they'd otherwise fight over the port.

---

## Part 3 — Wire Claude Code's statusline

This is what feeds live data into the bridge. Claude Code runs your `statusLine` command every time
it renders the status bar and pipes it a JSON blob — model, cost, duration, context‑window %, and
(on Pro/Max) the 5‑hour and weekly rate‑limit figures. The bridge's `statusline` subcommand reads
that JSON and posts it to the running bridge. Pick whichever of the three wirings below fits how you
already run your statusline.

> **Windows path tip:** Claude Code runs the statusline command through Git Bash, so use
> **forward‑slash** paths (`C:/Tools/…/xeneon-bridge.exe`) everywhere below. Windows backslashes get
> swallowed by the shell and you'll get "command not found" and a blank status bar.

### Option A — let the bridge be your statusline

Simplest, if you don't have a statusline you care about. In `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "C:/Tools/xeneon-bridge/xeneon-bridge.exe statusline"
  }
}
```

The bridge posts the numbers and prints its own minimal bar (`Model | wk NN% | $C.CCC`).

### Option B — wrap an existing statusline command

Keep the bridge as the `statusLine` command and point the `XENEON_WRAP_CMD` environment variable at
your current statusline command. The bridge posts the numbers, then runs your command and passes its
output straight through to the terminal — so your bar looks exactly the same.

### Option C — keep your statusline script, just feed the bridge from it

If your statusline is already a shell script (e.g. `~/.claude/statusline.sh` that renders your own
bar with `jq`), you don't need to replace or wrap it — add **one fully‑detached line** that feeds
the bridge. Where the script captures the incoming JSON (most do `input=$(cat)` near the top), add:

```bash
input=$(cat)   # you almost certainly already have this line

# Post usage to the Xeneon bridge, fully detached so a slow or down bridge never
# delays your bar. Its own output is discarded — your script still renders as usual.
( printf '%s' "$input" | "C:/Tools/xeneon-bridge/xeneon-bridge.exe" statusline >/dev/null 2>&1 & )

# ... the rest of your script prints your statusline exactly as before ...
```

The `( … & )` subshell detaches the post, so your bar keeps rendering in well under a second even if
the bridge is down or slow. Don't set `XENEON_WRAP_CMD` in this mode — here the bridge is used purely
for its POST side effect, and your script owns the visible output.

---

## Notifications

The widget pops Claude Code notifications — permission prompts and "Claude is waiting for your
input" — over the dashboard: one per session, stacking when several arrive, auto-switching to the
notifying session, and fading after a configurable time. **Tap a toast to dismiss it early.** Tune
it under **Settings → Notifications** (`Notifications` on/off, `Notification duration` 3–120 s,
default 20 s).

### Test it without wiring anything

With the bridge running, fire a notification by hand — it shows on the widget within one poll cycle:

```bash
xeneon-bridge notify --title "Permission needed" --message "Bash wants to run a command" --session <session-id>
```

`--session` is optional (omit it and the toast shows on whatever session is on screen); add
`--type permission` for the amber "needs attention" styling.

### Wire real notifications

Add a `Notification` hook to `~/.claude/settings.json`, pointing at your `xeneon-bridge.exe`. The
command runs through bash, so use **forward-slash paths**, and it is fully detached (`( … & )`) so a
slow or down bridge can never stall Claude Code:

```json
"hooks": {
  "Notification": [
    { "hooks": [
      { "type": "command",
        "command": "input=$(cat); ( printf '%s' \"$input\" | \"C:/path/to/xeneon-bridge.exe\" hook notify >/dev/null 2>&1 & )" }
    ] }
  ]
}
```

Replace `C:/path/to/xeneon-bridge.exe` with your actual binary — the same one your statusline uses.

> **Editing `settings.json` mid-session doesn't take effect immediately.** Run `/hooks` once (or
> restart Claude Code) so the running session picks up the new hook. After that, real Claude Code
> notifications land on the Edge automatically — alongside whatever your terminal already shows.

---

## Remote sessions (multi-machine)

One bridge can serve Claude Code sessions from several machines — local ones over loopback,
and remote ones (e.g. an SSH box) over your LAN. Remote sessions appear in the widget's
`‹ folder · host n/m ›` switcher, labeled by hostname; local sessions show just the folder.

**1. Expose the bridge (Windows box).** By default it listens on loopback only. To accept LAN
connections, set `bind` in the bridge config (`C:\ProgramData\xeneon-bridge\config.json` for the
service, or `%USERPROFILE%\.xeneon-bridge\config.json` for portable mode) and restart it:

```json
{ "port": 8787, "token": "…", "bind": "0.0.0.0" }
```

`"0.0.0.0"` = all interfaces; a specific LAN IP (e.g. `"192.168.1.50"`) exposes only that one and
still keeps loopback for the local widget. Then allow the port through Windows Firewall (elevated):

```
netsh advfirewall firewall add rule name="Xeneon Bridge 8787" dir=in action=allow protocol=TCP localport=8787
```

Find the box's LAN IP with `ipconfig`. **Security:** the `X-Bridge-Token` is the only gate and it
travels in cleartext over plain HTTP — only expose the bridge on a network you trust (home/office
LAN or a VPN like Tailscale), never the open internet.

**2. Build a Linux binary.** Cross-compile from the Windows box and copy it over:

```bash
GOOS=linux GOARCH=amd64 go -C companion build -o xeneon-bridge-linux .   # arm64 if the box is ARM
ssh you@remote mkdir -p /home/you/bin
scp companion/xeneon-bridge-linux you@remote:/home/you/bin/xeneon-bridge
ssh you@remote chmod +x /home/you/bin/xeneon-bridge
```

**3. Point the remote box at the bridge.** Create `~/.xeneon-bridge/config.json` **by hand** with
the Windows bridge's URL and the **same token** (do this before running any `xeneon-bridge` command,
or it will auto-generate a fresh token that won't match):

```json
{ "url": "http://192.168.1.50:8787", "token": "<same-token-as-the-windows-bridge>" }
```

**4. Wire the remote statusline + Notification hook** — same patterns as the local box, with the
Linux binary path. In the remote `~/.claude/statusline.sh`:

```bash
( printf '%s' "$input" | /home/you/bin/xeneon-bridge statusline >/dev/null 2>&1 & )
```

In the remote `~/.claude/settings.json`:

```json
"hooks": { "Notification": [ { "hooks": [ { "type": "command",
  "command": "input=$(cat); ( printf '%s' \"$input\" | /home/you/bin/xeneon-bridge hook notify >/dev/null 2>&1 & )" } ] } ] }
```

Start a Claude session on the remote box — it shows up on the Edge labeled with its hostname, and
its notifications pop there too.

---

## Part 4 — Package the widget

The widget source is in [`claudeusage/`](claudeusage/). To build the installable `.icuewidget`
you need the **iCUE Widget CLI** (`icuewidget`) on your PATH.

```powershell
# From the repo root
icuewidget validate claudeusage
icuewidget package claudeusage -o claudeusage.icuewidget
```

`validate` checks the structure/manifest; `package` produces `claudeusage.icuewidget`, the single
file you install in iCUE.

Notes for anyone editing the widget:
- `manifest.json` sets **`"interactive": true`** — required, or touch/tap/swipe never reach the
  widget.
- iCUE's in‑app importer is stricter than `validate`: keep `<title>` **before**
  `<link rel="icon">` in the `<head>`, with an uppercase `<!DOCTYPE html>`. (The current
  `index.html` already follows this.)

---

## Part 5 — Install the widget in iCUE

1. Make sure the **bridge is running** (Part 2) and the **statusline is wired** (Part 3).
2. Open **iCUE** (5.44+).
3. Go to the **widgets** section, click the **+** above the list of available widgets, and select
   your `claudeusage.icuewidget` file.
4. **Claude Usage** now appears in the widget list — add it to your **Xeneon Edge** canvas/layout.
5. Open the widget's **settings** and fill in the **Connection** group:
   - **Bridge Token** — paste the `token` from your `config.json`
     (`%USERPROFILE%\.xeneon-bridge\config.json` in portable mode, or
     `C:\ProgramData\xeneon-bridge\config.json` in service mode). *This is the one required step.*
   - **Bridge URL** — leave as `http://localhost:8787` unless you changed the port.
   - **Refresh** — poll interval in seconds (default 2).
   - **Show** — display limits as **Used** or **Left**.
   - **Inactivity timeout** — how long a quiet session stays on screen before the widget goes idle
     (default 30 min).
   - Plus **Personalization** (text/accent/background color, transparency).

Start a Claude Code session and the numbers appear within a second or two.

**Updating the widget later:** re‑importing on top of a running instance doesn't always take.
The reliable way to update is to **remove the existing Claude Usage widget from the layout, then
import the new `.icuewidget` and add it back**.

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| Widget shows **"Add your bridge token in settings"** | Paste the `token` from `config.json` into the widget's **Bridge Token** setting. |
| Widget shows **"Bridge offline…"** | The bridge isn't running or the URL/port is wrong. Start it (`serve` or `service start`) and confirm **Bridge URL** = `http://localhost:8787`. |
| Widget shows **"Waiting for Claude Code…"** | Bridge is up but no session has posted real usage yet. Start a `claude` session; make sure the **statusline** is wired (Part 3). |
| Numbers never update | The statusline command path is wrong or points at the wrong binary. Verify the path in `~/.claude/settings.json` and that token matches. |
| **`service install` → "access denied"** | Run the command from an **elevated (Administrator)** PowerShell. |
| Taps/swipes do nothing on device | The installed widget is missing `"interactive": true`, or it's a stale import — re‑package and re‑import fresh (remove, then add). |
| Rate‑limit figures show `—` | The 5‑hour/weekly fields only appear on Pro/Max plans and after the first API response of a session. |

---

## License

MIT
