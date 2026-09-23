# Changelog

## v4.18.0: 2026-09-23

### New features
- **Automatic update checks and installs**: the app now checks for updates at launch (the "Check for Updates" button in Settings is still there for an on-demand check). A user 3 or more releases behind is blocked with a full-screen prompt until they update; updating downloads the new release, verifies its SHA256 checksum, and replaces the running binary in place before relaunching. The version comparison bug that made `4.10.0` look older than `4.9.0` (a lexicographic string comparison, not semver) is also fixed, as is asset selection always grabbing the first release asset regardless of platform.

## v4.17.2: 2026-09-10

### Fixes
- **Some Tidal downloads failed even when the endpoint itself was healthy** (#14): a track with no manifest at the requested quality is a per-track gap, not an endpoint problem, but it was blacklisting the whole endpoint anyway. After three such tracks the endpoint went fully dead for up to two hours, taking it out of rotation for every other download. Root cause and fix live in flacidal-core (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md)), bumped to v0.21.1.
- **A Tidal metadata endpoint returning 401 never showed up as unhealthy on the Status page** (#16): the metadata request path never reported failures back to the endpoint pool, so a broken or misconfigured endpoint could keep failing every metadata call while Settings > Status still reported it live. Fixed in the same flacidal-core bump.

## v4.17.1: 2026-09-09

### Fixes
- **Release binaries kept showing the old version number in the app UI**: `wails.json` and `frontend/package.json` weren't bumped for the v4.16.0 and v4.17.0 tags, so both builds shipped with the fixes but still reported themselves as 4.15.2. No functional change from v4.17.0, this release exists to get the version string in sync with the tag.

## v4.17.0: 2026-08-27

### Fixes
- **Self-hosted priority endpoints (Tidal/Qobuz/Amazon) were configured but never actually used** (#13): the setting existed in Settings and got saved, but nothing in the desktop app or the headless server (`cmd/server`) ever wired it into the endpoint pool. A self-hosted instance just sat there doing nothing: every request still went through the public community pool, which is what was actually getting rate-limited and blacklisted, and the source got reported as fully dead even though your own instance was fine the whole time. Root cause lived in flacidal-core (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md) for the pool internals). Self-hosted endpoints are now tried first, with the public pool only as fallback if self-host is actually down.
- **Status page could still show a source as "dead" while its self-host instance was healthy**: a blacklisted-but-not-dead endpoint counted as "up" for the self-host badge but "down" for the overall status, so the two could contradict each other on the same row. The overall status now defers to self-host health when it's the one keeping a source alive.
- **A source with no valid endpoints configured at all (e.g. a rejected self-host URL) was misreported as "upstream is down"** instead of "nothing configured"; mattered most for Amazon, whose pool has no fallback-to-defaults if every configured URL gets rejected by the security filter (must be `https://`, or `http://` on loopback/private addresses only).
- Double `soulseek: soulseek:` prefix in fallback error messages, and a couple of cases where a Soulseek failure could get misattributed.

### Known limitation
A self-hosted endpoint that accepts a connection and then never responds (hangs, no error) isn't currently detected as unhealthy; only real failures (5xx, 401/403, connection refused, DNS failure) are. If your self-host instance is up but broken in a way that just hangs requests, it'll keep getting tried and slow things down rather than falling back. A real crash or an HTTP error response is handled correctly.

### Internal
- Bumped `flacidal-core` to v0.20.0: see its changelog for the pool/tier internals.

## v4.16.0: 2026-08-21

### New features
- **Dolby Atmos in the quality selector**: the UI-side counterpart to flacidal-core's Atmos support; `ATMOS` was already a valid value for `Config.downloadQuality`, just missing its `<option>`.
- **ReplayGain toggle in Settings**: wires `Config.enableReplayGain` through `GetConfig`/`SaveConfig`/`ResetToDefaults`.
- **BPM and musical key in the Audio Quality Analyzer**: best-effort via the optional `aubiotrack`/`keyfinder-cli` binaries (zero value if either is missing), shown as new columns in the results table and now persisted into the file's tags via `EmbedAudioFeatures`, not just shown in the UI and lost when the page closes. Detected keys are converted to Camelot notation so they match what auto-tagged Tidal downloads already show, instead of the Analyzer saying "Dm" for the same key Tidal tags as "7A".
- **Scan an existing folder for fake-lossless FLAC files**: the Analyzer's "select folder" path used to be a stub that silently fell back to single-file picking. Reuses the same recursive FLAC listing as folder conversion, so a library can be checked retroactively for files an untrusted source delivered as fake lossless before the download-time gate existed.
- **Deezer retag from the Quality Analyzer**: Scan Folder and single-file analysis now also fill in album/tracknumber/discnumber/year/genre/cover from Deezer, not just BPM/key, in the same pass.
- **The analyzer accepts non-FLAC files**: both the multipart upload and the `{"path": ...}` JSON variant used to only handle `.flac`; both now go through flacidal-core's format-agnostic analyzer, which still gives FLAC its full fake-lossless spectral check and gives every other supported format (mp3/m4a/wav/ogg/opus/...) real sample-rate/spectral data without a false lossless verdict.

### Fixes
- **Self-hosted Tidal endpoints didn't apply to playlist/album/track browsing**: that path runs through a separate `TidalHifiService` instance than the downloader, which never received the custom/priority endpoints configured in Settings, so browsing stayed on the public pool even with a self-host set up. The instances are now kept in sync.
- **Download history stayed empty**: `DownloadManager.SetJobCompleteCallback` existed to persist a history entry after every job, but nothing ever registered it, making failed tracks impossible to diagnose after the fact. Wired on startup.
- **Single-track downloads and retries lost album metadata**: `QueueSingleDownload`/`RetryDownload` already fetched the full Tidal track before queueing but only pulled ISRC/title/artist out of it, so the retag step never saw album/tracknumber/discnumber/year/cover. Now passes the whole track through.

### Internal
- Bumped `flacidal-core` to v0.19.0: installed extensions can now actually be used as a download source (previously listable but inert), song.link scraping kicks in when Odesli's API rate-limits, and several crash-safety fixes (atomic tag writes, staged downloads, oversized-cover rejection, streaming ISRC scans). See flacidal-core's own changelog for detail.
- Go bumped to 1.26.5, `.golangci.yml` migrated to v2 format, GitHub Actions pinned to commit SHA, a stale `postcss` bumped for an osv-scanner CVE flag.

## v4.15.2: 2026-07-25

### Fixes
- Soulseek could still be silently dropped from the source order via the desktop app's Settings save or the REST API (`POST /api/sources/order`): the fix previously shipped for FLACidal-Core's internal RPC layer didn't cover these two paths. Both now re-add it automatically when omitted.
- The headless server (`cmd/server`) never constructed or registered a Soulseek source at all; even with Soulseek enabled, it could never be reached through this binary. Now registered on startup, same as the desktop app.
- The default source order resolved on first run wasn't written back to the saved config, so Settings could keep showing a stale order.

## v4.15.1: 2026-07-17

### Fixes
- **`sldl` auto-installer downloaded a dead URL**: pinned to a `fiso64/sockseek` tag that's been removed upstream, breaking both the in-app "Install sldl" button and the Docker image build (also fixed there). Repinned to the current stable release.

### Internal
- Core dependency bumped to `v0.16.1` (same fix, see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md)).

## v4.15.0: 2026-07-17

### New features
- **Self-hosted instance priority for Amazon**: reaches parity with Tidal/Qobuz: a priority-endpoint list tried before the shared community pool, live-reloaded on Settings save without a restart.
- **Instance-aware default source order**: a source with a self-hosted instance configured now skips ahead of Soulseek in the default priority order, instead of Soulseek always leading regardless of instance setup.
- **Odesli/song.link URL resolution**: pasting a Spotify, Apple Music, YouTube Music (or other Odesli-supported service) URL now resolves it to an equivalent Tidal or Deezer link automatically, in both the desktop app and the headless REST API. A toast confirms when this fallback was used.
- **Docker packaging** for the headless server (multi-stage build + `docker-compose.yml` + CI image publish).
- Settings: the self-host endpoint fields are now grouped under a collapsible "Advanced" section with a single explanation, a link to the README, and a live count badge per field, instead of three always-visible, unexplained text areas.

### Fixes
- `qobuzPriorityEndpoints` was persisted and shown in Settings but never actually applied; setting it silently did nothing.
- Self-hosted override endpoints (Tidal/Qobuz/Amazon) only took effect at app startup, not when changed via Settings; required a restart.
- The download orchestrator used a separate hardcoded priority list that ignored the configured source order entirely and always put Soulseek last, contradicting the documented Soulseek-first default.
- Home's URL fetch silently bypassed the Odesli fallback for unrecognized URLs, falling into a Tidal-only validation path that always failed for non-Tidal input.

### Internal
- Core dependency bumped to `v0.16.0`: Amazon self-host support, instance-aware default source order, Odesli URL resolver (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md)).
- AUR package definition bumped to track v4.14.0 (was still on v4.11.0).

---

## v4.14.0: 2026-07-14

### New features
- **Headless server is now fully usable in a browser**: the server previously returned `501 not implemented` for 9 endpoints (search, file listing, metadata, cover art, ffmpeg info, conversion, lyrics) and served an empty embedded frontend, and every frontend component called Wails-only bindings with no browser fallback. The server now has full API coverage (reusing existing `internal/app` logic rather than duplicating it) and serves the built SPA; the frontend itself now runs correctly in a plain browser via a runtime-detecting client layer (`lib/api.ts`/`lib/websocket.ts`/`lib/runtime.ts`) that picks Wails bindings or `fetch()`/WebSocket calls depending on where it's running. Native-OS-only actions (file/folder dialogs, native drag-drop) degrade gracefully in browser mode instead of throwing. See the new README section on running headless in a browser.
- `go test`/`go vet`/`golangci-lint` now run in CI (previously only a build check ran; the existing test suite under `internal/` was never executed).

### Fixes
- `GetConversionFormats`'s HTTP handler returned a hardcoded stub missing the `qualities` field the frontend reads unconditionally; would have crashed the converter in browser mode. Now returns real data.
- History filters (`contentType`/`search`) were silently dropped by the HTTP handler; config reset was wiping the download folder instead of preserving it.
- Two `nolint:errcheck` suppressions were silently non-functional (a stray em dash broke golangci-lint's directive parser).

### Internal
- Core dependency bumped to `v0.15.0`: real spectral fake-lossless detection, YouTube/Cobalt fallback dispatch fix, dehardcoded endpoints (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md)).

---

## v4.13.0: 2026-07-11

### Fixes
- A toast now shows when starting a download without a folder configured, instead of failing silently
- Nil-guards added around config/Qobuz source/downloader/source manager, preventing crashes on missing state
- `gofiber/fiber` and `golang.org/x/net` bumped, resolving 3 Dependabot advisories
- Corrected a stale `flacidal-core@v0.13.0` checksum in `go.sum`
- Platform emoji icons replaced with inline SVG (gold → violet accent)

### Internal
- Core dependency bumped to `v0.14.0`: native Soulseek client for mobile parity, several Soulseek reliability fixes (nil-context panic, login-scoped context starving search, truncated files reported as success), endpoint cooldown ETA surfaced, internal Spotify/Tidal credentials and the Tidal HiFi mirror base URL now configurable via env instead of hardcoded (see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md))
- `app.go` (2,600+ lines) split into per-domain files under `internal/app/`, each with new characterization tests
- Removed the dead Analyzer page and its now-unused handler stubs
- Reliable, self-hosted star-history badge (replaces the flaky third-party service), with a dedicated PAT for branch-protected pushes
- French UI strings and comments translated to English
- Go version badge bumped to 1.26+

---

## v4.12.0: 2026-07-02

### New features
- **Soulseek tried first**: the download manager now attempts Soulseek before the proxy-dependent Tidal/Qobuz path, instead of only as a last resort, so it's reliable by default once configured
- **External Library Paths**: Settings -> Skip Existing Files now accepts additional folders (e.g. a separately-located Navidrome/Jellyfin library) to check for ISRC matches, alongside the download folder
- **Jellyfin scan trigger**: Settings -> Soulseek adds a Jellyfin toggle, server URL, and API key; triggers a debounced library scan a few seconds after a download batch finishes
- **AUR packaging**: `packaging/aur/PKGBUILD` for a `flacidal-bin` package (not yet published to aur.archlinux.org)
- **Landing page**: `docs/index.html`, a single-file GitHub Pages site with a live source-health preview, per-OS downloads, and app screenshots

### Fixes
- Four "Naming Preset" entries mixed folder and filename templates in the wrong field, silently producing a mangled flat filename instead of the folder structure their label promised; removed the redundant ones, kept "Multi-disc" as filename-only
- `tidalPriorityEndpoints`/`qobuzPriorityEndpoints` were missing from the settings save payload; edits to those fields never persisted across a restart
- AUR PKGBUILD depended on `webkit2gtk` (4.0, not in official Arch repos); corrected to `webkit2gtk-4.1` after confirming the actual runtime dependency by launching the built binary and inspecting its loaded libraries

### Internal
- Core dependency bumped to `v0.13.0` (multi-source endpoint discovery, dedup across all sources, Jellyfin scan trigger, see [flacidal-core's changelog](https://github.com/kushiemoon-dev/flacidal-core/blob/main/CHANGELOG.md))

---

## v4.11.0: 2026-06-23

### New features
- **Lyrics Manager**: new tool page (Tools sidebar): batch-fetch and embed lyrics into FLAC files via LRCLIB, per-file success/error results
- **AIFF converter**: added AIFF (`pcm_s16be`) to Audio Converter alongside existing WAV/ALAC/MP3/AAC/Opus
- **Cooldown auto-stop**: when all Tidal endpoints enter cooldown, queue auto-pauses and emits an `endpoint-cooldown` Wails event; toast notification shows countdown; toggle in Settings → Downloads
- **Google Fonts dynamic injection**: `applyFontFamily` now injects a `<link>` tag for any Google Font at runtime, enabling custom fonts beyond the static presets
- **Preview URL propagation**: `SourceTrack.PreviewURL` now populated from Tidal and Spotify sources; desktop home page preview player already consumed this field
- **UPC metadata**: UPC/barcode written as `UPC=` in Vorbis comments and `TXXX:BARCODE` in ID3 tags; sourced from Deezer enrichment and Qobuz album response
- **Popularity field**: play count/popularity score (0–100) written as `POPULARITY=` in Vorbis and `TXXX:POPULARITY` in ID3; sourced from Tidal and Spotify
- **ISRC region**: Spotify search now passes `&market={countryCode}` on ISRC and query lookups; country code flows from Config through `SpotifyClient.SetCountryCode`

### Fixes
- E2E mock: added `GetRecentAlbums` and `GetSldlStatus` (missing stubs caused console-error cascade in 4 tests)
- E2E settings tests: updated selectors to match current UI (textarea + renamed labels)
- Navigation test: updated tool count 4→5 and added Lyrics Manager route test

### Internal
- `progressEvent` struct uses named fields + `eventType` for non-default event routing
- Core dependency bumped to `v0.12.0`

---

## v4.10.0: 2026-06-21

### New features
- **Endpoint revival backoff**: dead proxy endpoints are no longer permanently blacklisted for the session. A dead endpoint enters a backoff queue and re-enters the pool as a probation candidate, starting at 5 minutes and doubling on each failed revival up to a 2 hour cap; a successful download from a probation endpoint resets it. A `200 OK` on metadata routes no longer counts as healthy on its own either: a response body containing the "upstream api error" signature of a proxy with banned upstream credentials now triggers immediate blacklisting instead of a 30 to 90 second stall before falling through to Soulseek.
- **Self-host priority endpoints** (Settings > Sources): the single custom endpoint fields for Tidal and Qobuz became multi-line lists, tried in order before the public pool, which stays as a fallback rather than being removed. Existing single custom endpoint values migrate automatically into the new lists.
- **Per-endpoint health panel** (Settings > Status): each proxy endpoint now shows its own state badge (live, probation, blacklisted, or dead), last-request latency, and revival count. Health is read from in-memory pool snapshots, no network requests, which also fixes a Linux crash where live HTTP probes conflicted with WebKitGTK's signal handling.
- **Queue cascade badges**: a verdict badge (Lossless, Likely upscaled, or Upscaled) on Soulseek and Bandcamp downloads, the sources where bit-perfect lossless can't be guaranteed at the protocol level; Tidal, Qobuz, and Amazon aren't analyzed since they're lossless by construction. A cascade badge shows every source that was tried before the one that succeeded.
- Home page empty state now shows source chips (Tidal HiFi, Qobuz, Bandcamp, Soulseek P2P, Spotify), and the placeholder typewriter includes Spotify and Bandcamp example URLs.

## v4.9.0: 2026-06-17

### New features
- **Source health engine**: a 403, 401, 429, or Cloudflare/captcha challenge page from a proxy endpoint now blacklists it immediately instead of counting as a success; after 3 consecutive failures an endpoint is marked dead and never revived for the session (superseded by the backoff system in v4.10.0), and `IsAvailable()` for Qobuz/Amazon now returns false once every endpoint is dead so the orchestrator skips straight to Soulseek instead of timing out first. Each source in the fallback chain is also wrapped in its own panic recovery, so one source panicking no longer aborts the rest of the chain.
- **Source Health panel** (Settings > Status): a Check Sources button runs concurrent live probes, bounded to 8s per source, Qobuz/Amazon over real proxy requests and Soulseek as an instant local check to avoid ban risk, reporting online, degraded, or dead with latency and a reason.
- **One-click sldl installer** (Settings > Soulseek): downloads, extracts, and installs the pinned Soulseek client release for the current platform with a progress bar, and re-initializes the Soulseek source without a restart.

### Internal
- First Go test suite for the download orchestration chain: endpoint blacklisting/revival, orchestrator panic isolation, and Soulseek output parsing.
- Core dependency bumped to v0.10.7.

## v4.8.6: 2026-06-17
- **Soulseek download succeeded despite transient reconnection failures being reported as errors** (#8).

## v4.8.5: 2026-06-17

### Fixes
- **Soulseek fallback failed silently on Windows/macOS**: `sldl`'s error output was discarded, so a Gatekeeper quarantine kill on macOS or an AV/SmartScreen block on Windows produced an empty result that matched no error branch, and the app just showed a generic "Connection failed" with nothing logged (#10). The binary is now made executable and de-quarantined at startup, its raw output is logged to the in-app terminal (password excluded), and new error branches give an actionable hint for each failure mode.
- Reaching Soulseek when the Qobuz fallback fails on Windows (#8).

## v4.8.4: 2026-06-08

### Fixes
- **Soulseek login test always failed on Windows/macOS (#10)**: it waited for search results on an inbound listen port, which the default firewall on both platforms blocks; Soulseek authentication itself is outbound-only, which is why Nicotine+ worked fine with the same credentials. The test now checks for the outbound `Logged in <user>` line from `sldl -v` instead, so it no longer depends on inbound connectivity.
- **Downloads failed whenever Soulseek was the only enabled source and Tidal was down (#8)**: metadata (artist, title, ISRC) was resolved exclusively through the Tidal proxy, so the Fetch button stayed disabled with no Tidal reachable, and Soulseek was never actually reached. The Universal (Deezer) search tab now provides a fully Tidal-independent path: it resolves metadata from the public Deezer API and routes straight to the orchestrator, which can search Soulseek by title and artist alone, no ISRC required.

### Internal
- `sldl`'s search timeout raised to 12s to tolerate slower indirect (server-mediated) peer connections on firewalled clients, plus a startup log when Soulseek is enabled but fails to initialize.
- Core dependency bumped to v0.10.4.

## v4.8.3: 2026-06-06

### Fixes
- **Qobuz short share-button URLs weren't recognized**: `open.qobuz.com/album/{id}` and `play.qobuz.com/album/{id}` (the format Qobuz's own share button produces) were routed to the Tidal parser and rejected as an invalid Tidal URL. The URL detection now accepts both the short format and the full storefront format.

### Internal
- App version is now read dynamically from `wails.json` at build time instead of being hardcoded, removing a source of version drift between the config and the UI.
- Core dependency bumped to v0.10.3.

## v4.8.2: 2026-06-06

### Fixes
- **Soulseek settings silently reset to disabled** (#7): `Settings.svelte`'s config loader omitted the `soulseekEnabled`/username/password fields from the response it read, so remounting the Settings page (it's destroyed and recreated on every navigation) reset the toggle to its default and the next save overwrote the persisted values with it.

### Internal
- Core dependency bumped to v0.10.1, which also fixes a concurrent-download panic.

## v4.8.1: 2026-06-05

### Fixes
- **Soulseek (`sldl`) was never found on Windows**: FLACidal looked for the binary at a Linux-only path on every platform, so Windows users always got a "not found" error regardless of where they placed it. The expected path is now `%APPDATA%\flacidal\sldl.exe` on Windows and the existing `~/.local/share/flacidal/sldl` elsewhere, and the in-app message shows the correct path for the current platform.

## v4.8.0: 2026-06-03

### New features
- **Guided, verifiable Soulseek setup** (Settings > General > Soulseek): previously required knowing the right binary path and hoping the credentials were correct, with no feedback until a download failed. Adds an info box explaining what Soulseek is and that existing Nicotine+ credentials work as-is, a status indicator for whether `sldl` is installed and its version, and a Login button that tests credentials live against the network before saving.
- Settings' General tab layout rebalanced: Soulseek moved to the left column so both columns are roughly equal height instead of Downloads/Appearance sitting alone against a much taller Sources/Quality/Soulseek column.

### Fixes
- Soulseek toggle wasn't rendering (wrong CSS class), long setting descriptions crushed the input field width, and the Soulseek source now registers/unregisters live on Save without a restart.

### Internal
- Core dependency bumped to v0.10.0, which adds `UnregisterSource` to `SourceManager`.

## v4.7.0: 2026-05-23

### New features
- **7 Qobuz proxy providers**: the proxy pool grew from 3 endpoints sharing one request format to 7 across four independent providers (dab, wjhe, gdstudio, musicdl, from [SpotiFLAC](https://github.com/spotbye/SpotiFLAC)), tried in order with the first success winning. Previously, all three original endpoints going down at once meant Qobuz downloads failed outright. A provider can be excluded via `qobuzProvidersDisabled` in `~/.flacidal/config.json`. No UI changes, existing config and downloads are unaffected.

## v4.6.0: 2026-05-23

### New features
- **Drag-and-drop source priority** (Settings): reorder Tidal/Qobuz/Amazon/Soulseek priority live via the new `SetSourceOrder` RPC, with allowlist and dedup validation and a drag-cancel reset.
- **Universal Deezer search tab**: works even when Tidal is down, resolving metadata from the public Deezer API for 30 results with ISRC.
- **Recent albums grid on the home page**: shows previously downloaded albums pulled from download history.

### Fixes
- Replaced a raw `console.error` with a toast on the Deezer search handler, propagated a swallowed DB error, and switched an album-fetch effect to `onMount` on the home page.

## v4.4.0: 2026-05-23

### New features
- **Bandcamp source**: name-your-price FLAC downloads from `bandcamp.com/track` and `bandcamp.com/album` URLs, positioned in the fallback chain between Amazon and Soulseek (core v0.7.0).

## v4.3.0: 2026-05-22

### New features
- **Circuit breaker + two new metadata sources**: Deezer and Spotify wired in as metadata-only URL sources, Tidal wired into the download manager for circuit-breaker health checks, and the database wired into the orchestrator and Soulseek source for ISRC caching (core v0.6.0).

### Internal
- Core dependency bumped to v0.5.1 (a Soulseek port conflict fix) and then v0.5.2 (a Soulseek mutex fix).

## v4.2.0: 2026-05-22

### New features
- **Multi-source download fallback**: FLACidal now tries sources in sequence, Tidal HiFi, then Qobuz, then Amazon Music, then Soulseek, instead of failing outright when the first is unavailable. Shipped in response to every major Tidal HiFi community proxy returning `403 Upstream API error` that month, with Qobuz proxies also down.
- **Soulseek P2P source**: a last-resort source powered by [sldl](https://github.com/fiso64/slsk-batchdl), the Soulseek batch downloader. Unlike the proxy-based sources it can't be broken by an upstream API change; downloads are FLAC-only and serialized one at a time to avoid connection conflicts. Setup is optional, via new credentials fields in Settings > Sources > Soulseek.

### Internal
- New `DownloadOrchestrator` (generic multi-source fallback engine), `ISRCSearchable`/`TitleArtistSearchable` interfaces for cross-source matching, `.gitignore` now excludes audio files and Playwright artifacts. Core dependency bumped to v0.5.2.

## v4.1.0: 2026-05-18

### New features
- **Spotify discography queuing**: paste a Spotify artist discography URL to fetch and queue every album, matched against Tidal.
- **Per-track download history**: a paginated log of every completed or failed download with source, quality, and file path, backed by new `/api/track-history` and a WebSocket queue panel broadcasting live progress to every connected client.
- **Audio Quality Analyzer**: drag-and-drop FLAC spectrum analysis with a lossless/upscaled verdict, its first appearance as a dedicated tool page.
- **Dynamic Tidal endpoints**: background refresh of the community HiFi proxy pool with a disk cache, plus custom self-hosted Tidal/Qobuz endpoint fields in Settings.
- **Vorbis output format** added to the Audio Converter, with q4/q6/q8/q10 quality presets.
- **Qobuz community mirrors**: credential-free fallback via community proxies.

## v4.0.6: 2026-04-16

### Fixes
- **Qobuz credentials UI removed**: credential-free proxy mode is the default now, no account needed.
- **Selected download quality didn't persist across sessions.**
- **Community endpoints blocked by ad-blocking DNS resolvers**: added a DNS sinkhole bypass.

## v4.0.5: 2026-04-16

### Fixes
- Removed UPX compression on the Windows build and added `-trimpath -s -w` instead, to reduce antivirus false positives.

### Internal
- Core dependency bumped to v0.4.4.

## v4.0.4: 2026-04-13

### Fixes
- **Album and artist search stopped working**: Tidal revoked the v1 credentials the app used directly; search is now routed through the community proxy pool instead, the same path track downloads already used.

## v4.0.3: 2026-04-13
- Core dependency bumped to v0.4.3 (search query normalization).

## v4.0.2: 2026-04-13
- Core dependency bumped to v0.4.2. CI's Go version bumped to 1.26 to match.

## v4.0.1: 2026-04-13

### Fixes
- **Search was completely broken after the Svelte 5 runes migration**: six state variables (search query, results, in-progress flag) were left as plain `let` bindings instead of `$state()`, so they were never reactive; the search box always read as empty, keeping the button disabled and the Enter key a no-op. Also, the desktop backend had a single hardcoded Tidal proxy with no fallback, so search broke completely whenever that one host was unreachable; it now carries the same endpoint list as flacidal-core.
- A type error and an accessibility label warning, and Vite pinned to v6.4.2 for plugin compatibility and a security fix.

### Internal
- Added `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, a PR template, issue templates, and a revised `SECURITY.md`.

## v4.0.0: 2026-04-09

Major UI redesign: a new design system, four dedicated audio tool pages, and improved UX across every page.

### New features
- **Design system**: new typography (Plus Jakarta Sans), an accent color system with gold highlights, animated tab navigation.
- **Sidebar redesign**: icon-only navigation with tooltips and a Tools flyout for the new audio utilities.
- **Audio tools suite**: Audio Quality Analyzer (batch upscale detection across FLAC/MP3/M4A/AAC), Audio Resampler, Audio Converter (MP3/AAC/OGG/Opus/ALAC/WAV), and File Manager (rename via metadata templates, browse tracks/lyrics/covers).
- **Home page**: typewriter URL animation, region selector, Recent Fetches cards with cover art.
- **History**: tabbed Downloads/Fetches view with search, sort, and styled empty states.
- **About page**: project showcase with live GitHub stats and a Ko-fi support link.
- **Settings**: reorganized into General, File Management, and Status tabs with a 2-column layout.
- Page fade transitions, reusable empty-state components, click/success/error sound effects, a GitHub issue reporter modal, card hover effects, and a font selector (Plus Jakarta Sans, Outfit, Bricolage Grotesque).

### Fixes
- Vite bumped to v6.4.2, fixing a path traversal and a WebSocket vulnerability.

## v3.3.0: 2026-04-04

### Fixes
- **Qobuz Hi-Res downloads failed validation**: fixed alongside a migration of the headless server command from the old `backend` package to the new `flacidal-core` module.

### Internal
- Core dependency bumped to v0.3.0.

## v3.2.1: 2026-04-04

### New features
- **Direct Qobuz download support**: download jobs are now source-aware and route Qobuz-sourced jobs straight to `QobuzSource.DownloadTrack` instead of through the Tidal path, with an HTTP status check and a dedicated Wails binding.

### Fixes
- **Queue events could be silently dropped**: the event channel send was non-blocking, so a full buffer just discarded the event instead of waiting; it now blocks.
- `HI_RES_LOSSLESS`/`HI_RES_MAX` normalized to the valid Tidal API quality parameter `HI_RES` before requests, the default quality changed from `HI_RES_LOSSLESS` to `HI_RES`, and duplicate quality options merged in the Settings dropdown.

### Internal
- Core dependency bumped three times this release (v0.2.1, then v0.2.2 and v0.2.3 the same day to work around a Go checksum-DB mismatch on the first retagged v0.2.1), picking up an FFI use-after-free fix, quality normalization, and a playlist pagination fix for playlists of 500+ tracks.

## v3.2.0: 2026-04-03

### New features
- **Hi-Res by default**: default quality changed from Lossless to Hi-Res Lossless, with automatic fallback through Hi-Res, then Lossless, then High instead of failing outright.
- Clearer error messages for HTML manifest failures, previously a cryptic JSON parse error.

### Internal
- First scaffold of the Flutter mobile app: FFI bindings to the Go core, Riverpod providers, go_router with 5-tab navigation, and a Material 3 dark theme, the starting point for what later became the separate FLACidal-Mobile repo.
- Desktop's imports migrated from the local `backend` package to the standalone `flacidal-core` module.

## v3.1.0: 2026-03-25

12 new features focused on media server compatibility, search, and download workflow.

### New features
- **Folder cover art**: saves `folder.jpg` in album directories, recognized by Plex, Jellyfin, and Kodi.
- **LRC lyrics export**: synced `.lrc` or plain `.txt` lyrics alongside FLAC files, for players like foobar2000 or Poweramp.
- **Multi-type search**: separate Tracks, Albums, and Artists tabs, with whole-album download and artist profile browsing from search results.
- **Folder structure presets**: templates like `{artist}/{album}` or `{year}/{artist}/{album}`, or a custom pattern.
- **Full date metadata**: `DATE` now carries the full `YYYY-MM-DD`, plus a new `ORIGINALDATE` Vorbis comment.
- **Source badge**: completed downloads show whether they came from Tidal or Qobuz.
- **Convert Folder** button on the Files page batch-converts every FLAC in a folder.
- Animated cycling URL placeholder on the home page input, and locale-formatted track/file counts.

### Fixes
- The update checker pointed at the wrong repository.

## v3.0.0: 2026-03-11

Feature parity release: smart downloads, a redesigned settings panel, and dozens of UX improvements.

### New features
- **Smart ISRC skip**: detects an existing file by its ISRC tag and skips re-downloading it.
- **Track availability checking**: grays out unavailable tracks before they're queued.
- **192kHz Hi-Res Lossless** quality tier with automatic fallback.
- **Playlist pagination** for playlists over 100 tracks, and `DISCNUMBER`/`DISCTOTAL` tags for multi-disc albums.
- **Settings overhaul**: tabbed into General, Sources, Metadata, Appearance, and Advanced, with 15+ naming presets, a configurable artist separator, new `{date}`/`{albumartist}`/`{discnumber}` template variables, a Tidal region selector, and a flat-vs-organized playlist subfolder toggle.
- **API status checker**, an update checker (automatic and manual), a debounced search filter, sort-by-status on the queue, one-click Clean & Retry, confirmation dialogs on destructive actions, a track right-click context menu, and an Open Config Folder button.
- **FFmpeg auto-installer** with a progress bar, and a folder-wide audio converter.
- Explicit content badges, formatted play counts.

### Internal
- The queue event channel was serialized to prevent WebKit/GTK crashes on Linux.

## v2.0.1: 2026-02-20

Patch release, no functional changes: adds the automated GitHub Actions build and release pipeline that produces `flacidal.exe` (Windows), `flacidal.dmg` (macOS universal), and `flacidal.AppImage` (Linux), all published automatically on each version tag.

## v2.0.0: 2026-02-20

Major download engine overhaul, a full Svelte 5 migration, and a wide range of new features.

### New features
- **Download engine**: HiFi/Qobuz endpoint rotation, cross-source fallback by ISRC, automatic quality fallback, M3U8 playlist generation, and failed-download export.
- **Tidal**: full artist discography browsing, profile picture/banner download, a 30 second audio preview, an explicit content badge, popularity-based sorting, ALAC format support, drag-and-drop FLAC analysis, and `COPYRIGHT`/`ORGANIZATION` Vorbis tags.
- Toast notifications, a queue status filter bar, HTTP/SOCKS5 proxy support (for restricted regions), per-icon hover animations, and accessibility `aria-label`s.

### Fixes
- The Files page, History page, Mix URLs, Tidal v1 credentials, and light mode, all broken at various points before this release.

### Internal
- Migrated the frontend from Svelte 3 to Svelte 5 (runes) and Vite 4 to Vite 6, updated Go and frontend dependencies, and replaced debug `println` calls with structured logging.

## v1.0.0: 2026-02-12

Initial public release. A desktop app for downloading lossless FLAC music from Tidal, built with Go, Svelte, and Wails.

### New features
- Track, album, and playlist downloads in lossless FLAC quality, with a concurrent worker-pool queue and real-time progress.
- Spotify ISRC and metadata matching against Tidal results via the Client Credentials flow.
- Automatic FLAC metadata tagging (Vorbis comments) and album artwork embedding.
- A queue manager, a file browser for downloaded tracks, and search.
- SQLite-backed match and history caching, with configuration persisted at `~/.flacidal/`.

Supported platforms at launch: Windows (amd64), macOS (amd64, arm64), Linux (amd64).
