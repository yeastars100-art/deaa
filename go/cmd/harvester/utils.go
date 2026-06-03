package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func runAdvancedNmapScan(subnet string, ifaceName string, targetDir string) {
	fmt.Printf("[*] Launching Deep Nmap Scan on %s via %s...\n", subnet, ifaceName)
	
	rawOutputFile := fmt.Sprintf("%s/nmap_raw.txt", targetDir)
	tableOutputFile := fmt.Sprintf("%s/internal_devices.txt", targetDir)
	sipOutputFile := fmt.Sprintf("%s/sipvicious_raw.txt", targetDir)

	// 1. تشغيل Nmap للفحص العام
	nmapCmd := exec.Command("sudo", "nmap", 
		"-p", "23,80,443,5060,8080", 
		"--script", "http-title", 
		"-e", ifaceName, 
		subnet,
	)
	nmapOutput, _ := nmapCmd.CombinedOutput()
	_ = os.WriteFile(rawOutputFile, nmapOutput, 0644)

	// 2. تشغيل SIPVicious للفحص المتخصص على بورت 5060
	fmt.Printf("[*] Launching SIPVicious (svmap) on %s...\n", subnet)
	sipCmd := exec.Command("python3", "-m", "sipvicious.svmap", 
		"-p", "5060",
		subnet,
	)
	sipOutput, _ := sipCmd.CombinedOutput()
	_ = os.WriteFile(sipOutputFile, sipOutput, 0644)
	
	// 3. معالجة البيانات وبناء الجدول لحفظه وعرضه
	generateAndSaveTable(string(nmapOutput), string(sipOutput), tableOutputFile)
}

func generateAndSaveTable(nmapOutput string, sipOutput string, savePath string) {
	type DeviceInfo struct {
		Ports  []string
		Titles []string
		SIP    string
	}

	devices := make(map[string]*DeviceInfo)
	
	// معالجة مخرجات Nmap
	scanner := bufio.NewScanner(strings.NewReader(nmapOutput))
	var currentIP string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "Nmap scan report for") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				currentIP = fields[4]
				if _, exists := devices[currentIP]; !exists {
					devices[currentIP] = &DeviceInfo{}
				}
			}
		}
		if currentIP != "" {
			if strings.Contains(line, "open") && !strings.Contains(line, "|") {
				fields := strings.Fields(line)
				if len(fields) >= 1 {
					devices[currentIP].Ports = append(devices[currentIP].Ports, fields[0])
				}
			}
			if strings.Contains(line, "http-title:") {
				titleVal := strings.TrimSpace(line[strings.Index(line, "http-title:")+len("http-title:"):])
				if titleVal != "" {
					devices[currentIP].Titles = append(devices[currentIP].Titles, titleVal)
				}
			}
		}
	}

	// معالجة مخرجات SIPVicious
	sipScanner := bufio.NewScanner(strings.NewReader(sipOutput))
	for sipScanner.Scan() {
		line := strings.TrimSpace(sipScanner.Text())
		if strings.Contains(line, "|") && (strings.Contains(line, "SIP") || strings.Contains(line, "User-Agent")) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				sipIP := fields[1]
				if _, exists := devices[sipIP]; !exists {
					devices[sipIP] = &DeviceInfo{Ports: []string{"5060/udp"}}
				}
				// محاولة استخراج User-Agent إذا وجد
				if strings.Contains(line, "User-Agent:") {
					ua := line[strings.Index(line, "User-Agent:")+len("User-Agent:"): ]
					devices[sipIP].SIP = strings.TrimSpace(ua)
				} else {
					devices[sipIP].SIP = "SIP Device Detected"
				}
			}
		}
	}

	if len(devices) == 0 {
		fmt.Println("   [-] No active services found in this subnet scan.")
		return
	}

	f, err := os.OpenFile(savePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	var tableLines []string
	tableLines = append(tableLines, "   +-----------------+--------+--------------------+-------------------------------------------+")
	tableLines = append(tableLines, "   |   IP Address    | Status |     Open Ports     |              Device / Title               |")
	tableLines = append(tableLines, "   +-----------------+--------+--------------------+-------------------------------------------+")

	for ip, info := range devices {
		portsStr := strings.Join(info.Ports, ", ")
		titleStr := "Unknown Device"
		
		if info.SIP != "" {
			titleStr = "[SIP] " + info.SIP
		} else if len(info.Titles) > 0 {
			titleStr = info.Titles[0]
		}

		if (titleStr == "Unknown Device" || strings.Contains(titleStr, "doesn't have a title")) {
			for _, p := range info.Ports {
				if strings.Contains(p, "5060") {
					titleStr = "VoIP / SIP Gateway"
					break
				} else if strings.Contains(p, "23") {
					titleStr = "Telnet Access Enabled"
				}
			}
		}

		if len(titleStr) > 40 { titleStr = titleStr[:37] + "..." }
		if len(portsStr) > 18 { portsStr = portsStr[:15] + "..." }

		line := fmt.Sprintf("   | %-15s |  %-4s  | %-18s | %-41s |", ip, "ALIVE", portsStr, titleStr)
		tableLines = append(tableLines, line)
	}
	tableLines = append(tableLines, "   +-----------------+--------+--------------------+-------------------------------------------+")

	for _, l := range tableLines {
		fmt.Println(l)
		_, _ = f.WriteString(l + "\n")
	}
	fmt.Printf("   [+] Clean table dashboard saved to %s\n", savePath)
}

func calculateSubnet(ipStr string) []string {
	ipStr = strings.TrimSpace(ipStr)
	segments := strings.Split(ipStr, ".")
	if len(segments) < 4 {
		return []string{"10.0.0.0/24"}
	}
	
	// إضافة الرينج المكتشف مباشرة /24
	subnets := []string{fmt.Sprintf("%s.%s.%s.0/24", segments[0], segments[1], segments[2])}
	
	// إضافة رينجات شائعة أخرى بناءً على الفئة
	if segments[0] == "10" {
		subnets = append(subnets, "10.0.0.0/16")
	} else if segments[0] == "192" && segments[1] == "168" {
		subnets = append(subnets, "192.168.0.0/16")
	} else if segments[0] == "172" {
		subnets = append(subnets, "172.16.0.0/12")
	}
	
	return subnets
}

func saveTargetDetails(ip, user, pass, local, remote string, subnets []string) {
	targetDir := fmt.Sprintf("results/%s", ip)
	_ = os.MkdirAll(targetDir, 0755)

	filePath := fmt.Sprintf("%s/network_info.txt", targetDir)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	subnetStr := strings.Join(subnets, ", ")
	// تنسيق مخصص بناءً على طلب المستخدم: IP الشبكة، الحساب، والآيبي الداخلي
	content := fmt.Sprintf("========================================\n")
	content += fmt.Sprintf("PPTP NETWORK DETAILS\n")
	content += fmt.Sprintf("========================================\n")
	content += fmt.Sprintf("Public IP: %s\n", ip)
	content += fmt.Sprintf("Credentials: %s:%s\n", user, pass)
	content += fmt.Sprintf("Internal Local IP: %s\n", local)
	content += fmt.Sprintf("Internal Gateway IP: %s\n", remote)
	content += fmt.Sprintf("Discovered Subnets: %s\n", subnetStr)
	content += fmt.Sprintf("Harvest Date: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	content += fmt.Sprintf("========================================\n")
	
	f.WriteString(content)
}

func parseValue(line, key string) string {
	idx := strings.Index(line, key)
	if idx == -1 {
		return ""
	}
	sub := line[idx+len(key):]
	end := strings.Index(sub, "|")
	if end != -1 {
		sub = sub[:end]
	}
	return strings.TrimSpace(sub)
}