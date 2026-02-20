#!/bin/bash

# Configuration
BINARY_NAME="chrome-url-tracker"
INSTALL_PATH="$HOME/usr/local/bin/$BINARY_NAME"
PLIST_NAME="com.user.chrome-url-tracker.plist"
PLIST_PATH="$HOME/Library/LaunchAgents/$PLIST_NAME"

echo "Building $BINARY_NAME..."
go build -o "$BINARY_NAME" main.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Building native UI components..."
swiftc ui/manager.swift -o whitelist-manager

echo "Installing binary to $INSTALL_PATH..."
mkdir -p "$(dirname "$INSTALL_PATH")"
cp "$BINARY_NAME" "$INSTALL_PATH"
cp "whitelist-manager" "$(dirname "$INSTALL_PATH")/whitelist-manager"

echo "Updating LaunchAgent..."
# Ensure directory exists
mkdir -p "$HOME/Library/LaunchAgents"

# Create plist if it doesn't exist
if [ ! -f "$PLIST_PATH" ]; then
    echo "Creating new LaunchAgent plist..."
    cat <<EOF > "$PLIST_PATH"
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
fi

# Reload LaunchAgent
echo "Reloading LaunchAgent..."
launchctl unload "$PLIST_PATH" 2>/dev/null
launchctl load "$PLIST_PATH"

echo "Done! Chrome URL Tracker is updated and running."
