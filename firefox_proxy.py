import argparse
import json
import os
import tempfile
import zipfile
from pathlib import Path
from urllib.parse import quote, urlparse

from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.firefox.service import Service
from webdriver_manager.firefox import GeckoDriverManager

DEFAULT_URL = "https://api.ipify.org"
DEFAULT_SCREENSHOT = "firefox_proxy.png"


def proxy_config() -> tuple[str, int, str, str]:
    proxy_url = os.environ.get("PROXY_URL")
    if proxy_url:
        parsed = urlparse(proxy_url)
        if not all((parsed.hostname, parsed.port, parsed.username, parsed.password)):
            raise ValueError("PROXY_URL must include scheme, username, password, host, and port")
        return parsed.hostname, parsed.port, parsed.username, parsed.password

    required = ("PROXY_HOST", "PROXY_PORT", "PROXY_USERNAME", "PROXY_PASSWORD")
    missing = [name for name in required if not os.environ.get(name)]
    if missing:
        raise ValueError(f"Missing proxy environment variables: {', '.join(missing)}")
    return (
        os.environ["PROXY_HOST"],
        int(os.environ["PROXY_PORT"]),
        os.environ["PROXY_USERNAME"],
        os.environ["PROXY_PASSWORD"],
    )


def authenticated_proxy_url(host: str, port: int, username: str, password: str) -> str:
    return f"http://{quote(username, safe='')}:{quote(password, safe='')}@{host}:{port}"


def firefox_options(headless: bool) -> webdriver.FirefoxOptions:
    options = webdriver.FirefoxOptions()
    if headless:
        options.add_argument("-headless")
    return options


def create_proxy_addon(directory: Path, host: str, port: int, username: str, password: str) -> Path:
    manifest = {
        "manifest_version": 2,
        "name": "Temporary authenticated proxy",
        "version": "1.0",
        "permissions": ["proxy", "webRequest", "webRequestBlocking", "<all_urls>"],
        "background": {"scripts": ["background.js"]},
    }
    background = f"""
const proxyInfo = {{type: "http", host: {json.dumps(host)}, port: {port}}};
browser.proxy.onRequest.addListener(() => proxyInfo, {{urls: ["<all_urls>"]}});
browser.webRequest.onAuthRequired.addListener(
  details => details.isProxy
    ? {{authCredentials: {{username: {json.dumps(username)}, password: {json.dumps(password)}}}}}
    : {{}},
  {{urls: ["<all_urls>"]}},
  ["blocking"]
);
""".strip()
    addon_path = directory / "firefox_proxy_auth.xpi"
    with zipfile.ZipFile(addon_path, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("manifest.json", json.dumps(manifest))
        archive.writestr("background.js", background)
    return addon_path


def selenium_wire_driver(options: webdriver.FirefoxOptions, proxy_url: str) -> webdriver.Firefox:
    from seleniumwire import webdriver as wire_webdriver

    return wire_webdriver.Firefox(
        service=Service(GeckoDriverManager().install()),
        options=options,
        seleniumwire_options={
            "proxy": {
                "http": proxy_url,
                "https": proxy_url,
                "no_proxy": "localhost,127.0.0.1",
            }
        },
    )


def addon_driver(
    options: webdriver.FirefoxOptions,
    addon_directory: Path,
    host: str,
    port: int,
    username: str,
    password: str,
) -> webdriver.Firefox:
    driver = webdriver.Firefox(service=Service(GeckoDriverManager().install()), options=options)
    addon_path = create_proxy_addon(addon_directory, host, port, username, password)
    driver.install_addon(str(addon_path), temporary=True)
    return driver


def main() -> int:
    parser = argparse.ArgumentParser(description="Verify an authenticated Firefox proxy with ipify")
    parser.add_argument("--url", default=DEFAULT_URL)
    parser.add_argument("--expected-ip", default=os.environ.get("PROXY_EXPECTED_IP"))
    parser.add_argument("--screenshot", default=DEFAULT_SCREENSHOT)
    parser.add_argument("--headed", action="store_true", help="Show the Firefox window")
    parser.add_argument("--addon-only", action="store_true", help="Skip selenium-wire and use the temporary add-on")
    args = parser.parse_args()

    host, port, username, password = proxy_config()
    expected_ip = args.expected_ip or host
    options = firefox_options(not args.headed)
    proxy_url = authenticated_proxy_url(host, port, username, password)

    driver = None
    mode = "temporary Firefox add-on"
    with tempfile.TemporaryDirectory(prefix="firefox-proxy-") as temp_directory:
        if not args.addon_only:
            try:
                driver = selenium_wire_driver(options, proxy_url)
                mode = "selenium-wire"
            except Exception as error:
                print(f"selenium-wire unavailable ({type(error).__name__}); using the add-on fallback")

        try:
            if driver is None:
                driver = addon_driver(options, Path(temp_directory), host, port, username, password)
            driver.set_page_load_timeout(45)
            driver.get(args.url)
            displayed_ip = driver.find_element(By.TAG_NAME, "body").text.strip()
            driver.save_screenshot(args.screenshot)
            print(f"Proxy mode: {mode}")
            print(f"Displayed IP: {displayed_ip}")
            print(f"Screenshot: {Path(args.screenshot).resolve()}")
            if displayed_ip != expected_ip:
                raise RuntimeError(f"Proxy verification failed: expected {expected_ip}, received {displayed_ip}")
            print("Proxy verification succeeded")
            return 0
        finally:
            if driver is not None:
                driver.quit()


if __name__ == "__main__":
    raise SystemExit(main())
