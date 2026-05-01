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
if ! command -v wget &> /dev/null; then
    echo "  → Installing wget..."
    pkg install -y wget || { echo "✗ Failed to install wget"; exit 1; }
fi
if ! command -v unzip &> /dev/null; then
    echo "  → Installing unzip..."
    pkg install -y unzip || { echo "✗ Failed to install unzip"; exit 1; }
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
echo "[4/6] Downloading Xray core..."
mkdir -p config
wget https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-arm64-v8a.zip -O xray.zip
unzip -o xray.zip xray -d .
chmod +x xray
mv xray config/xray || true
rm -f xray.zip

cat > config/xray_config.json << 'EOF'
{
  "inbounds": [
    {
      "port": 10808,
      "protocol": "socks",
      "settings": {
        "udp": true
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom"
    }
  ]
}
EOF
echo "✓ Xray core and config ready"

echo ""
echo "[5/6] Building cf-scanner..."
echo "  (This may take 1-2 minutes...)"
CGO_ENABLED=0 go build -ldflags="-s -w" -o cf-scanner || { echo "✗ Build failed"; exit 1; }
if [ ! -f "cf-scanner" ]; then
    echo "✗ Build failed - executable not created"
    exit 1
fi
echo "✓ Build completed"

echo ""
echo "[6/6] Installing to system..."
cat > $PREFIX/bin/cf-scanner << 'SCRIPT'
#!/data/data/com.termux/files/usr/bin/bash
cd ~/CF-Clean-IP-Scanner
./cf-scanner "$@"
SCRIPT
chmod +x $PREFIX/bin/cf-scanner
echo "✓ Installation successful!"