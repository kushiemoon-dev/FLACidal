#!/usr/bin/env bash
# Builds a self-contained FLACidal AppImage (GTK3 + webkit2gtk-4.1 bundled).
# Run on Ubuntu 22.04 after `wails build`; the glibc floor is that of the build host.
set -euo pipefail

BIN="${BIN:-build/bin/flacidal}"
ICON="${ICON:-build/appicon.png}"
OUT="${OUT:-build/bin/flacidal.AppImage}"
WORK="${WORK:-build/bin/appimage-work}"
LIBDIR=/usr/lib/x86_64-linux-gnu
WEBKIT_DIR="$LIBDIR/webkit2gtk-4.1"
GIO_TLS="$LIBDIR/gio/modules/libgiognutls.so"

# Pinned by version + sha256, bump by hand (linuxdeploy-plugin-gtk has no releases, so it is pinned by commit)
LINUXDEPLOY_URL=https://github.com/linuxdeploy/linuxdeploy/releases/download/1-alpha-20251107-1/linuxdeploy-x86_64.AppImage
LINUXDEPLOY_SHA=c20cd71e3a4e3b80c3483cef793cda3f4e990aca14014d23c544ca3ce1270b4d
GTK_PLUGIN_URL=https://raw.githubusercontent.com/linuxdeploy/linuxdeploy-plugin-gtk/7a3fbc31a9e5075073ff8790f26effbac5f84453/linuxdeploy-plugin-gtk.sh
GTK_PLUGIN_SHA=b0f4cbc684a0103a9651f0955b635eaea0096b3a66c0f5a2c2aa337960375171
APPIMAGETOOL_URL=https://github.com/AppImage/appimagetool/releases/download/1.9.1/appimagetool-x86_64.AppImage
APPIMAGETOOL_SHA=ed4ce84f0d9caff66f50bcca6ff6f35aae54ce8135408b3fa33abfc3cb384eb0

command -v convert >/dev/null || { echo "ImageMagick (convert) is required" >&2; exit 1; }
for f in "$BIN" "$ICON" "$WEBKIT_DIR/WebKitWebProcess" "$WEBKIT_DIR/WebKitNetworkProcess" "$GIO_TLS"; do
  [ -f "$f" ] || { echo "missing required file: $f" >&2; exit 1; }
done

fetch() { # url sha256 dest
  curl -fsSL "$1" -o "$3"
  echo "$2  $3" | sha256sum -c --quiet -
  chmod +x "$3"
}

rm -rf "$WORK"
mkdir -p "$WORK"
WORK="$(cd "$WORK" && pwd)"
fetch "$LINUXDEPLOY_URL" "$LINUXDEPLOY_SHA" "$WORK/linuxdeploy"
fetch "$GTK_PLUGIN_URL" "$GTK_PLUGIN_SHA" "$WORK/linuxdeploy-plugin-gtk.sh"
fetch "$APPIMAGETOOL_URL" "$APPIMAGETOOL_SHA" "$WORK/appimagetool"

cat > "$WORK/flacidal.desktop" <<'EOF'
[Desktop Entry]
Name=FLACidal
Exec=flacidal
Icon=flacidal
Type=Application
Categories=AudioVideo;Audio;
EOF

# linuxdeploy only accepts standard icon sizes; appicon.png is 1024x1024
convert "$ICON" -resize 512x512 "$WORK/flacidal.png"

export APPIMAGE_EXTRACT_AND_RUN=1 ARCH=x86_64 DEPLOY_GTK_VERSION=3
export PATH="$WORK:$PATH"
APPDIR="$WORK/AppDir"

# libwebkit spawns its helper processes by absolute path, so linuxdeploy does not see them via NEEDED.
# Deploy each helper as an executable so its own dependencies get bundled too.
"$WORK/linuxdeploy" --appdir "$APPDIR" \
  --executable "$BIN" \
  --executable "$WEBKIT_DIR/WebKitWebProcess" \
  --executable "$WEBKIT_DIR/WebKitNetworkProcess" \
  --library "$WEBKIT_DIR/injected-bundle/libwebkit2gtkinjectedbundle.so" \
  --desktop-file "$WORK/flacidal.desktop" \
  --icon-file "$WORK/flacidal.png" \
  --plugin gtk

# WebKit needs a GIO TLS backend for https (images, fonts, media). The host modules cannot be reused because they
# are built against a newer glib than the bundled one, so ship the glib-networking module and its dependencies.
mkdir -p "$APPDIR/usr/lib/gio/modules"
cp "$GIO_TLS" "$APPDIR/usr/lib/gio/modules/"
"$WORK/linuxdeploy" --appdir "$APPDIR" --deploy-deps-only "$APPDIR/usr/lib/gio/modules"

# WebKit ignores WEBKIT_EXEC_PATH in release builds and spawns its helpers from the path compiled into
# libwebkit2gtk. Rewrite that prefix in the bundled copy to a same-length relative path (padded with
# slashes) that resolves from $APPDIR, and run the app from there (see the AppRun hook below).
mkdir -p "$APPDIR/usr/bin/injected-bundle"
mv "$APPDIR/usr/lib/libwebkit2gtkinjectedbundle.so" "$APPDIR/usr/bin/injected-bundle/"
LIBWEBKIT="$(readlink -f "$APPDIR/usr/lib/libwebkit2gtk-4.1.so.0")"
export OLD_PREFIX="$WEBKIT_DIR"
NEW_PREFIX="$(printf '%-*s' "${#OLD_PREFIX}" './usr/bin' | tr ' ' '/')"
export NEW_PREFIX
grep -qaF "$OLD_PREFIX" "$LIBWEBKIT" || { echo "helper path not found in $LIBWEBKIT" >&2; exit 1; }
perl -0777 -pi -e 's/\Q$ENV{OLD_PREFIX}\E/$ENV{NEW_PREFIX}/g' "$LIBWEBKIT"
if grep -qaF "$OLD_PREFIX" "$LIBWEBKIT"; then echo "helper path still present after patch" >&2; exit 1; fi

mkdir -p "$APPDIR/apprun-hooks"
cat > "$APPDIR/apprun-hooks/webkit-helpers.sh" <<'EOF'
cd "$APPDIR"
export GIO_EXTRA_MODULES="$APPDIR/usr/lib/gio/modules"
EOF
# linuxdeploy's generated AppRun only sources the gtk hook by name, so replace it with one that sources every hook
cat > "$APPDIR/AppRun" <<'EOF'
#! /usr/bin/env bash
set -e
this_dir="$(readlink -f "$(dirname "$0")")"
for hook in "$this_dir"/apprun-hooks/*.sh; do
  source "$hook"
done
exec "$this_dir"/AppRun.wrapped "$@"
EOF
chmod +x "$APPDIR/AppRun"

"$WORK/appimagetool" "$APPDIR" "$OUT"
echo "built $OUT ($(du -h "$OUT" | cut -f1))"
