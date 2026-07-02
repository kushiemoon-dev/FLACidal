# Changelog

## v4.12.0 — 2026-07-02

### New features
- **Soulseek tried first** — the download manager now attempts Soulseek before the proxy-dependent Tidal/Qobuz path, instead of only as a last resort, so it's reliable by default once configured
- **External Library Paths** — Settings -> Skip Existing Files now accepts additional folders (e.g. a separately-located Navidrome/Jellyfin library) to check for ISRC matches, alongside the download folder
- **Jellyfin scan trigger** — Settings -> Soulseek adds a Jellyfin toggle, server URL, and API key; triggers a debounced library scan a few seconds after a download batch finishes
- **AUR packaging** — `packaging/aur/PKGBUILD` for a `flacidal-bin` package (not yet published to aur.archlinux.org)
- **Landing page** — `docs/index.html`, a single-file GitHub Pages site with a live source-health preview, per-OS downloads, and app screenshots

### Fixes
- Four "Naming Preset" entries mixed folder and filename templates in the wrong field, silently producing a mangled flat filename instead of the folder structure their label promised; removed the redundant ones, kept "Multi-disc" as filename-only
- `tidalPriorityEndpoints`/`qobuzPriorityEndpoints` were missing from the settings save payload — edits to those fields never persisted across a restart
- AUR PKGBUILD depended on `webkit2gtk` (4.0, not in official Arch repos); corrected to `webkit2gtk-4.1` after confirming the actual runtime dependency by launching the built binary and inspecting its loaded libraries

### Internal
- Core dependency bumped to `v0.13.0` (multi-source endpoint discovery, dedup across all sources, Jellyfin scan trigger — see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md))

---

## v4.11.0 — 2026-06-23

### New features
- **Lyrics Manager** — new tool page (Tools sidebar): batch-fetch and embed lyrics into FLAC files via LRCLIB, per-file success/error results
- **AIFF converter** — added AIFF (`pcm_s16be`) to Audio Converter alongside existing WAV/ALAC/MP3/AAC/Opus
- **Cooldown auto-stop** — when all Tidal endpoints enter cooldown, queue auto-pauses and emits an `endpoint-cooldown` Wails event; toast notification shows countdown; toggle in Settings → Downloads
- **Google Fonts dynamic injection** — `applyFontFamily` now injects a `<link>` tag for any Google Font at runtime, enabling custom fonts beyond the static presets
- **Preview URL propagation** — `SourceTrack.PreviewURL` now populated from Tidal and Spotify sources; desktop home page preview player already consumed this field
- **UPC metadata** — UPC/barcode written as `UPC=` in Vorbis comments and `TXXX:BARCODE` in ID3 tags; sourced from Deezer enrichment and Qobuz album response
- **Popularity field** — play count/popularity score (0–100) written as `POPULARITY=` in Vorbis and `TXXX:POPULARITY` in ID3; sourced from Tidal and Spotify
- **ISRC region** — Spotify search now passes `&market={countryCode}` on ISRC and query lookups; country code flows from Config through `SpotifyClient.SetCountryCode`

### Fixes
- E2E mock: added `GetRecentAlbums` and `GetSldlStatus` (missing stubs caused console-error cascade in 4 tests)
- E2E settings tests: updated selectors to match current UI (textarea + renamed labels)
- Navigation test: updated tool count 4→5 and added Lyrics Manager route test

### Internal
- `progressEvent` struct uses named fields + `eventType` for non-default event routing
- Core dependency bumped to `v0.12.0`

---

## v4.10.0 — 2026-06-23

- Self-host priority pool, per-endpoint health panel, cascade transparency
- Soulseek UX (Nicotine+ info box, login test, layout rebalanced)
