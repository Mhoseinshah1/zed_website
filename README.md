# ZedProxy Website

یک وبسایت پریمیوم فارسی RTL برای سرویس پروکسی ZedProxy، ساخته شده با Go، Gin، SQLite، Tailwind CSS و Alpine.js.

## ویژگی‌ها

- 🎨 طراحی premium، dark، futuristic با glassmorphism
- 🇮🇷 کاملاً فارسی و RTL
- 📱 Mobile-first و Responsive
- ⚡ سریع و سبک
- 🔒 امنیت بالا
- 🎛️ پنل مدیریت کامل در `/zed-admin`
- 📊 ردیابی کلیک‌های تلگرام
- 🗺️ Sitemap.xml و Robots.txt
- 📰 وبلاگ و SEO Articles
- 📚 صفحه آموزش‌ها
- ❓ صفحه FAQ با schema markup
- 🌍 مدیریت لوکیشن‌ها
- 📢 اطلاعیه‌های وضعیت سرویس
- 📁 آپلود و مدیریت فایل
- 🔑 امنیت کامل (bcrypt، session، rate limit)

> **Note:** Installer terminal messages are English for compatibility with Ubuntu SSH terminals. The website and admin panel remain fully Persian RTL.

## نصب سریع

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/install.sh)
```

## بروزرسانی سایت

پس از نصب، برای بروزرسانی به آخرین نسخه:

```bash
sudo bash /opt/zedproxy/update.sh
```

این دستور:
- نسخه پشتیبان از دیتابیس می‌گیرد
- آخرین کد را از GitHub دریافت می‌کند
- باینری، قالب‌ها و فایل‌های استاتیک را بروزرسانی می‌کند
- دیتابیس، آپلودها و `.env` را دست نمی‌زند
- سرویس را ری‌استارت و سلامت آن را بررسی می‌کند
- لاگ بروزرسانی در `/opt/zedproxy/logs/update-YYYYMMDD-HHMMSS.log` ذخیره می‌شود

## بازیابی update.sh

اگر `update.sh` در دسترس نبود:

```bash
sudo curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh \
  -o /opt/zedproxy/update.sh
sudo chmod +x /opt/zedproxy/update.sh
sudo bash /opt/zedproxy/update.sh
```

## مدیریت حالت تعمیر از CLI

```bash
# فعال کردن حالت تعمیر
sudo /opt/zedproxy/zedproxy --db=/opt/zedproxy/data/zedproxy.db --maintenance-on

# غیرفعال کردن حالت تعمیر
sudo /opt/zedproxy/zedproxy --db=/opt/zedproxy/data/zedproxy.db --maintenance-off

# بررسی وضعیت حالت تعمیر
sudo /opt/zedproxy/zedproxy --db=/opt/zedproxy/data/zedproxy.db --maintenance-status
```

## تست سیستم

```bash
sudo /opt/zedproxy/zedproxy --db=/opt/zedproxy/data/zedproxy.db \
  --templates=/opt/zedproxy/templates --static=/opt/zedproxy/static \
  --uploads=/opt/zedproxy/static/uploads --self-test
```

## Rollback به نسخه قبلی

```bash
sudo bash /opt/zedproxy/rollback.sh
```

## نصب دستی

### پیش‌نیازها
- Ubuntu 20.04/22.04/24.04
- Go 1.22+
- Nginx
- SQLite3
- gcc (برای go-sqlite3)

### مراحل

```bash
# 1. کلون پروژه
git clone https://github.com/mhoseinshah1/zed_website.git
cd zed_website

# 2. دانلود وابستگی‌ها
go mod download

# 3. بیلد
VERSION=2.1.0
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date +%Y-%m-%d)
CGO_ENABLED=1 go build \
  -ldflags="-s -w -X main.Version=$VERSION -X main.BuildDate=$DATE -X main.GitCommit=$COMMIT" \
  -o zedproxy .

# 4. مقداردهی اولیه دیتابیس
./zedproxy --seed --admin-user=admin --admin-email=admin@yourdomain.com --admin-pass=YOUR_PASS --secret=YOUR_SECRET

# 5. اجرا
./zedproxy --addr=:8080 --secret=YOUR_SECRET
```

## ساختار پروژه

```
zed_website/
├── main.go                    # نقطه ورود برنامه
├── go.mod
├── internal/
│   ├── database/database.go   # اتصال SQLite و migrations
│   ├── models/models.go       # مدل‌های داده
│   ├── handlers/
│   │   ├── helpers.go         # توابع کمکی قالب
│   │   ├── public.go          # هندلرهای صفحات عمومی
│   │   └── admin.go           # هندلرهای پنل مدیریت
│   ├── middleware/auth.go     # احراز هویت و rate limiting
│   └── seed/seed.go          # داده‌های اولیه پارسی
├── templates/
│   ├── layouts/
│   │   ├── base.html          # قالب اصلی صفحات عمومی
│   │   └── admin.html         # قالب پنل مدیریت
│   ├── public/               # صفحات عمومی
│   └── admin/                # صفحات پنل مدیریت
├── static/
│   └── uploads/              # فایل‌های آپلود شده
├── install.sh                # اسکریپت نصب خودکار
├── zedproxy.service           # فایل systemd
└── nginx.conf.template        # نمونه تنظیمات Nginx
```

## پنل مدیریت

آدرس: `/zed-admin`

### بخش‌های قابل مدیریت:
- **تنظیمات سایت**: نام، لوگو، شعار، رنگ‌ها، لینک‌های تلگرام، SEO
- **پلن‌ها**: افزودن، ویرایش، حذف پلن‌های سرویس
- **ویژگی‌ها**: ویژگی‌های نمایش داده شده در صفحه اصلی
- **سوالات متداول**: FAQ با دسته‌بندی
- **مقالات**: وبلاگ با SEO کامل
- **آموزش‌ها**: راهنمای نصب برای پلتفرم‌های مختلف
- **لوکیشن‌ها**: سرورهای موجود
- **اطلاعیه‌ها**: وضعیت سرویس
- **صفحات قانونی**: شرایط استفاده و حریم خصوصی
- **مدیریت فایل**: آپلود تصاویر
- **آمار کلیک**: تعداد کلیک‌های دکمه‌های تلگرام

## متغیرهای محیطی

| متغیر | توضیح | پیش‌فرض |
|-------|-------|---------|
| `SESSION_SECRET` | رمز رمزگذاری session | - |

## پارامترهای اجرا

| پارامتر | توضیح | پیش‌فرض |
|---------|-------|---------|
| `--addr` | آدرس listen | `:8080` |
| `--db` | مسیر فایل SQLite | `./data/zedproxy.db` |
| `--templates` | مسیر قالب‌ها | `./templates` |
| `--static` | مسیر فایل‌های استاتیک | `./static` |
| `--uploads` | مسیر آپلودها | `./static/uploads` |
| `--secret` | رمز session | - |
| `--dev` | حالت توسعه | `false` |
| `--seed` | مقداردهی اولیه | `false` |
| `--admin-user` | نام کاربری ادمین | `admin` |
| `--admin-email` | ایمیل ادمین | - |
| `--admin-pass` | رمز عبور ادمین | - |

## مجوز

MIT License
