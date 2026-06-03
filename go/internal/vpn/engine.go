package vpn

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ConnectPPTP يفحص الحسابات ويقوم بإنهاء العملية فور رصد نجاح التوثيق لمنع التعليق
func ConnectPPTP(ip, username, password string, timeout time.Duration) (bool, error) {
	uniqueID := fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Intn(100000))
	fileName := fmt.Sprintf("pptp_warrior_%s_%s", strings.ReplaceAll(ip, ".", "_"), uniqueID)
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
		return false, fmt.Errorf("failed to create options file: %v", err)
	}
	defer os.Remove(filePath)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "pppd", "call", fileName)
	
	// قراءة المخرجات بشكل حي أو عند الانتهاء معالجة الاستجابة
	output, _ := cmd.CombinedOutput()
	outStr := string(output)

	// قتل العملية فوراً إذا تجاوزت الوقت المحدد
	if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}

	// الفحص المرن والشامل لمؤشرات النجاح
	if strings.Contains(outStr, "CHAP authentication succeeded") || 
	   strings.Contains(outStr, "local  IP address") || 
	   strings.Contains(outStr, "remote IP address") || 
	   strings.Contains(outStr, "authenticated") {
		
		// قفل استباقي لعملية pppd المرتبطة بهذا الخيط لضمان عدم بقائها عالقة
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return true, nil
	}

	return false, nil
}