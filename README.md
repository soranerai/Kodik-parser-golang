# Kodik Parser (Golang)

A fast, lightweight command-line parser and downloader for Kodik video pages — written in Go.  
This tool extracts video links (including best-quality streams), handles serials (multiple episodes), and can optionally download or open results in mpv-net.

Cool, minimal and pragmatic — for developers who want reliable scraping of Kodik pages.

---

## Features

- Parse Kodik movie and serial pages
- Extract iframe/player URLs and secret method calls
- Handle serial episodes ranges interactively
- Attempt to find best-quality stream URL
- Optional downloading (two downloader modes) and opening in mpv-net
- Progress bars and basic logging

---

## Requirements

- Go 1.18+ (or later)
- Windows / Linux / macOS with a Go toolchain
- Optional: mpv-net (if using OpenInMpvNet option)

---

## Install / Build

Clone the repo and build:

```bash
git clone <repo-url> "Kodik-parser-golang"
cd "Kodik-parser-golang"
go build -o kodik-parser main.go
```

Run locally:

```bash
./kodik-parser
```

---

## Configuration

The app reads `config.json` from the current directory. Example `config.json`:

```json
{
  "DownloadResults": true,
  "DownloaderVersion": 1,
  "OpenInMpvNet": false,
  "OutputDirectory": "downloads",
  "LogLevel": "info"
}
```

Key options:
- DownloadResults (bool) — whether to download found videos
- DownloaderVersion (1 or 2) — select downloader implementation (1 = normal, 2 = HLS-aware)
- OpenInMpvNet (bool) — open results in mpv-net instead of downloading or printing
- OutputDirectory (string) — where downloads are stored
- LogLevel (string) — logging verbosity (e.g., debug, info, warn)

Adjust `config.json` according to your needs.

---

## Usage

1. Start the program:
   - If `debug` flag in code is true, the URL can be hardcoded for quick debugging.
   - Otherwise the program prompts: `Введите URL:` — paste the Kodik page URL.

2. For serials, you'll be asked to input episode range:
   - Example input: `1-10` (from episode 1 to 10)
   - If only one episode exists, it will auto-select it.

3. After parsing, the tool prints results or downloads/open them depending on configuration.

Notes:
- The program normalizes and validates URLs before processing.
- Long-running network operations use custom timeouts and progress bars.

---

## Examples

Parse a single movie page:

```bash
./kodik-parser
# Enter movie URL when prompted
```

Parse a serial and download episodes (ensure config.json has DownloadResults=true):

```bash
./kodik-parser
# Enter serial URL when prompted
# Input episode range like: 1-8
```

---

## Troubleshooting

- If parsing fails, check logs printed to console. Increase verbosity in `config.json` (if implemented).
- Network timeouts can be caused by the remote host or local firewall; check connectivity.
- If downloads fail, verify `OutputDirectory` permissions.

---

## Contributing

Contributions and improvements are welcome. Please:
- Open issues for bugs or feature requests
- Send pull requests with focused changes and clear descriptions

---

Enjoy ripping Kodik responsibly. Keep scraping ethical and respect site terms of service.
