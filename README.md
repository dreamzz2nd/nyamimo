# Nyamimo - Anime Stream Platform 🐱✨

Aplikasi streaming anime modern, cepat, dan ringan berbasis **Next.js & Go + HTMX**.

## 🚀 Quick Deployment Guide (Gratis 100%)

### Opsi 1: Deploy Gratis ke Vercel (Rekomendasi Utama)

Repository ini (`dreamzz2nd/nyamimo`) sudah 100% kompatibel dan siap di-deploy secara instan ke Vercel:

1. Buka [Dashboard Vercel Import](https://vercel.com/new).
2. Pilih akun GitHub **`dreamzz2nd`** dan klik **Import** pada repository **`nyamimo`**.
3. Klik tombol **Deploy**!
4. Website Nyamimo langsung live dengan domain gratis `nyamimo.vercel.app` & HTTPS SSL otomatis.

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2Fdreamzz2nd%2Fnyamimo)

---

### Opsi 2: Jalankan Backend Server Go + HTMX (Lokal / VPS)

Untuk menjalankan server Go + HTMX lokal / di VPS:

```bash
cd go-app
go build -o nyamimo-server main.go
./nyamimo-server
```

Server Go akan berjalan di `http://localhost:3000`.

---

## 🛠️ Fitur Utama Nyamimo

- 🍿 **Streaming Player Serba Guna**: Pilihan resolusi cepat & Watchdog failover otomatis ke server cadangan jika server utama bermasalah.
- 🔑 **Popup Modal Login In-Place**: Login dengan Google & Akun Nyamimo langsung di tempat tanpa *redirect* halaman.
- 📅 **Jadwal Tayang Realtime**: Jadwal rilis harian (Senin - Minggu) lengkap dengan jam rilis WIB & skor rating.
- 🔍 **Pencarian Cerdas & Riwayat**: Pencarian anime instan dengan saran populer & penyimpanan riwayat pencarian lokal.
- 📊 **Statistik Katalog Live**: Counter total koleksi anime aktif di bagian footer.

---

© 2026 **Nyamimo Anime Stream**. Developed by [@dreamzz2nd](https://github.com/dreamzz2nd).
