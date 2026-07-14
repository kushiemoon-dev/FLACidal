# Changelog

## v4.14.0 — 2026-07-14

### New features
- **Headless server is now fully usable in a browser** — the server previously returned `501 not implemented` for 9 endpoints (search, file listing, metadata, cover art, ffmpeg info, conversion, lyrics) and served an empty embedded frontend, and every frontend component called Wails-only bindings with no browser fallback. The server now has full API coverage (reusing existing `internal/app` logic rather than duplicating it) and serves the built SPA; the frontend itself now runs correctly in a plain browser via a runtime-detecting client layer (`lib/api.ts`/`lib/websocket.ts`/`lib/runtime.ts`) that picks Wails bindings or `fetch()`/WebSocket calls depending on where it's running. Native-OS-only actions (file/folder dialogs, native drag-drop) degrade gracefully in browser mode instead of throwing. See the new README section on running headless in a browser.
- `go test`/`go vet`/`golangci-lint` now run in CI (previously only a build check ran — the existing test suite under `internal/` was never executed).

### Fixes
- `GetConversionFormats`'s HTTP handler returned a hardcoded stub missing the `qualities` field the frontend reads unconditionally — would have crashed the converter in browser mode. Now returns real data.
- History filters (`contentType`/`search`) were silently dropped by the HTTP handler; config reset was wiping the download folder instead of preserving it.
- Two `nolint:errcheck` suppressions were silently non-functional (a stray em dash broke golangci-lint's directive parser).

### Internal
- Core dependency bumped to `v0.15.0` — real spectral fake-lossless detection, YouTube/Cobalt fallback dispatch fix, dehardcoded endpoints (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md)).

---

## v4.13.0 — 2026-07-11

### Fixes
- A toast now shows when starting a download without a folder configured, instead of failing silently
- Nil-guards added around config/Qobuz source/downloader/source manager, preventing crashes on missing state
- `gofiber/fiber` and `golang.org/x/net` bumped, resolving 3 Dependabot advisories
- Corrected a stale `flacidal-core@v0.13.0` checksum in `go.sum`
- Platform emoji icons replaced with inline SVG (gold → violet accent)

### Internal
- Core dependency bumped to `v0.14.0` — native Soulseek client for mobile parity, several Soulseek reliability fixes (nil-context panic, login-scoped context starving search, truncated files reported as success), endpoint cooldown ETA surfaced, internal Spotify/Tidal credentials and the Tidal HiFi mirror base URL now configurable via env instead of hardcoded (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md))
- `app.go` (2,600+ lines) split into per-domain files under `internal/app/`, each with new characterization tests
- Removed the dead Analyzer page and its now-unused handler stubs
- Reliable, self-hosted star-history badge (replaces the flaky third-party service), with a dedicated PAT for branch-protected pushes
- French UI strings and comments translated to English
- Go version badge bumped to 1.26+

---

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
