# Build Directory

This directory holds the build assets and output files for the application.

Structure:

* bin - where build output lands
* darwin - macOS-specific files
* windows - Windows-specific files

## Mac

Files specific to Mac builds live under `darwin`.
Feel free to customize them; to reset to defaults, delete them
and
build with `wails build`.

The directory contains these files:

- `Info.plist` - the primary plist consumed by `wails build`.
- `Info.dev.plist` - the same, but used by `wails dev` instead.

## Windows

`windows` holds the manifest and rc files used by `wails build`.
Customize them as needed; deleting them and rebuilding with `wails build` restores the defaults.

- `icon.ico` - the application's icon, used by `wails build`. Swap in your own file to change it;
  if it's missing, a fresh `icon.ico` gets generated from `appicon.png` in this directory.
- `installer/*` - assets consumed by `wails build` to produce the Windows installer.
- `info.json` - Windows build metadata, surfaced both by the installer and by the app itself
  (exe -> right-click -> properties -> details)
- `wails.exe.manifest` - the application's manifest file.
