<div align="center">

[![Download](https://img.shields.io/badge/Download-Latest-blue?style=for-the-badge&logo=codeberg)](https://codeberg.org/KushieMoon-dev/FLACidal/releases)
[![GitHub](https://img.shields.io/badge/GitHub-Mirror-181717?style=for-the-badge&logo=github)](https://github.com/kushiemoon-dev/FLACidal/releases)

<img src="assets/banner.png" width="600" alt="FLACidal">

_Download lossless FLAC music from Tidal & Qobuz_

![Windows 10+](https://img.shields.io/badge/Windows-10+-0078D6?style=for-the-badge&logo=windows)
![macOS 10.13+](https://img.shields.io/badge/macOS-10.13+-000000?style=for-the-badge&logo=apple)
![Linux](https://img.shields.io/badge/Linux-Any-FCC624?style=for-the-badge&logo=linux&logoColor=black)

</div>

---

<div align="center">
<img src="assets/screenshot.png" alt="FLACidal Screenshot" width="800">
</div>

---

## Features

| Download | Tools & UI |
|----------|------------|
| Tidal & Qobuz support | Built-in Tidal search |
| HI_RES 24-bit & LOSSLESS 16-bit | Queue with real-time progress |
| Playlists, albums, single tracks | Download history & re-download |
| Up to 10 parallel downloads | Integrated file manager |
| Auto Vorbis comment tagging | FLAC quality analyzer |
| Embedded cover art | HTTP & SOCKS5 proxy support |
| Custom filename templates | Artist profile picture download |

---

## Download

| Platform | Download |
|----------|----------|
| Windows (x64) | [flacidal.exe](https://github.com/kushiemoon-dev/FLACidal/releases/latest/download/flacidal.exe) |
| macOS (Universal) | [flacidal.dmg](https://github.com/kushiemoon-dev/FLACidal/releases/latest/download/flacidal.dmg) |
| Linux (x64) | [flacidal.AppImage](https://github.com/kushiemoon-dev/FLACidal/releases/latest/download/flacidal.AppImage) |

All releases on [GitHub](https://github.com/kushiemoon-dev/FLACidal/releases) · [Codeberg](https://codeberg.org/KushieMoon-dev/FLACidal/releases)

---

## Usage

1. Launch the application
2. Paste a Tidal or Qobuz URL into the input field
3. Select your download folder
4. Click **Download All FLAC**

### Supported URLs

| Service | Supported Types |
|---------|----------------|
| **Tidal** | Playlist, Album, Track, Mix, Artist |
| **Qobuz** | Album, Playlist, Track |

---

## Output Structure

```
~/Music/
└── Playlist Name/
    ├── Artist - Track One.flac
    ├── Artist - Track Two.flac
    └── cover.jpg
```

---

## Configuration

Settings location: `~/.flacidal/config.json`

| Setting | Default | Options |
|---------|---------|---------|
| Quality | `LOSSLESS` | `HI_RES`, `LOSSLESS`, `HIGH` |
| File naming | `{artist} - {title}` | Custom template |
| Embed cover art | `true` | `true` / `false` |
| Concurrent downloads | `4` | `1` – `10` |
| Proxy | _(none)_ | `http://...` or `socks5://...` |

---

## FAQ

**Is this free?**
Yes. No account, subscription, or API credentials required — authentication is handled by the app itself.

**Do I need a Tidal or Qobuz account?**
No. FLACidal handles authentication internally. Just paste a URL and download.

**Why does my antivirus flag the file?**
False positive. Go-compiled binaries are sometimes flagged heuristically. Build from source if you have concerns.

**What audio quality can I download?**
From Tidal: HI_RES (24-bit / up to 192 kHz) and LOSSLESS (16-bit / 44.1 kHz). From Qobuz: up to 24-bit FLAC depending on your subscription tier.

**Can I use a proxy?**
Yes. HTTP and SOCKS5 proxies are supported and configurable in Settings.

---

## Build from Source

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
git clone https://codeberg.org/KushieMoon-dev/FLACidal.git
cd FLACidal
wails build
# Binary: build/bin/flacidal
```

Development mode: `wails dev`

---

## Disclaimer

> This project is for **educational and personal use only**. The developer does not condone or encourage copyright infringement.
>
> **FLACidal** is a third-party tool and is not affiliated with, endorsed by, or connected to Tidal, Qobuz, or any other streaming service.
>
> You are solely responsible for:
> 1. Ensuring your use complies with your local laws.
> 2. Reading and adhering to the Terms of Service of Tidal and Qobuz.
> 3. Any legal consequences resulting from misuse of this tool.
>
> The software is provided "as is", without warranty of any kind. The author assumes no liability for any bans, damages, or legal issues arising from its use.

---

<div align="center">

Made with ♥ by [KushieMoon](https://codeberg.org/KushieMoon-dev)

⭐ Star this repo to be notified of new releases

</div>
