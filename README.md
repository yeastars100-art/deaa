# Firefox proxy verification

يشغّل `firefox_proxy.py` متصفح Firefox عبر بروكسي HTTP موثّق، ثم يفتح `https://api.ipify.org` ويتحقق من عنوان IP الظاهر ويحفظ لقطة شاشة. يستخدم `selenium-wire` أولاً، ويعود تلقائياً إلى إضافة Firefox مؤقتة عند تعذّر تشغيله.

## المتطلبات

- Python 3.10 أو أحدث
- Firefox
- `geckodriver` اختياري؛ ينزّله `webdriver-manager` تلقائياً عند الحاجة

```bash
python -m pip install -r requirements.txt
```

## التشغيل

لا تُخزّن بيانات البروكسي في المستودع. مرّرها عبر متغيرات البيئة:

```bash
export PROXY_HOST="proxy.example.com"
export PROXY_PORT="6010"
export PROXY_USERNAME="username"
export PROXY_PASSWORD="password"
export PROXY_EXPECTED_IP="203.0.113.10"
python firefox_proxy.py
```

يمكن بدلاً من ذلك استخدام `PROXY_URL` بصيغة `http://username:password@host:port`. لتجربة الإضافة المؤقتة مباشرةً استخدم `--addon-only`، ولإظهار نافذة Firefox استخدم `--headed`.

عند النجاح يطبع السكربت عنوان IP ورسالة `Proxy verification succeeded` ويحفظ الدليل في `firefox_proxy.png`.
