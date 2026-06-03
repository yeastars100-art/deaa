package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"pptp_warrior_go/internal/scanner"
)

func main() {
	threadsFlag := flag.Int("T", 50, "Number of concurrent VPN threads")
	timeoutFlag := flag.Int("t", 5, "Connection timeout in seconds")
	domainsFile := flag.String("d", "config/Domains.txt", "Path to domains/IPs file")
	usersFile := flag.String("u", "config/users.txt", "Path to users file")

	flag.Parse()

	timeout := time.Duration(*timeoutFlag) * time.Second

	if _, err := os.Stat(*domainsFile); os.IsNotExist(err) {
		_ = os.MkdirAll("config", 0755)
		_ = os.WriteFile(*domainsFile, []byte("151.71.92.0/24\n"), 0644)
		fmt.Printf("[!] Created default config file at %s\n", *domainsFile)
	}

	ipsToScan, err := localExpandTargets(*domainsFile)
	if err != nil || len(ipsToScan) == 0 {
		log.Fatalf("[-] Error reading or parsing targets from %s", *domainsFile)
	}

	pptpScanner, err := scanner.NewScanner(*threadsFlag, timeout, *usersFile)
	if err != nil {
		log.Fatalf("[-] Failed to initialize scanner: %v", err)
	}

	startTime := time.Now()
	pptpScanner.ScanIPs(ipsToScan)
	
	fmt.Printf("[+] Session completed in %v.\n", time.Since(startTime))
}

func localExpandTargets(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var ips []string
	fileScanner := bufio.NewScanner(file)
	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.Contains(line, "/") {
			ip, ipNet, err := net.ParseCIDR(line)
			if err != nil {
				continue
			}
			for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); inc(ip) {
				ips = append(ips, ip.String())
			}
		} else {
			if net.ParseIP(line) != nil {
				ips = append(ips, line)
			}
		}
	}
	return ips, fileScanner.Err()
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}