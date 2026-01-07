<div align="center">

# FLACidal

[![Download](https://img.shields.io/badge/Download-Latest-blue?style=for-the-badge&logo=codeberg)](https://codeberg.org/KushieMoon-dev/FLACidal/releases)
[![GitHub](https://img.shields.io/badge/GitHub-Mirror-181717?style=for-the-badge&logo=github)](https://github.com/kushiemoon-dev/FLACidal/releases)

**Download Lossless FLAC Music from Tidal & Qobuz**

*A beautiful desktop app for downloading your music library in pristine lossless quality*

![Windows 10+](https://img.shields.io/badge/Windows-10+-0078D6?style=flat-square&logo=windows)
![macOS 10.13+](https://img.shields.io/badge/macOS-10.13+-000000?style=flat-square&logo=apple)
![Linux](https://img.shields.io/badge/Linux-Any-FCC624?style=flat-square&logo=linux&logoColor=black)

</div>

---

## Features

- **Multi-Source** — Download from both Tidal and Qobuz
- **Lossless Quality** — HI_RES (24-bit), LOSSLESS (16-bit), HIGH options
- **Full Albums** — Download entire playlists, albums, or single tracks
- **Auto Metadata** — Complete tagging with Vorbis comments
- **Cover Art** — Album artwork embedded directly in FLAC files
- **Fast Downloads** — Up to 10 concurrent parallel downloads
- **Queue System** — Manage and monitor download progress in real-time
- **Built-in Search** — Find tracks, albums, and artists directly in the app
- **Download History** — Track and re-download previous items
- **Organized Output** — Automatic folder structure per playlist/album

---

## Download

**[⬇️ Download Latest Release](https://codeberg.org/KushieMoon-dev/FLACidal/releases)**

| Platform | Download |
|----------|----------|
| Windows (x64) | `flacidal-windows-amd64.exe` |
| macOS (Intel) | `flacidal-darwin-amd64` |
| macOS (Apple Silicon) | `flacidal-darwin-arm64` |
| Linux (x64) | `flacidal-linux-amd64` |

---

## Screenshot

<div align="center">
<img src="screenshot.jpg" alt="FLACidal Screenshot" width="800">
</div>

---

## Usage

```
1. Launch the application
2. Paste a Tidal or Qobuz URL
3. Select your download folder
4. Click "Download All FLAC"
```

### Supported URLs

| Service | URL Format |
|---------|-----------|
| **Tidal** | `https://tidal.com/browse/playlist/...` |
| | `https://tidal.com/browse/album/...` |
| | `https://tidal.com/browse/track/...` |
| **Qobuz** | `https://www.qobuz.com/album/...` |
| | `https://www.qobuz.com/playlist/...` |
| | `https://www.qobuz.com/track/...` |

---

## Output Structure

```
~/Music/
└── Artist Name/
    └── Album Title/
        ├── 01 - Track One.flac
        ├── 02 - Track Two.flac
        ├── 03 - Track Three.flac
        └── cover.jpg
```

---

## Configuration

Settings location: `~/.flacidal/config.json`

| Setting | Default | Options |
|---------|---------|---------|
| Quality | `LOSSLESS` | `HI_RES`, `LOSSLESS`, `HIGH` |
| File Format | `{artist} - {title}` | Custom template |
| Embed Cover | `true` | `true`, `false` |
| Concurrent Downloads | `4` | `1` - `10` |

---

## Build from Source

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone from Codeberg
git clone https://codeberg.org/KushieMoon-dev/FLACidal.git

# Or clone from GitHub
git clone https://github.com/kushiemoon-dev/FLACidal.git

cd FLACidal
wails build

# Binary output: build/bin/flacidal
```

### Development Mode

```bash
wails dev
```

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.23 |
| Frontend | Svelte + TypeScript |
| Framework | [Wails](https://wails.io/) v2.11 |
| Audio | FLAC with Vorbis comments |
| Database | SQLite |

---

## Project Structure

```
FLACidal/
├── main.go                 # App entry point
├── app.go                  # Wails bindings
├── backend/
│   ├── tidal.go            # Tidal API client
│   ├── source_qobuz.go     # Qobuz API client
│   ├── downloader.go       # FLAC download service
│   ├── download_manager.go # Concurrent queue
│   ├── tagger.go           # Metadata tagging
│   ├── database.go         # SQLite storage
│   └── config.go           # Configuration
└── frontend/src/
    ├── pages/              # Home, Queue, Search, etc.
    ├── components/         # Reusable UI
    └── stores/             # Svelte state
```

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

## Disclaimer

> **This tool is intended for educational and personal use only.**
>
> - Only download music you have the legal right to access
> - Respect the terms of service of Tidal and Qobuz
> - Support artists by purchasing their music
> - Not affiliated with Tidal, Qobuz, or any streaming service

---

<div align="center">

Made with ♥ by [KushieMoon](https://codeberg.org/KushieMoon-dev)

</div>
