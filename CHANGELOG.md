# Changelog

## v4.17.0 — 2026-08-27

### Fixes
- **Self-hosted priority endpoints (Tidal/Qobuz/Amazon) were configured but never actually used** (#13) — the setting existed in Settings and got saved, but nothing in the desktop app or the headless server (`cmd/server`) ever wired it into the endpoint pool. A self-hosted instance just sat there doing nothing: every request still went through the public community pool, which is what was actually getting rate-limited and blacklisted, and the source got reported as fully dead even though your own instance was fine the whole time. Root cause lived in flacidal-core (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md) for the pool internals) — self-hosted endpoints are now tried first, with the public pool only as fallback if self-host is actually down.
- **Status page could still show a source as "dead" while its self-host instance was healthy** — a blacklisted-but-not-dead endpoint counted as "up" for the self-host badge but "down" for the overall status, so the two could contradict each other on the same row. The overall status now defers to self-host health when it's the one keeping a source alive.
- **A source with no valid endpoints configured at all (e.g. a rejected self-host URL) was misreported as "upstream is down"** instead of "nothing configured" — mattered most for Amazon, whose pool has no fallback-to-defaults if every configured URL gets rejected by the security filter (must be `https://`, or `http://` on loopback/private addresses only).
- Double `soulseek: soulseek:` prefix in fallback error messages, and a couple of cases where a Soulseek failure could get misattributed.

### Known limitation
A self-hosted endpoint that accepts a connection and then never responds (hangs, no error) isn't currently detected as unhealthy — only real failures (5xx, 401/403, connection refused, DNS failure) are. If your self-host instance is up but broken in a way that just hangs requests, it'll keep getting tried and slow things down rather than falling back. A real crash or an HTTP error response is handled correctly.

### Internal
- Bumped `flacidal-core` to v0.20.0 — see its changelog for the pool/tier internals.

## v4.16.0 — 2026-08-21

### New features
- **Dolby Atmos in the quality selector** — the UI-side counterpart to flacidal-core's Atmos support; `ATMOS` was already a valid value for `Config.downloadQuality`, just missing its `<option>`.
- **ReplayGain toggle in Settings** — wires `Config.enableReplayGain` through `GetConfig`/`SaveConfig`/`ResetToDefaults`.
- **BPM and musical key in the Audio Quality Analyzer** — best-effort via the optional `aubiotrack`/`keyfinder-cli` binaries (zero value if either is missing), shown as new columns in the results table and now persisted into the file's tags via `EmbedAudioFeatures`, not just shown in the UI and lost when the page closes. Detected keys are converted to Camelot notation so they match what auto-tagged Tidal downloads already show, instead of the Analyzer saying "Dm" for the same key Tidal tags as "7A".
- **Scan an existing folder for fake-lossless FLAC files** — the Analyzer's "select folder" path used to be a stub that silently fell back to single-file picking. Reuses the same recursive FLAC listing as folder conversion, so a library can be checked retroactively for files an untrusted source delivered as fake lossless before the download-time gate existed.
- **Deezer retag from the Quality Analyzer** — Scan Folder and single-file analysis now also fill in album/tracknumber/discnumber/year/genre/cover from Deezer, not just BPM/key, in the same pass.
- **The analyzer accepts non-FLAC files** — both the multipart upload and the `{"path": ...}` JSON variant used to only handle `.flac`; both now go through flacidal-core's format-agnostic analyzer, which still gives FLAC its full fake-lossless spectral check and gives every other supported format (mp3/m4a/wav/ogg/opus/...) real sample-rate/spectral data without a false lossless verdict.

### Fixes
- **Self-hosted Tidal endpoints didn't apply to playlist/album/track browsing** — that path runs through a separate `TidalHifiService` instance than the downloader, which never received the custom/priority endpoints configured in Settings, so browsing stayed on the public pool even with a self-host set up. The instances are now kept in sync.
- **Download history stayed empty** — `DownloadManager.SetJobCompleteCallback` existed to persist a history entry after every job, but nothing ever registered it, making failed tracks impossible to diagnose after the fact. Wired on startup.
- **Single-track downloads and retries lost album metadata** — `QueueSingleDownload`/`RetryDownload` already fetched the full Tidal track before queueing but only pulled ISRC/title/artist out of it, so the retag step never saw album/tracknumber/discnumber/year/cover. Now passes the whole track through.

### Internal
- Bumped `flacidal-core` to v0.19.0 — installed extensions can now actually be used as a download source (previously listable but inert), song.link scraping kicks in when Odesli's API rate-limits, and several crash-safety fixes (atomic tag writes, staged downloads, oversized-cover rejection, streaming ISRC scans). See flacidal-core's own changelog for detail.
- Go bumped to 1.26.5, `.golangci.yml` migrated to v2 format, GitHub Actions pinned to commit SHA, a stale `postcss` bumped for an osv-scanner CVE flag.

## v4.15.2 — 2026-07-25

### Fixes
- Soulseek could still be silently dropped from the source order via the desktop app's Settings save or the REST API (`POST /api/sources/order`) — the fix previously shipped for FLACidal-Core's internal RPC layer didn't cover these two paths. Both now re-add it automatically when omitted.
- The headless server (`cmd/server`) never constructed or registered a Soulseek source at all — even with Soulseek enabled, it could never be reached through this binary. Now registered on startup, same as the desktop app.
- The default source order resolved on first run wasn't written back to the saved config, so Settings could keep showing a stale order.

## v4.15.1 — 2026-07-17

### Fixes
- **`sldl` auto-installer downloaded a dead URL** — pinned to a `fiso64/sockseek` tag that's been removed upstream, breaking both the in-app "Install sldl" button and the Docker image build (also fixed there). Repinned to the current stable release.

### Internal
- Core dependency bumped to `v0.16.1` (same fix, see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md)).

## v4.15.0 — 2026-07-17

### New features
- **Self-hosted instance priority for Amazon** — reaches parity with Tidal/Qobuz: a priority-endpoint list tried before the shared community pool, live-reloaded on Settings save without a restart.
- **Instance-aware default source order** — a source with a self-hosted instance configured now skips ahead of Soulseek in the default priority order, instead of Soulseek always leading regardless of instance setup.
- **Odesli/song.link URL resolution** — pasting a Spotify, Apple Music, YouTube Music (or other Odesli-supported service) URL now resolves it to an equivalent Tidal or Deezer link automatically, in both the desktop app and the headless REST API. A toast confirms when this fallback was used.
- **Docker packaging** for the headless server (multi-stage build + `docker-compose.yml` + CI image publish).
- Settings: the self-host endpoint fields are now grouped under a collapsible "Advanced" section with a single explanation, a link to the README, and a live count badge per field — instead of three always-visible, unexplained text areas.

### Fixes
- `qobuzPriorityEndpoints` was persisted and shown in Settings but never actually applied — setting it silently did nothing.
- Self-hosted override endpoints (Tidal/Qobuz/Amazon) only took effect at app startup, not when changed via Settings — required a restart.
- The download orchestrator used a separate hardcoded priority list that ignored the configured source order entirely and always put Soulseek last, contradicting the documented Soulseek-first default.
- Home's URL fetch silently bypassed the Odesli fallback for unrecognized URLs, falling into a Tidal-only validation path that always failed for non-Tidal input.

### Internal
- Core dependency bumped to `v0.16.0` — Amazon self-host support, instance-aware default source order, Odesli URL resolver (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md)).
- AUR package definition bumped to track v4.14.0 (was still on v4.11.0).

---

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
