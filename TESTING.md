# ZedProxy — راهنمای تست و بررسی

## نصب اولیه

```bash
# روی سرور Ubuntu (به عنوان root یا با sudo):
bash <(curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/install.sh)
```

پس از نصب، اطلاعات زیر نمایش داده می‌شود:
- آدرس سایت
- آدرس پنل مدیریت (`/zed-admin`)
- نام کاربری و رمز عبور ادمین

---

## بروزرسانی سایت

```bash
sudo bash /opt/zedproxy/update.sh
```

---

## بررسی وضعیت سرویس

```bash
# وضعیت سرویس
sudo systemctl status zedproxy

# مشاهده لاگ زنده
sudo journalctl -u zedproxy -f

# ریستارت سرویس
sudo systemctl restart zedproxy
```

---

## تست Health Check

```bash
curl -s http://127.0.0.1:8080/health | python3 -m json.tool
```

خروجی موردانتظار:
```json
{
  "db": {"ok": true},
  "status": "ok",
  "timestamp": "...",
  "version": "2.0.0"
}
```

---

## تست صفحه اصلی

```bash
curl -I http://127.0.0.1:8080/
```

باید HTTP 200 برگرداند.

---

## تست صفحات عمومی

| صفحه | آدرس | نتیجه موردانتظار |
|------|------|------------------|
| صفحه اصلی | `/` | 200 OK |
| پلن‌ها | `/plans` | 200 OK |
| آموزش‌ها | `/tutorials` | 200 OK |
| سوالات متداول | `/faq` | 200 OK |
| وبلاگ | `/blog` | 200 OK |
| وضعیت سرویس | `/status` | 200 OK |
| Sitemap | `/sitemap.xml` | 200 OK |
| Robots.txt | `/robots.txt` | 200 OK |
| Health | `/health` | 200 OK (JSON) |

---

## تست پنل مدیریت

```bash
# باز کردن در مرورگر:
https://yourdomain.com/zed-admin
```

### چک‌لیست پنل مدیریت

- [ ] ورود با نام کاربری و رمز عبور
- [ ] تنظیمات سایت — ذخیره و بررسی
- [ ] پلن‌ها — افزودن، ویرایش، حذف
- [ ] کمپین‌ها — افزودن با کد تخفیف و شمارش معکوس
- [ ] صفحات فرود SEO — افزودن، بررسی `/l/slug`
- [ ] اطلاعیه‌ها — افزودن و بررسی نمایش در صفحه اصلی
- [ ] کدهای تخفیف — افزودن و بررسی نمایش در صفحه اصلی
- [ ] وضعیت سرویس‌ها — افزودن آیتم و بررسی `/status`
- [ ] کارت‌های اعتماد — افزودن و بررسی در صفحه اصلی
- [ ] مقایسه پلن‌ها — افزودن ردیف و بررسی جدول
- [ ] مدیریت بخش‌های صفحه اصلی — تغییر ترتیب/وضعیت
- [ ] پاپ‌آپ فروش — افزودن و بررسی نمایش
- [ ] آمار کلیک — بررسی داشبورد
- [ ] رسانه — آپلود تصویر، ویرایش alt text
- [ ] آموزش‌ها — افزودن با video_url
- [ ] سوالات متداول — افزودن با show_on_homepage
- [ ] مقالات وبلاگ — افزودن و بررسی در `/blog`
- [ ] ادمین‌ها — افزودن (فقط owner)
- [ ] پشتیبان — گرفتن، دانلود، حذف
- [ ] حالت تعمیر — فعال/غیرفعال

---

## تست کمپین و صفحه فرود

```bash
# کمپین (اگر slug = black-friday باشد):
curl -I https://yourdomain.com/campaign/black-friday

# صفحه فرود (اگر slug = vpn-iran باشد):
curl -I https://yourdomain.com/l/vpn-iran
```

---

## تست ردیابی کلیک تلگرام

```bash
# کلیک روی دکمه خرید (به Telegram ریدایرکت می‌شود):
curl -I "http://127.0.0.1:8080/track?page=home&source=hero&plan=gold"
```

باید HTTP 302 و redirect به telegram برگرداند.

---

## تست حالت تعمیر

1. از پنل مدیریت، حالت تعمیر را فعال کنید
2. آدرس سایت را در مرورگر باز کنید → باید صفحه تعمیر نمایش داده شود
3. آدرس `/zed-admin` → ادمین‌ها هنوز دسترسی دارند
4. حالت تعمیر را غیرفعال کنید → سایت طبیعی برمی‌گردد

---

## تست پشتیبان‌گیری

```bash
# فایل‌های پشتیبان:
ls -lh /opt/zedproxy/backups/
ls -lh /opt/zedproxy/data/backups/

# بررسی سلامت دیتابیس:
sqlite3 /opt/zedproxy/data/zedproxy.db "PRAGMA integrity_check;"
```

---

## چک‌لیست ایمنی بروزرسانی

پس از اجرای `sudo bash /opt/zedproxy/update.sh` بررسی کنید:

- [ ] دیتابیس دست نخورده: `ls -lh /opt/zedproxy/data/zedproxy.db`
- [ ] آپلودها حفظ شده: `ls /opt/zedproxy/static/uploads/`
- [ ] `.env` دست نخورده: `cat /opt/zedproxy/.env`
- [ ] رمز عبور ادمین تغییر نکرده (ورود به پنل)
- [ ] تنظیمات سایت حفظ شده (نام سایت، لینک‌های تلگرام و ...)
- [ ] پشتیبان DB ایجاد شده: `ls /opt/zedproxy/backups/`

---

## تست HEAD requests

```bash
curl -I http://127.0.0.1:8080/        # باید 200 برگرداند
curl -I http://127.0.0.1:8080/plans   # باید 200 برگرداند
curl -I http://127.0.0.1:8080/health  # باید 200 برگرداند
```

---

## تست حالت تعمیر (CLI)

```bash
# فعال کردن
sudo /opt/zedproxy/zedproxy --db=/opt/zedproxy/data/zedproxy.db --maintenance-on

# بررسی
sudo /opt/zedproxy/zedproxy --db=/opt/zedproxy/data/zedproxy.db --maintenance-status

# غیرفعال کردن
sudo /opt/zedproxy/zedproxy --db=/opt/zedproxy/data/zedproxy.db --maintenance-off
```

---

## تست Self-Test

```bash
sudo /opt/zedproxy/zedproxy \
  --db=/opt/zedproxy/data/zedproxy.db \
  --templates=/opt/zedproxy/templates \
  --static=/opt/zedproxy/static \
  --uploads=/opt/zedproxy/static/uploads \
  --self-test
```

---

## بازیابی update.sh

```bash
sudo curl -fsSL https://raw.githubusercontent.com/mhoseinshah1/zed_website/main/update.sh \
  -o /opt/zedproxy/update.sh
sudo chmod +x /opt/zedproxy/update.sh
```

---

## Rollback

```bash
sudo bash /opt/zedproxy/rollback.sh
```

---

## دستورات عیب‌یابی

```bash
# وضعیت سرویس
sudo systemctl status zedproxy

# لاگ‌های اخیر
sudo journalctl -u zedproxy -n 100 --no-pager

# لاگ زنده
sudo journalctl -u zedproxy -f

# لاگ‌های بروزرسانی
ls -lh /opt/zedproxy/logs/

# تست دستی اجرای باینری
sudo -u www-data /opt/zedproxy/zedproxy \
  --addr=127.0.0.1:8081 \
  --db=/opt/zedproxy/data/zedproxy.db \
  --templates=/opt/zedproxy/templates \
  --static=/opt/zedproxy/static \
  --uploads=/opt/zedproxy/static/uploads \
  --secret=test_secret_only

# بررسی فایروال
sudo ufw status

# بررسی Nginx
sudo nginx -t
sudo systemctl status nginx

# سلامت سیستم (پنل ادمین)
# آدرس: /zed-admin/system/health
# لاگ‌های سیستم: /zed-admin/system/logs
```
