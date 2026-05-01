#!/data/data/com.termux/files/usr/bin/bash

set -e

clear
echo "=========================================="
echo "   CF Clean IP Scanner - Installer"
echo "=========================================="
echo ""
echo "Installing for Termux (Android ARM64)"
echo ""

echo "[1/6] Checking and installing packages..."
if ! command -v git &> /dev/null; then
    echo "  → Installing git..."
    pkg install -y git || { echo "✗ Failed to install git"; exit 1; }
fi
if ! command -v go &> /dev/null; then
    echo "  → Installing golang..."
    pkg install -y golang || { echo "✗ Failed to install golang"; exit 1; }
fi
if ! command -v curl &> /dev/null; then
    echo "  → Installing curl..."
    pkg install -y curl
fi
if ! command -v unzip &> /dev/null; then
    echo "  → Installing unzip..."
    pkg install -y unzip
fi
echo "✓ All packages ready"

echo ""
echo "[2/6] Downloading source code..."
cd ~
if [ -d "CF-Clean-IP-Scanner" ]; then
    echo "  → Removing old installation..."
    rm -rf CF-Clean-IP-Scanner
fi
git clone -q https://github.com/4n0nymou3/CF-Clean-IP-Scanner.git || { echo "✗ Failed to clone repository"; exit 1; }
cd CF-Clean-IP-Scanner || { echo "✗ Directory not found"; exit 1; }
echo "✓ Source code downloaded"

echo ""
echo "[3/6] Downloading dependencies..."
go mod tidy || { echo "✗ Failed to download dependencies"; exit 1; }
echo "✓ Dependencies ready"

echo ""
echo "[4/6] Installing Xray core (latest stable for Android ARM64-v8a)..."

# Check if xray binary already exists (e.g., from direct zip package)
if [ -f "./xray/xray" ]; then
    echo "  → Xray binary already present, skipping download."
else
    echo "  → Downloading Xray from GitHub..."
    # Get download URL using GitHub API
    LATEST_TAG=$(curl -s https://api.github.com/repos/XTLS/Xray-core/releases/latest | grep -o '"tag_name": "[^"]*' | cut -d '"' -f 4)
    if [ -z "$LATEST_TAG" ]; then
        echo "✗ Could not determine latest Xray tag. Please check your internet connection."
        exit 1
    fi
    DOWNLOAD_URL="https://github.com/XTLS/Xray-core/releases/download/${LATEST_TAG}/Xray-android-arm64-v8a.zip"
    echo "  → Downloading from $DOWNLOAD_URL"
    curl -L -o xray-core.zip "$DOWNLOAD_URL" || { echo "✗ Failed to download Xray"; exit 1; }
    unzip -o xray-core.zip -d xray_temp || { echo "✗ Failed to unzip Xray"; exit 1; }
    mkdir -p xray
    cp xray_temp/xray xray/
    chmod +x xray/xray
    rm -rf xray_temp xray-core.zip
fi
echo "✓ Xray core installed"

echo ""
echo "[5/6] Setting up Xray config sample..."
mkdir -p config
if [ ! -f "config/xray_config.json" ]; then
    cat > config/xray_config.json << 'EOF'
{
  "log": { "loglevel": "warning" },
  "inbounds": [
    {
      "port": 1080,
      "protocol": "socks",
      "settings": { "udp": false },
      "listen": "127.0.0.1"
    }
  ],
  "outbounds": [
    {
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "IP_PLACEHOLDER",
            "port": 443,
            "users": [
              { "id": "your-uuid-here", "encryption": "none", "flow": "xtls-rprx-vision" }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "tls",
        "tlsSettings": {
          "serverName": "your-domain.com",
          "allowInsecure": false
        }
      }
    }
  ]
}
EOF
    echo "✓ Sample config created at config/xray_config.json"
    echo "  Please edit this file with your own Xray configuration before using Xray mode."
else
    echo "✓ Existing xray_config.json found, keeping it."
fi

echo ""
echo "[6/6] Building cf-scanner..."
echo "  (This may take 1-2 minutes...)"
CGO_ENABLED=0 go build -ldflags="-s -w" -o cf-scanner || { echo "✗ Build failed"; exit 1; }
if [ ! -f "cf-scanner" ]; then
    echo "✗ Build failed - executable not created"
    exit 1
fi
echo "✓ Build completed"

echo ""
echo "Installing to system..."
cat > $PREFIX/bin/cf-scanner << 'SCRIPT'
#!/data/data/com.termux/files/usr/bin/bash
cd ~/CF-Clean-IP-Scanner
./cf-scanner "$@"
SCRIPT
chmod +x $PREFIX/bin/cf-scanner
echo "✓ Installed to PATH"

echo ""
echo "=========================================="
echo "   Installation completed successfully!"
echo "=========================================="
echo ""
echo "Usage:"
echo "  cf-scanner"
echo ""
echo "  You will be asked to choose scan mode:"
echo "    1) Normal scan (TCP ping + speed test)"
echo "    2) Xray scan (uses Xray core with your config)"
echo ""
echo "  For Xray mode, edit: ~/CF-Clean-IP-Scanner/config/xray_config.json"
echo "  Results saved to: clean_ips.txt and clean_ips_list.txt"
echo ""
echo "You can now run: cf-scanner"
echo ""