<div align="center">

<img src="docs/banner.png" alt="FLACidal" width="600">

### Multi-source lossless FLAC downloader with a Soulseek P2P backbone

[![GitHub Release](https://img.shields.io/github/v/release/kushiemoon-dev/FLACidal?style=flat-square&color=e5a00d)](https://github.com/kushiemoon-dev/FLACidal/releases/latest)
[![Stars](https://img.shields.io/github/stars/kushiemoon-dev/FLACidal?style=flat-square&color=a855f7)](https://github.com/kushiemoon-dev/FLACidal/stargazers)
[![License](https://img.shields.io/github/license/kushiemoon-dev/FLACidal?style=flat-square&color=gray)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)

![Windows](https://img.shields.io/badge/Windows-10+-0078D6?style=flat-square&logo=windows&logoColor=white)
![macOS](https://img.shields.io/badge/macOS-10.13+-000000?style=flat-square&logo=apple&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-any-FCC624?style=flat-square&logo=linux&logoColor=black)

</div>

---

## Overview

**FLACidal** is a desktop app that grabs lossless FLAC files with full metadata and embedded cover art attached. It tries Soulseek P2P, Tidal, Qobuz, Amazon Music, and Bandcamp in sequence, automatically moving to the next source until one succeeds.

> **Note:** Tidal, Qobuz, and Amazon have all tightened their APIs against third-party access. The community proxy pools FLACidal relies on to reach them go offline regularly, occasionally for days at a stretch. **Soulseek is currently the source you can count on**: it needs no proxy pool at all and takes about 5 minutes to set up. Tidal is tried first by default (see [source order](#download-chain) below); to give Soulseek priority over the proxy-dependent sources instead, move it to the top under **Settings -> General -> Source Mode**.

---

## Quick Start

**1. [Download FLACidal](#download) for your platform**

**2. [Set up Soulseek](#setting-up-soulseek) first: it matters**

**3. Paste a URL or run a search, then hit download**

```
Home tab  ->  paste a Tidal or Qobuz URL  ->  Fetch  ->  Download All FLAC
Search tab  ->  search by track / album / artist  ->  Download
```

Anytime, check **Settings -> Status** to see which sources are online.

---

## How It Works

### Download chain

FLACidal works through sources one at a time, in order, until one succeeds:

| Priority | Source | Quality | Notes |
|----------|--------|---------|-------|
| 1 | **Tidal** | FLAC / Hi-Res (24-bit) | Through the community proxy pool |
| 2 | **Qobuz** | FLAC / Hi-Res (24-bit) | Through the community proxy pool (optional) |
| 3 | **Amazon Music** | FLAC / UHD | Through the community proxy pool |
| 4 | **Bandcamp** | FLAC | Direct |
| 5 | **Soulseek P2P** | FLAC | Through `sldl` (needs a free account) |

Sources can be reordered under **Settings -> General -> Source Mode**, and whatever order is set gets followed top to bottom with no exceptions. Given how often the proxy pools go down, many users move Soulseek to the top for reliability.

### Two things called "proxy": they are different

The **community proxy pool** is relay infrastructure FLACidal maintains for Tidal, Qobuz, and Amazon. It comes built in, needs no setup, and if these servers go down, those sources stop working along with them.

Separately, an **outbound proxy (HTTP / SOCKS5)** routes FLACidal's own traffic through a network proxy you control: a corporate VPN, a SOCKS5 tunnel, whatever you already run. Most people can skip this entirely; when needed, it's configured under **Settings -> General -> HTTP / SOCKS5 Proxy**.

### Self-hosted / private endpoints

Every FLACidal user shares the community proxy pool, and that's exactly why it struggles: rate limits and cooldowns land on everyone at once. Run your own Tidal HiFi API or Qobuz proxy instance and point FLACidal at it, and it gets **tried before the community pool**, with no shared rate-limit queue and no dependence on pool uptime.

Set this up under **Settings -> General**:

- `tidalPriorityEndpoints`: one or more self-hosted Tidal HiFi API URLs, tried in order ahead of the public pool
- `qobuzPriorityEndpoints`: one or more self-hosted Qobuz proxy URLs, tried in order ahead of the public pool
- `amazonPriorityEndpoints`: one or more self-hosted Amazon proxy URLs, tried in order ahead of the public pool

Tidal and Qobuz also keep their old single-URL fields (`tidalCustomEndpoint`, `qobuzCustomEndpoint`) for backward compatibility. For new setups, though, the priority-list fields are the better pick, since they allow more than one fallback instance.

Leave these blank to stick with the default community pool.

### Source availability: what to expect

Major streaming platforms actively push back against unofficial API access, so expect the following:

- Proxy pools may drop offline with no warning
- Downloads can quietly fall through to Soulseek as a last resort
- **Right now, Soulseek is the most consistently available source**

You can check endpoint health live at any time under **Settings -> Status**.

---

## Features

- **Multi-Source Fallback**: cascades automatically across Soulseek, Tidal, Qobuz, Amazon, and Bandcamp
- **Soulseek P2P** stays unaffected by streaming-proxy uptime, so put it first in Source Mode for the most reliable results
- **Smart Dedup** checks every source plus an optional external library path (a Navidrome/Jellyfin library, say) and skips anything already on disk, matched by ISRC
- **Jellyfin Integration**: triggers a library scan automatically once a download batch finishes
- Streaming sources deliver **Hi-Res and Lossless** quality: 24-bit up to 192 kHz for Hi-Res, 16-bit / 44.1 kHz for Lossless
- Full **Tidal and Qobuz** coverage: playlists, albums, tracks, mixes, and artist pages
- **Built-in Search** across Tidal (Tracks / Albums / Artists) or Deezer via the Universel tab, which keeps working even when Tidal is down
- Up to 10 **Concurrent Downloads** in parallel, with live queue progress
- **Smart Metadata** handling: Vorbis comment tags, embedded cover art, and lyrics
- An **Audio Tools Suite** covering Quality Analyzer, Resampler, FFmpeg-powered Converter, and File Manager
- **Custom Filename Templates**, so you set your own naming format, e.g. `{artist} - {title}`
- **Artist Artwork** pulled in alongside the music
- A **Source Status Panel** showing live endpoint health under Settings -> Status
- **Outbound Proxy Support** for every outbound request, HTTP or SOCKS5

---

## Download

**[Download Latest Release](https://github.com/kushiemoon-dev/FLACidal/releases/latest)**

| Platform | File |
|----------|------|
| Windows x64 | `flacidal.exe` |
| macOS Universal | `flacidal.dmg` |
| Linux x64 | `flacidal.AppImage` |
| **Android** | [`FLACidal.apk`](https://github.com/kushiemoon-dev/flacidal-mobile/releases/latest) |
| **iOS** | [`FLACidal.ipa`](https://github.com/kushiemoon-dev/flacidal-mobile/releases/latest) (via AltStore) |

> **Linux:** No AUR package exists yet. Grab the AppImage directly, or [build from source](#build-from-source).

An Android and iOS build exists too: **[FLACidal Mobile](https://github.com/kushiemoon-dev/flacidal-mobile)**

Find every release on [GitHub](https://github.com/kushiemoon-dev/FLACidal/releases)

---

## Setting up Soulseek

Right now, Soulseek is FLACidal's most dependable source, and getting it running takes about 5 minutes.

### Step 1: Get a Soulseek account

- **Already using Nicotine+?** Use your existing username and password; FLACidal shares the same account system.
- **New user:** Sign up for a free account at [slsknet.org](https://www.slsknet.org/) (no email needed) or through the Nicotine+ app.

### Step 2: Install sldl

`sldl` (slsk-batchdl) is the command-line tool FLACidal relies on to talk to the Soulseek network.

1. Grab the latest binary for your platform from [github.com/fiso64/slsk-batchdl/releases](https://github.com/fiso64/slsk-batchdl/releases)
2. Drop it at this exact path:
   - **Linux / macOS:** `~/.local/share/flacidal/sldl`, then mark it executable: `chmod +x ~/.local/share/flacidal/sldl`
   - **Windows:** `%APPDATA%\flacidal\sldl.exe` (i.e. `C:\Users\YourName\AppData\Roaming\flacidal\sldl.exe`)

FLACidal picks up the binary on its own. Once found, the Soulseek section under **Settings -> General** shows a green checkmark.

### Step 3: Connect your account

1. Open FLACidal -> **Settings -> General**
2. Scroll to **Soulseek (Fallback P2P)**
3. Switch **Enable Soulseek** on
4. Fill in your **username** and **password**
5. Click **Login**; FLACidal checks the connection live and reports success or an error
6. Click **Save Changes**

<div align="center">
<img src="docs/screenshots/settings-general.png" alt="Settings, General tab showing Soulseek configuration" width="800">
</div>

### Step 4: Verify in Settings -> Status

Head to **Settings -> Status**. The `sldl` row should show green. If any proxy pool endpoints turn red, Soulseek automatically picks up the slack.

<div align="center">
<img src="docs/screenshots/settings-status.png" alt="Settings, Status tab showing endpoint health" width="800">
</div>

---

## Usage

### Home: download by URL

<div align="center">
<img src="docs/screenshots/home.png" alt="FLACidal Home tab" width="800">
</div>

1. **Paste a URL** into the field and hit **Fetch**
2. FLACidal pulls the track list and shows it on screen
3. Click **Download All FLAC** to add the tracks to the Queue
4. URLs you've fetched before show up as cards under the input, ready for a quick re-download

**Supported URL types:**

| Service | Types |
|---------|-------|
| **Tidal** | Playlist · Album · Track · Mix · Artist |
| **Qobuz** | Album · Playlist · Track |
| **Spotify** | Track · Album · Playlist (metadata only, routed to Tidal/Qobuz/Amazon/Soulseek for the actual FLAC) |
| **Deezer** | Track · Album · Playlist |

**Other services (Apple Music, YouTube Music, Deezer short links, ...):** FLACidal can't parse these directly. Instead it resolves them automatically through [Odesli/song.link](https://song.link) into an equivalent Tidal or Deezer URL before fetching, so pasting the link is all that's needed.

### Search: find music without leaving the app

<div align="center">
<img src="docs/screenshots/search.png" alt="FLACidal Search tab" width="800">
</div>

Open the **Search** tab, where four sub-tabs live:

| Tab | What it searches | Works when Tidal is down? |
|-----|-----------------|:---:|
| **Tracks** | Tidal track catalog | No |
| **Albums** | Tidal album catalog | No |
| **Artists** | Tidal artist pages | No |
| **Universel** | Deezer public catalog (by ISRC) | Yes |

**Universel** runs on Deezer's public API, so it keeps working regardless of Tidal proxy health. Reach for it whenever a Tidal search turns up empty.

### Queue: monitor and control downloads

<div align="center">
<img src="docs/screenshots/queue.png" alt="FLACidal Queue tab" width="800">
</div>

The **Queue** tab lists every active and pending download:

- A live progress bar per track
- **Pause / Resume** for the whole queue at once
- **Retry** a single failed download, or retry every failure in one go
- An export option for the list of failed downloads

### History and Files

**History** logs every download and URL fetch. Click any past entry to re-fetch it instantly.

**Files** shows every FLAC file in your download folder, with a button to open the folder in your system's file manager.

### Audio Tools

Reach the Tools panel through the grid icon in the sidebar:

| Tool | What it does |
|------|-------------|
| **Quality Analyzer** | Examines actual frequency content to confirm true lossless status, and reports BPM/musical key |
| **Resampler** | Adjusts sample rate (192 kHz down to 44.1 kHz, for instance) |
| **Converter** | Transcodes to other formats (MP3, AAC, Opus) through FFmpeg |
| **File Manager** | Batch-renames files based on metadata templates |

Converter, Resampler, and the Quality Analyzer's lossless check all need FFmpeg. Get it through your system's package manager, or use the in-app installer under **Settings -> Status**.

The Quality Analyzer's BPM/key detection needs [`aubio`](https://aubio.org/) and [`keyfinder-cli`](https://github.com/EvanPurkhiser/keyfinder-cli) on PATH, and both are optional (on Arch, for example: `pacman -S aubio libkeyfinder`, then `yay -S keyfinder-cli` for the AUR package). Skip them and BPM/key show blank, while the rest of the analysis still runs.

---

## Output Structure

```
~/Music/
└── Playlist Name/
    ├── Artist - Track One.flac
    ├── Artist - Track Two.flac
    └── cover.jpg
```

Both the download folder and the filename template can be fully customized under **Settings -> File Management**.

---

## Configuration

Settings live at `~/.flacidal/config.json` and can be edited from within the app via the Settings panel. The **Open Config Folder** button in Settings jumps straight to that directory.

| Setting | Default | Options |
|---------|---------|---------|
| Quality | `Lossless` | `Hi-Res` (24-bit/48kHz+) · `Lossless` (16-bit/44.1kHz) · `High` (320kbps, lossy) |
| File naming | `{artist} - {title}` | Custom template with metadata variables |
| Embed cover art | `true` | `true` · `false` |
| Concurrent downloads | `4` | `1` – `10` |
| Outbound proxy | _(none)_ | `http://host:port` or `socks5://host:port` |

The config file sits at `~/.flacidal/config.json`, while the `sldl` binary lives separately at `~/.local/share/flacidal/sldl` on Linux and macOS. Two distinct locations, easy to mix up if you're not expecting it.

---

## Build from Source

**Requirements:** [Go](https://go.dev) 1.26+ and [Wails v2](https://wails.io)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
git clone https://github.com/kushiemoon-dev/FLACidal.git
cd FLACidal
wails build
# Binary: build/bin/flacidal
```

For hot-reload development mode:

```bash
wails dev
```

---

## Headless / Run in browser

FLACidal can also run as a plain HTTP server, with no Wails and no desktop shell, controllable from any browser on the machine or the LAN. That fits a NAS, a home server, or anywhere a desktop UI doesn't make sense.

### Docker

The image bundles the server, the built frontend, `ffmpeg`, `aubio` (BPM), and `sldl` (Soulseek), so nothing extra needs installing. `keyfinder-cli` (musical key) isn't bundled yet, since there's no Debian package for it and it needs a source build, so key detection stays empty in Docker for now.

```bash
curl -O https://raw.githubusercontent.com/kushiemoon-dev/FLACidal/main/docker-compose.yml
docker compose up -d
```

Then open `http://localhost:8080`. Downloads go to `./music/FLACidal`, while config and the database persist across restarts in the `flacidal-config` volume. Port and volume layout details are in [docker-compose.yml](docker-compose.yml).

### From source

```bash
git clone https://github.com/kushiemoon-dev/FLACidal.git
cd FLACidal
cd frontend && npm install && npm run build && cd ..
go run ./cmd/server
# or: make serve
```

Then open `http://localhost:8080`.

The server reads from the same `~/.flacidal/config.json` as the desktop app. It's controlled by two optional environment variables:

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `8080` | HTTP port the server listens on |
| `FRONTEND_DIST_DIR` | `frontend/dist` | Where to find the built SPA on disk |

Running `go run ./cmd/server` before the frontend is built works fine: the server still comes up and the API works on its own, though requests to `/` return a 503 nudging you to run `npm run build` first.

---

## FAQ

**Is a Tidal or Qobuz account required?**
Not for the streaming sources: FLACidal handles authentication itself through the community proxy pools. Soulseek is a different story though: Soulseek P2P needs an account, and given how the proxies have been behaving lately, setting one up for free is well worth it.

**Everything fails or times out; nothing downloads. What now?**
Check **Settings -> Status** first. Red proxy pool endpoints mean the streaming sources are down right now, which happens and is usually temporary. Make sure Soulseek is set up too (see [Setting up Soulseek](#setting-up-soulseek)); it doesn't depend on proxy pool health at all.

**What quality can I actually expect?**
Tidal gives Hi-Res (24-bit, up to 192 kHz) or Lossless (16-bit, 44.1 kHz). Qobuz goes up to 24-bit depending on what's available for a given album. Soulseek is entirely dependent on what other users are sharing, though FLACidal specifically searches for FLAC there.

**Why is my antivirus flagging the binary?**
That's a false positive. Heuristic scanners flag Go binaries for no real reason sometimes. Build from source instead if it bothers you.

**What does the outbound proxy setting do, and do I need it?**
It routes FLACidal's traffic through a personal proxy of your choosing: a corporate VPN, a SOCKS5 tunnel, whatever you already run. Most people can skip it entirely, and it has nothing to do with the community proxy pool that Tidal and Amazon rely on.

**Does Arch Linux get an AUR package?**
It does, as `flacidal-bin`. Install it with `yay -S flacidal-bin` or `paru -S flacidal-bin`; it just wraps the same `.AppImage` from the releases page.

---

## Star History

<div align="center">

[![Star History](docs/star-history.svg)](https://github.com/kushiemoon-dev/FLACidal/stargazers)

</div>

---

## Disclaimer

FLACidal exists strictly for **educational and personal use**. It has no affiliation with, endorsement from, or connection to Tidal, Qobuz, or any other streaming service. Making sure your use stays within local law and each platform's Terms of Service is entirely on you. The software comes "as is," with no warranty of any kind.

---

<div align="center">

**MIT License** · [Releases](https://github.com/kushiemoon-dev/FLACidal/releases) · [Mobile App](https://github.com/kushiemoon-dev/flacidal-mobile) · [YouFLAC](https://github.com/kushiemoon-dev/YouFLAC) · [OpenDrop-VJ](https://github.com/kushiemoon-dev/OpenDrop-VJ)

Built with love by [KushieMoon](https://github.com/kushiemoon-dev)

</div>
