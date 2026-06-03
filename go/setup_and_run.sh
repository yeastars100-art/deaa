#!/bin/bash

# PPTP Warrior Pro - GO Edition (v7.2)
# Unified Setup, Build & Run Script

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}==================================================${NC}"
echo -e "${GREEN}   PPTP Warrior Pro - Unified Setup & Builder     ${NC}"
echo -e "${BLUE}==================================================${NC}"

# 1. Check for Root
if [[ $EUID -ne 0 ]]; then
   echo -e "${RED}[!] This script must be run as root (sudo).${NC}"
   exit 1
fi

# 2. Install System Dependencies
echo -e "${BLUE}[*] Updating system and installing dependencies...${NC}"
apt-get update -y && apt-get install -y golang-go pptp-linux network-manager-pptp zip build-essential

# 3. Setup Project Structure
echo -e "${BLUE}[*] Setting up project structure...${NC}"
mkdir -p config results

# Create default config files if they don't exist
if [ ! -f config/users.txt ]; then
    echo -e "admin\nvpn\ntest\nroot\nuser" > config/users.txt
    echo -e "${GREEN}[+] Created default config/users.txt${NC}"
fi

if [ ! -f config/Domains.txt ]; then
    echo -e "127.0.0.1" > config/Domains.txt
    echo -e "${GREEN}[+] Created default config/Domains.txt${NC}"
fi

# 4. Build the Go Program
echo -e "${BLUE}[*] Building PPTP Warrior Go Edition...${NC}"
if [ -f cmd/pptp_warrior/main.go ]; then
    go mod tidy
    go build -o pptp_warrior cmd/pptp_warrior/main.go
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[+] Build successful!${NC}"
        chmod +x pptp_warrior
    else
        echo -e "${RED}[!] Build failed. Please check Go installation.${NC}"
        exit 1
    fi
else
    echo -e "${RED}[!] Source code not found!${NC}"
    exit 1
fi

# 5. Execution Menu
echo -e "${BLUE}==================================================${NC}"
echo -e "${GREEN}   Setup Complete! How would you like to start?   ${NC}"
echo -e "${BLUE}==================================================${NC}"
echo -e "1) Start Brute-force (using config files)"
echo -e "2) Scan ASN and start Brute-force"
echo -e "3) Exit"
read -p "Select an option [1-3]: " choice

case $choice in
    1)
        read -p "Enter number of threads [default 20]: " threads
        threads=${threads:-20}
        ./pptp_warrior -u config/users.txt -d config/Domains.txt -T $threads -v
        ;;
    2)
        read -p "Enter ASN (e.g., AS12345): " asn_val
        read -p "Enter Port to scan [default 1723]: " asn_port
        asn_port=${asn_port:-1723}
        ./pptp_warrior -asn $asn_val -P $asn_port -T 50 -v
        ;;
    3)
        echo -e "${GREEN}[*] Exiting. You can run the program later using: sudo ./pptp_warrior${NC}"
        exit 0
        ;;
    *)
        echo -e "${RED}[!] Invalid choice.${NC}"
        ;;
esac
