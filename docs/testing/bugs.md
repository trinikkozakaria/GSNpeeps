# Already implemented
bugs
- create forms for BPJS (image or pdf) & NPWP (image or pdf), Kontak Darurat (emergency contact name and number), Pendidikan (education, school, year), Riwayat Jabatan (employee position history), Gaji bulan berjalan (current salary) on create / edit "Karyawan"
- /app/pengajuan?tab=lembur on API /api/v1/lembur?page=1&limit=10 got forbidden error
- "Foto" column on table in the page /app/live-feed, prefer to use button which shows picture in modal

# Already implemented
UI feedback
- navbar position fixed on top
- sidebar menu items position fixed on the left side
- responsive page for web browser
- if "Ajukan Lembur" / "Ajukan Ketidakhadiran" nav item is active, "Kehadiran Saya" will be active
- move "Dashboard HR" nav item before "Beranda"
- attendance will be counted as 1 if a user's check_in and check_out rows are exist
- UI color change to, Color Code: Primary: White (FFFCFB), Secondary: Blue (093FB4), Additional: Red (ED3500); Soft Pink (FFD8D8)
- font change to eloquia (attached in \home\linux\isbm\GSNpeeps\docs\source\font)

# To implement
UI feedback
- "Ajukan Ketidakhadiran" and "Ajukan Lembur" is submenu for "Pengajuan". Make dropdown or indent the label to emphasize the hierarchy. "Pengajuan" doesn't go to any page only label for parent hierarchy. This is UI only changes, use existing route page but only change the sidebar UI.
- Add form input for profile picture. Add picture on navbar next to current logged in user's name. HR and current user can update the picture