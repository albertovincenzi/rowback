#!/usr/bin/env bash
# Build the self-contained "Rowback.app" bundle (the GUI plus the CLI it drives).
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
root="$(cd "$here/.." && pwd)"
app="$here/Rowback.app"
macos="$app/Contents/MacOS"

echo "› building CLI…"
( cd "$root" && go build -o rowback ./cmd/rowback )

echo "› building desktop GUI…"
# Prefer the true native WKWebView window (needs a working C++ toolchain).
# If that build fails (e.g. Command Line Tools missing the C++ stdlib), fall
# back to the pure-Go browser launcher so the bundle is still usable.
mode="webview (native window)"
if ! ( cd "$here" && go build -tags webview -o rowback-desktop . ) 2>/tmp/rowback-webview-build.log; then
  echo "  ⚠︎ webview build failed — falling back to the pure-Go browser launcher."
  echo "    (install Xcode Command Line Tools with a working C++ stdlib to get the native window;"
  echo "     see desktop/README.md for the fix). Build log:"
  sed 's/^/      /' /tmp/rowback-webview-build.log || true
  mode="browser launcher (pure Go fallback)"
  ( cd "$here" && go build -o rowback-desktop . )
fi
echo "  → built desktop binary in mode: $mode"

echo "› assembling bundle…"
rm -rf "$app"
mkdir -p "$macos" "$app/Contents/Resources"
cp "$here/rowback-desktop" "$macos/rowback-desktop"
# Ship the CLI inside the bundle so the GUI finds it next to itself (portable).
cp "$root/rowback" "$macos/rowback"

# App icon: regenerate the .icns from the source PNG if tooling is present,
# otherwise use the committed icon/AppIcon.icns.
if [ -f "$here/icon/icon_1024.png" ] && command -v iconutil >/dev/null 2>&1; then
  iconset="$here/icon/AppIcon.iconset"
  rm -rf "$iconset"; mkdir -p "$iconset"
  for sz in 16 32 64 128 256 512; do
    sips -z $sz $sz "$here/icon/icon_1024.png" --out "$iconset/icon_${sz}x${sz}.png" >/dev/null
    sips -z $((sz*2)) $((sz*2)) "$here/icon/icon_1024.png" --out "$iconset/icon_${sz}x${sz}@2x.png" >/dev/null
  done
  iconutil -c icns "$iconset" -o "$here/icon/AppIcon.icns"
  rm -rf "$iconset"
fi
[ -f "$here/icon/AppIcon.icns" ] && cp "$here/icon/AppIcon.icns" "$app/Contents/Resources/AppIcon.icns"

cat > "$app/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleName</key><string>Rowback</string>
  <key>CFBundleDisplayName</key><string>Rowback</string>
  <key>CFBundleIdentifier</key><string>io.github.albertovincenzi.rowback</string>
  <key>CFBundleExecutable</key><string>rowback-desktop</string>
  <key>CFBundleIconFile</key><string>AppIcon</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleVersion</key><string>1.0</string>
  <key>CFBundleShortVersionString</key><string>1.0</string>
  <key>NSHighResolutionCapable</key><true/>
  <key>LSMinimumSystemVersion</key><string>10.13</string>
</dict>
</plist>
PLIST
echo "APPL" > "$app/Contents/PkgInfo"

# The browser-launcher has no window of its own, so as a normal Dock app its
# icon bounces forever. Mark it an accessory (agent) app to stop the bouncing.
# The native WebView build DOES have a window, so it keeps a normal Dock icon.
case "$mode" in
  browser*) /usr/libexec/PlistBuddy -c "Add :LSUIElement bool true" "$app/Contents/Info.plist" >/dev/null 2>&1 || true ;;
esac

echo "✓ built: $app  (mode: $mode)"
echo "  launch with:  open \"$app\"   (or double-click in Finder)"
