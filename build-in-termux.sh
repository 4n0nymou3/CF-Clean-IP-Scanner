#!/data/data/com.termux/files/usr/bin/bash

echo "=========================================="
echo "   CF Clean IP Scanner - Build Script"
echo "=========================================="
echo ""

echo "Installing required packages..."
pkg update -y
pkg install -y golang git

echo ""
echo "Cloning repository..."
cd ~
rm -rf CF-Clean-IP-Scanner
git clone https://github.com/4n0nymou3/CF-Clean-IP-Scanner.git
cd CF-Clean-IP-Scanner

echo ""
echo "Downloading dependencies..."
go mod download

echo ""
echo "Building cf-scanner..."
go build -ldflags="-s -w" -o cf-scanner

echo ""
echo "Verifying build..."
file cf-scanner
ls -lh cf-scanner

echo ""
echo "=========================================="
echo "   Build completed successfully!"
echo "=========================================="
echo ""
echo "You can now run:"
echo "  cd ~/CF-Clean-IP-Scanner"
echo "  ./cf-scanner"
echo ""