# 🔍 CF Clean IP Scanner

ابزار پیدا کردن IP های تمیز Cloudflare برای Termux (Android ARM64)

<div dir="rtl">

## ویژگی‌ها

- تست دقیق هزاران IP از رنج‌های رسمی Cloudflare
- طراحی و بهینه‌سازی مخصوص Termux و شرایط اینترنت ایران
- تست دو مرحله‌ای هوشمند: **TCPing × 4 بار** + **تست سرعت دانلود**
- محاسبه **نرخ Packet Loss** و **میانگین تأخیر** برای هر IP
- متوقف کردن اسکن در هر لحظه با **Ctrl+C** و نمایش نتایج یافت‌شده
- نمایش **مدت زمان اسکن** و **حجم اینترنت مصرف‌شده** در پایان
- ذخیره خودکار نتایج در فایل `clean_ips.txt`

---

## 📥 نصب

### روش اول: دانلود مستقیم (پیشنهادی)

**یک دستور ساده در Termux:**

```bash
pkg update && pkg upgrade && pkg install wget unzip && wget https://github.com/4n0nymou3/CF-Clean-IP-Scanner/releases/latest/download/cf-scanner-arm64.zip && unzip cf-scanner-arm64.zip && chmod +x cf-scanner
```

سپس اجرا کنید:

```bash
./cf-scanner
```

**مزایا:**
- ✅ سریع (حدود 30 ثانیه)
- ✅ بدون نیاز به نصب Go
- ✅ فایل آماده و کامپایل‌شده

---

### روش دوم: Build از سورس (اگر روش اول کار نکرد)

**یک دستور ساده:**

```bash
curl -sL https://raw.githubusercontent.com/4n0nymou3/CF-Clean-IP-Scanner/main/install.sh | bash
```

سپس از هرجا اجرا کنید:

```bash
cf-scanner
```

**مزایا:**
- ✅ 100% سازگار با دستگاه شما
- ✅ Build مستقیم در Termux
- ✅ نصب خودکار در PATH

**معایب:**
- زمان بیشتر (تا 10 دقیقه)
- نیاز به نصب Golang (بصورت خودکار انجام می‌گیرد)

---

## ▶️ استفاده

فقط همین یک دستور کافی است:

```bash
cf-scanner
```

یا:

```bash
./cf-scanner
```

ابزار تمام مراحل را **به‌صورت خودکار** انجام می‌دهد.

---

## ⚙️ روند کار ابزار

### مرحله 1: تست Latency

- تمام IP های رنج‌های Cloudflare (حدود 6000 عدد) تست می‌شوند
- هر IP دقیقاً **4 بار** TCPing می‌شود (روی پورت 443)
- برای هر IP محاسبه می‌شود:
  - **نرخ Packet Loss** (چند درصد از 4 پینگ موفق بوده)
  - **میانگین تأخیر** (میانگین زمان پاسخ پینگ‌های موفق)
- نتایج بر اساس **کمترین Packet Loss** و سپس **کمترین تأخیر** مرتب می‌شوند

### مرحله 2: تست سرعت دانلود

- از بین بهترین IP های مرحله اول، **10 تا IP اول** تست دانلود می‌شوند
- تست از سرور رسمی Cloudflare انجام می‌شود
- به محض یافتن 10 IP سالم، اسکن به‌طور خودکار متوقف می‌شود

### نتیجه نهایی

IP ها بر اساس **بالاترین سرعت دانلود** مرتب و نمایش داده می‌شوند.

---

## 🛑 توقف اسکن در هر لحظه

در هر لحظه‌ای که خواستید می‌توانید با فشار دادن **Ctrl+C** اسکن را متوقف کنید. ابزار تمام IP های سالم یافت‌شده تا آن لحظه را برای شما نمایش و ذخیره می‌کند.

---

## 📊 مثال خروجی

```
========================================
   CLOUDFLARE CLEAN IP SCANNER
   Find the fastest Cloudflare IPs
========================================
...:..::.::: Designed by: Anonymous :::.::..:...

Version: 1.6.0

Optimized for Iran network conditions
Press Ctrl+C at any time to stop and see results found so far.

Start latency test (Mode: TCP, Port: 443, Range: 0 ~ 9999 ms, Packet Loss: 1.00)
5956 / 5956 [--↗--] Available: 2184   2m3s

Latency test completed: 2184 responsive IPs found

Start download speed test (Minimum speed: 0.00 MB/s, Number: 10, Queue: 10)
10 / 10 [--↘--]   1m22s

Speed test completed: 10 clean IPs found

===========================================================================
                      CLEAN IPs FOUND
===========================================================================

Rank   IP Address           Sent   Received   Loss       Avg Delay      Download Speed
---------------------------------------------------------------------------
1.     188.114.97.163       4      4          0.00       241ms          1.32 MB/s
2.     190.93.246.213       4      4          0.00       212ms          1.05 MB/s
3.     190.93.244.169       4      4          0.00       201ms          1.04 MB/s
4.     190.93.245.115       4      4          0.00       189ms          0.99 MB/s
5.     173.245.49.8         4      4          0.00       244ms          0.93 MB/s
...

Results saved to clean_ips.txt

========================================
      Scan completed successfully!
========================================

  Scan Duration : 00:03:28
```

---

## 📁 فایل‌های خروجی

### روش اول (دانلود مستقیم):
- **فایل نتایج:** `clean_ips.txt` (در همان پوشه‌ای که اجرا کردید)

### روش دوم (Build از سورس):
- **برنامه:** `~/CF-Clean-IP-Scanner/`
- **فایل نتایج:** `~/CF-Clean-IP-Scanner/clean_ips.txt`

**مشاهده نتایج:**
```bash
cat clean_ips.txt
```

یا برای روش دوم:
```bash
cat ~/CF-Clean-IP-Scanner/clean_ips.txt
```

---

## ❓ سوالات متداول

### چرا IP پیدا نمی‌کنه؟

در شرایط فیلترینگ بسیار شدید، ابزار ممکن است IP سالم پیدا نکند. راه‌حل‌ها:
- در ساعات کم‌ترافیک (شب) دوباره امتحان کنید
- مطمئن شوید VPN روشن نیست

### ملاک انتخاب IP ها چیست؟

1. **اول:** کمترین Packet Loss (IP هایی که بیشترین پینگ‌ها رو پاسخ دادن)
2. **دوم:** کمترین تأخیر (میانگین زمان پاسخ)
3. **سوم:** بالاترین سرعت دانلود (در نتیجه نهایی)

### چند IP تست می‌کنه؟

- **مرحله 1:** همه IP ها (حدود 6000 عدد)
- **مرحله 2:** 10 تا از بهترین‌ها (به‌صورت خودکار)

### چقدر طول می‌کشه؟

- **مرحله 1 (پینگ):** حدود 2 تا 3 دقیقه
- **مرحله 2 (دانلود):** حدود 1 تا 2 دقیقه
- **جمعاً:** معمولاً زیر 5 دقیقه

---

## 🔧 عیب‌یابی

### خطا: Permission denied

```bash
chmod +x cf-scanner
```

### خطا: wget not found

```bash
pkg install wget
```

### خطا: unzip not found

```bash
pkg install unzip
```

### خطا: curl not found

```bash
pkg install curl
```

### خطا: command not found (روش دوم)

مطمئن شوید installer با موفقیت اجرا شده، سپس دوباره امتحان کنید:

```bash
curl -sL https://raw.githubusercontent.com/4n0nymou3/CF-Clean-IP-Scanner/main/install.sh | bash
```

### برنامه کرش می‌کنه

Termux را ریستارت کنید:

```bash
exit
```

سپس Termux را دوباره باز کنید و دوباره امتحان کنید.

### روش اول کار نمی‌کنه

از روش دوم استفاده کنید (Build از سورس) — این روش 100% کار می‌کند.

---

## 🔄 به‌روزرسانی

### روش اول (دانلود مستقیم):

```bash
rm -f cf-scanner cf-scanner-arm64.zip
wget https://github.com/4n0nymou3/CF-Clean-IP-Scanner/releases/latest/download/cf-scanner-arm64.zip
unzip cf-scanner-arm64.zip
chmod +x cf-scanner
```

### روش دوم (Build از سورس):

```bash
cd ~/CF-Clean-IP-Scanner
git pull
CGO_ENABLED=0 go build -ldflags="-s -w" -o cf-scanner
```

---

## 🗑️ حذف

### روش اول:
```bash
rm -f cf-scanner cf-scanner-arm64.zip clean_ips.txt
```

### روش دوم:
```bash
rm -rf ~/CF-Clean-IP-Scanner
rm /data/data/com.termux/files/usr/bin/cf-scanner
```

---

## 💡 نکات مهم

- ✅ تست را با VPN خاموش انجام دهید تا IP های واقعی ایران پیدا شوند
- ✅ فایل `clean_ips.txt` را برای استفاده بعدی نگه دارید
- ✅ اگر نتیجه خوب نگرفتید، در زمان دیگری دوباره تست کنید
- ✅ در هر لحظه می‌توانید با Ctrl+C اسکن را متوقف کنید
- ✅ حداقل 50 MB فضای خالی در Termux داشته باشید
- ✅ اینترنت خود را قطع نکنید تا تست کامل شود

---

## مجوز

این پروژه تحت [مجوز MIT](LICENSE) منتشر شده است — استفاده آزاد

---

## سازنده

طراحی و توسعه توسط: **Anonymous**

ارتباط: [Telegram](https://t.me/BXAMbot)

---

## حمایت از پروژه

اگر این ابزار برای شما مفید بود:

- ⭐ یک Star به repository بدهید
- آن را با دوستانتان به اشتراک بگذارید

</div>