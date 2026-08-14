# Bug status

Last verified: **14 August 2026**.

All items in this file have been implemented. Full frontend and backend tests pass, and the
live Chromium verification for role navigation, authorization, accessibility, leave limits,
logout, and user switching passes **18/18**.

## Ringkasan status

### Sudah selesai

- Seluruh bug dan UI feedback yang tercantum di dokumen ini sudah diimplementasikan.
- Bug foto profil/Nextcloud dan respons media HTML sudah diperbaiki.
- Referensi foto HR yang yatim sudah dibersihkan; pengguna dapat mengunggah foto baru.
- Race condition logout yang sesekali menghasilkan HTTP `500` sudah diperbaiki.
- Seluruh backend test dan **208/208** frontend unit test lulus.
- Browser test live lulus **18/18** dan pengujian ulang login/logout lulus **10/10**.
- `http://localhost:8080/health` merespons HTTP `200`.

### Belum selesai / masih perlu tindakan

- Sign-off UAT manual oleh pengguna belum lengkap. Item yang belum dicentang di
  [`browser-verification-checklist.md`](./browser-verification-checklist.md) tetap harus
  diperiksa langsung di browser dan tidak otomatis dianggap lulus.
- Foto profil lama tidak dapat dipulihkan karena sebelumnya tidak pernah tersimpan di
  Nextcloud; akun terkait perlu mengunggah ulang foto satu kali.
- Hiring Progress, Recruitment Cost, dan Benefit masih berstatus **Coming Soon** karena
  requirement dan sumber datanya belum diberikan. Ketiganya bukan bug aktif.

Tidak ada bug implementasi terbuka yang masih dapat direproduksi oleh pengujian otomatis
terakhir. Bug baru yang ditemukan saat UAT manual harus ditambahkan sebagai item baru di
dokumen ini.

## Resolved bugs

- Create/edit Karyawan includes BPJS and NPWP documents, emergency contacts, education,
  position history, and current-month salary.
- `/app/pengajuan?tab=lembur` can load `/api/v1/lembur?page=1&limit=10` for an authorized
  user without an incorrect forbidden response.
- The Foto column on `/app/live-feed` uses a button and opens the protected image in a modal.
- A stale HR profile-photo reference that produced `404 /api/v1/media` was removed after the
  previous incomplete Nextcloud setup. No user file was deleted; a new profile photo can be
  uploaded normally.
- Logout finalization now detaches its short-lived session-revocation and audit work from
  browser cancellation, preventing an intermittent `500` while navigating back to login.

## Resolved UI feedback

- Navbar stays fixed at the top.
- Sidebar navigation stays fixed on the left and remains usable in compact mode.
- Pages reflow for desktop and narrow browser viewports.
- Ajukan Ketidakhadiran and Ajukan Lembur are children of the non-link Pengajuan group.
- Dashboard HR appears before Beranda for roles that may access it.
- Attendance counts as present only when check-in and check-out both exist for the same day.
- The approved white, blue, red, and soft-pink palette is applied.
- Eloquia is used from the bundled font assets.
- Profile photos can be updated by HR or the current user and appear beside the signed-in
  user's name in the navbar.
