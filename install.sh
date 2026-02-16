#!/data/data/com.termux/files/usr/bin/bash

clear
echo "=========================================="
echo "   CF Clean IP Scanner - Installer"
echo "=========================================="
echo ""
echo "Installing for Termux (Android ARM64)"
echo ""

echo "[1/5] Installing required packages..."
pkg update -y >/dev/null 2>&1
pkg install -y golang git >/dev/null 2>&1
echo "✓ Packages installed"

echo ""
echo "[2/5] Downloading source code..."
cd ~
rm -rf CF-Clean-IP-Scanner >/dev/null 2>&1
git clone -q https://github.com/4n0nymou3/CF-Clean-IP-Scanner.git
cd CF-Clean-IP-Scanner
echo "✓ Source code downloaded"

echo ""
echo "[3/5] Downloading dependencies..."
go mod tidy >/dev/null 2>&1
echo "✓ Dependencies ready"

echo ""
echo "[4/5] Building cf-scanner..."
CGO_ENABLED=0 go build -ldflags="-s -w" -o cf-scanner >/dev/null 2>&1
echo "✓ Build completed"

echo ""
echo "[5/5] Creating shortcut..."
cat > ~/cf-scanner << 'SCRIPT'
#!/data/data/com.termux/files/usr/bin/bash
cd ~/CF-Clean-IP-Scanner
./cf-scanner "$@"
SCRIPT
chmod +x ~/cf-scanner
echo "✓ Shortcut created"

echo ""
echo "=========================================="
echo "   Installation completed successfully!"
echo "=========================================="
echo ""
echo "Usage:"
echo "  cf-scanner           # Test 500 IPs (default)"
echo "  cf-scanner 100       # Test 100 IPs (faster)"
echo "  cf-scanner 1000      # Test 1000 IPs (maximum)"
echo ""
echo "Files location:"
echo "  Program: ~/CF-Clean-IP-Scanner/"
echo "  Results: ~/CF-Clean-IP-Scanner/clean_ips.txt"
echo ""
echo "You can now run: cf-scanner"
echo ""