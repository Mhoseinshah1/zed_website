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

## نصب سریع

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/install.sh)
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
CGO_ENABLED=1 go build -o zedproxy .

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
