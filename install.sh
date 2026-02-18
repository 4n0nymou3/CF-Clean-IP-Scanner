#!/data/data/com.termux/files/usr/bin/bash

set -e

clear
echo "=========================================="
echo "   CF Clean IP Scanner - Installer"
echo "=========================================="
echo ""
echo "Installing for Termux (Android ARM64)"
echo ""

echo "[1/5] Checking and installing packages..."
if ! command -v git &> /dev/null; then
    echo "  → Installing git..."
    pkg install -y git || { echo "✗ Failed to install git"; exit 1; }
fi
if ! command -v go &> /dev/null; then
    echo "  → Installing golang..."
    pkg install -y golang || { echo "✗ Failed to install golang"; exit 1; }
fi
echo "✓ All packages ready"

echo ""
echo "[2/5] Downloading source code..."
cd ~
if [ -d "CF-Clean-IP-Scanner" ]; then
    echo "  → Removing old installation..."
    rm -rf CF-Clean-IP-Scanner
fi
git clone -q https://github.com/4n0nymou3/CF-Clean-IP-Scanner.git || { echo "✗ Failed to clone repository"; exit 1; }
cd CF-Clean-IP-Scanner || { echo "✗ Directory not found"; exit 1; }
echo "✓ Source code downloaded"

echo ""
echo "[3/5] Downloading dependencies..."
go mod tidy || { echo "✗ Failed to download dependencies"; exit 1; }
echo "✓ Dependencies ready"

echo ""
echo "[4/5] Building cf-scanner..."
echo "  (This may take 1-2 minutes...)"
CGO_ENABLED=0 go build -ldflags="-s -w" -o cf-scanner || { echo "✗ Build failed"; exit 1; }
if [ ! -f "cf-scanner" ]; then
    echo "✗ Build failed - executable not created"
    exit 1
fi
echo "✓ Build completed"

echo ""
echo "[5/5] Installing to system..."
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
echo "  Results saved to: ~/CF-Clean-IP-Scanner/clean_ips.txt"
echo ""
echo "You can now run: cf-scanner"
echo ""