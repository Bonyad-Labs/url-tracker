#!/bin/bash

# Configuration
APP_NAME="ChromeURLTracker.app"
APP_DIR="$HOME/Applications/$APP_NAME"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

BINARY_NAME="chrome-url-tracker"
INSTALL_PATH="$MACOS_DIR/$BINARY_NAME"

PLIST_NAME="com.user.chrome-url-tracker.plist"
LAUNCHAGENT_PATH="$HOME/Library/LaunchAgents/$PLIST_NAME"

echo "Building $BINARY_NAME..."
go build -o "$BINARY_NAME" main.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Building native UI components..."
swiftc ui/*.swift ui/Views/*.swift -o whitelist-manager

echo "Creating .app bundle structure..."
mkdir -p "$MACOS_DIR"
mkdir -p "$RESOURCES_DIR"

echo "Installing binaries to $MACOS_DIR..."
cp "$BINARY_NAME" "$INSTALL_PATH"
cp "whitelist-manager" "$MACOS_DIR/whitelist-manager"

echo "Creating Info.plist..."
cat <<EOF > "$CONTENTS_DIR/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>$BINARY_NAME</string>
    <key>CFBundleIdentifier</key>
    <string>com.user.chrome-url-tracker</string>
    <key>CFBundleName</key>
    <string>ChromeURLTracker</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>LSUIElement</key>
    <true/>
</dict>
</plist>
EOF

echo "Signing the application bundle..."
codesign --force --deep --sign - "$APP_DIR"

echo "Updating LaunchAgent..."
# Ensure directory exists
mkdir -p "$HOME/Library/LaunchAgents"

# Create plist if it doesn't exist
echo "Creating new LaunchAgent plist..."
cat <<EOF > "$LAUNCHAGENT_PATH"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.chrome-url-tracker</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_PATH</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$HOME/Library/Logs/chrome-url-tracker.log</string>
    <key>StandardErrorPath</key>
    <string>$HOME/Library/Logs/chrome-url-tracker-error.log</string>
</dict>
</plist>
EOF

# Reload LaunchAgent
echo "Reloading LaunchAgent..."
launchctl unload "$LAUNCHAGENT_PATH" 2>/dev/null
launchctl load "$LAUNCHAGENT_PATH"

echo "Done! Chrome URL Tracker is updated and running."
