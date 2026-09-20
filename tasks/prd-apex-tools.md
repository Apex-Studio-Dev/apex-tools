# PRD: apex-tools — Custom Android SDK Manager & Component Builder

Status: Draft v1
Date: 2026-09-18
Complexity: Level 4 (new product)
Owner: HomuHomu833 (assumed)

---

## 1. Introduction / Overview

**apex-tools** adalah proyek baru yang menggantikan Google `cmdline-tools` untuk ekosistem SDK Android kustom. Proyek ini terdiri dari dua bagian yang berjalan dalam satu repositori:

1. **CLI sdkmanager** (`apex`): pengelola paket SDK Android yang berfungsi seperti `sdkmanager` milik Google — `--list`, `--install`, `--uninstall`, `--licenses`, `--update` — tetapi fetching dari repository rilis milik sendiri (GitHub Releases), dan menggunakan lisensi milik sendiri.
2. **Pipeline builder CI**: membangun dan mempackage seluruh komponen SDK Android (platforms, build-tools, platform-tools, cmake, ndk) untuk host **aarch64** dan **arm**, lalu menerbitkannya sebagai rilis yang dapat di-fetch oleh CLI.

Motivasi inti: SDK resmi Google hanya menyediakan binari host `x86_64`. Pengguna perangkat aarch64/arm (mis. Termux di Android, server ARM) tidak dapat menggunakan cmdline-tools resmi. Proyek-proyek pendukung yang sudah teruji sudah ada dan menjadi fondasi:

- `android-sdk-custom` — rebuild host-tools AOSP (adb, fastboot, aapt2, aidl, zipalign, dll) untuk banyak target; artefak `android-sdk-$TARGET.tar.xz`.
- `android-ndk-custom` — NDK kustom r25–r30 untuk banyak host; artefak `android-ndk-rXX[-rev]-$TARGET.tar.xz`; sudah menghasilkan `package.xml` kompatibel sdkmanager.
- `cmake-custom` — CMake + Ninja statis cross-built; artefak `cmake-$TARGET.tar.xz`.

apex-tools **tidak menduplikasi** pipeline build ketiganya; ia mengonsumsi artefak rilisnya dan menambahkan lapisan: (a) packaging ke layout paket SDK, (b) manifest/registry versi, (c) CLI sdkmanager, (d) CI orkestrasi.

## 2. Problem Statement

- **Host aarch64/arm tidak terlayani.** cmdline-tools resmi Google berjalan di JVM x86_64-friendly dan seluruh paket SDK (platform-tools, build-tools, cmake, NDK) hanya tersedia sebagai binari x86_64 host. Tidak ada jalur resmi untuk menginstal SDK di host aarch64/arm.
- **Gradle/AGP bergantung pada sdkmanager.** Build aplikasi berbasis Gradle membaca struktur SDK (`platforms/`, `build-tools/`, `platform-tools/`, `ndk/`, `cmake/`) dan file `licenses/`. Tanpa sdkmanager yang kompatibel, pengguna harus menginstal manual.
- **Tidak ada sumber paket tunggal.** Artefak sudah dibangun per proyek (sdk/ndk/cmake-custom), tetapi belum ada yang menyatukannya sebagai katalog versi + installer + lisensi.

## 3. Goals

1. CLI `apex` yang fungsional setara `sdkmanager` untuk: list, install, uninstall, licenses, update — kompatibel dengan struktur direktori SDK yang dibaca Gradle/AGP.
2. Katalog paket lengkap untuk **platforms;android-30 s.d. android-37.2**, **build-tools;33.0.0–37.0.0**, **platform-tools**, **cmake;3.18.1–4.1.2** (daftar versi di §14), **ndk;26.x–30.x** (daftar di §14) — untuk host aarch64 dan arm (plus x86_64 sebagai bonus tanpa biaya tambahan).
3. Packaging yang membuat tiap paket dikenali sebagai paket SDK: `source.properties` + `package.xml` + `licenses/` dual-file (lisensi apex + hash kompatibilitas AGP).
4. CI builder + rilis otomatis: satu workflow dapat membangun/memupdate semua komponen dan menag rilis katalog.
5. CLI berjalan di Android aarch64 (Termux) dan Linux aarch64/arm minimal; idealnya lintas platform (macos/windows/bsd) memanfaatkan static binary.

## 4. Non-Goals / Out of Scope

- **TIDAK** membangun ulang binari dari sumber AOSP/CMake/LLVM di repo ini — konsumsi artefak dari repo saudara (reuse `android-sdk-custom`, `android-ndk-custom`, `cmake-custom`).
- **TIDAK** membuat emulator, system-images, sources, extras, NDK side-by-side r21 ke bawah, platform < android-30, build-tools < 33.0.0.
- **TIDAK** menggantikan Android Studio GUI SDK Manager.
- **TIDAK** menyediakan AVD Manager (`avdmanager`) — hanya sdkmanager-equivalent.
- **TIDAK** mensupport pembaruan incremental biner / delta patch (full package download saja, seperti Google).

## 5. Target Users

- Pengguna Termux/Android aarch64 & armv7 yang ingin build aplikasi Android dari perangkat (Gradle-based).
- Developer di server/host Linux aarch64/arm (mis. ARM VPS, Mac Apple Silicon via osxcross lineage, CI runner ARM).
- CI pipeline yang butuh SDK Android tanpa x86_64.

## 6. User Stories

### US-001: Instalasi CLI
As a Termux user, I want to install the `apex` CLI with one command (curl | sh / pkg / single binary), so that I can manage my SDK without JVM.

### US-002: List katalog
As a user, I want `apex --list` (alias `apex list`) to show available packages with installed/version status, so that I can see what is available for my host.

### US-003: Instalasi paket
As a user, I want `apex install "platforms;android-34" "build-tools;35.0.0" "platform-tools"`, so that components are downloaded from apex releases and extracted into `$ANDROID_HOME` in Gradle-compatible layout.

### US-004: Uninstall
As a user, I want `apex uninstall "cmake;3.31.6"`, so that I can free disk space cleanly (menghapus direktori paket + entry manifest lokal).

### US-005: Lisensi
As a user, I want `apex licenses` to show the apex license text and, upon acceptance, write `licenses/apex-sdk-license` **plus** the standard AGP-compatibility hashes into `$ANDROID_HOME/licenses/`, so that Gradle builds proceed without license errors.

### US-006: Update
As a user, I want `apex update` to detect installed packages with newer versions available and upgrade them, so that my SDK stays current.

### US-007: Bootstrapped fresh SDK
As a user, I want `apex install platform-tools` on an empty `$ANDROID_HOME` to create the whole tree including `licenses/`, so that no manual mkdir is needed.

### US-008: Build aplikasi Gradle end-to-end
As a developer, I want a freshly apex-managed SDK (`platforms;android-34` + `build-tools;35.0.0` + `platform-tools`) to complete a Gradle `assembleDebug` on aarch64 host, so that the SDK is drop-in for real projects.

### US-009: NDK package install
As a user, I want `apex install "ndk;27.2.12479018"` to install the custom NDK into `$ANDROID_HOME/ndk/27.2.12479018` with `package.xml` recognized, so that AGP externalNativeBuild works.

### US-010: CI catalog publish
As a maintainer, I want a CI workflow that regenerates the package catalog (manifest listing all packages, versions, URLs, sizes, checksums) after component releases, so that the CLI always has an up-to-date source of truth.

### US-011: Component builder workflows
As a maintainer, I want per-component workflows (platforms, build-tools, platform-tools, cmake, ndk) that produce SDK-layout packages for both `aarch64` and `arm` host triples, so that new versions only need a re-run.

## 7. Acceptance Criteria

### AC-US-001 (Instalasi CLI)
- [ ] Binary `apex` untuk target `aarch64-linux-android` dan `armv7a-linux-androideabi` (bionic, static) tersedia di rilis apex-tools.
- [ ] Binary `apex` untuk `aarch64-linux-musl` / `arm-linux-musleabihf` tersedia (host Linux).
- [ ] Perintah `apex --version` mengembalikan versi CLI dan exit 0.

### AC-US-002 (List)
- [ ] `apex --list` menampilkan kolom: path paket, versi, status (installed/available/update available), ukuran.
- [ ] Output mencantumkan semua paket dari katalog (≥ jumlah item di §14) untuk host yang cocok.
- [ ] Keluar tanpa jaringan memberikan pesan error jelas dan exit != 0.

### AC-US-003 (Install)
- [ ] `apex install "platforms;android-34"` mengunduh arsip dari URL katalog, memverifikasi checksum, mengekstrak ke `$ANDROID_HOME/platforms/android-34`, dan menulis `source.properties` + `package.xml` di dalamnya.
- [ ] Instalasi idempotent: menginstal versi yang sama lagi akan skip dengan pesan (kecuali `--force`).
- [ ] Exit 0 saat sukses; exit != 0 dengan pesan bermakna saat URL 404/checksum mismatch.

### AC-US-004 (Uninstall)
- [ ] `apex uninstall "build-tools;34.0.0"` menghapus `$ANDROID_HOME/build-tools/34.0.0` dan membersihkan manifest lokal.
- [ ] Uninstall paket yang tidak terinstal → pesan "not installed", exit 0 (idempotent).

### AC-US-005 (Licenses)
- [ ] `apex licenses` menampilkan teks lisensi apex-sdk-license dan meminta konfirmasi y/n.
- [ ] Setelah accept, `$ANDROID_HOME/licenses/apex-sdk-license` berisi teks lisensi apex.
- [ ] Setelah accept, `$ANDROID_HOME/licenses/android-sdk-license` dibuat/di-append dengan hash AGP standar (setidaknya: hash untuk lisensi 2019 dan hash set untuk platform modern) sehingga AGP tidak menolak SDK.
- [ ] `--licenses` (bentuk flag Google) adalah alias dari `licenses`.

### AC-US-006 (Update)
- [ ] Dengan `build-tools;34.0.0` terinstal dan 35.0.0 tersedia di katalog, `apex update` menginstal versi lebih baru (tanpa menghapus yang lama, sesuai semantik side-by-side Google).
- [ ] `apex update` tanpa koneksi → pesan error, tidak merusak instalasi yang ada.

### AC-US-007 (Bootstrap)
- [ ] Pada `$ANDROID_HOME` kosong, `apex install platform-tools` membuat `$ANDROID_HOME/{platforms,build-tools,ndk,cmake,licenses}` bila diperlukan dan menaruh platform-tools di tempatnya.

### AC-US-008 (Gradle E2E)
- [ ] Project Gradle minimal (AGP 8.x, compileSdk 34) di host aarch64 (Termux) menyelesaikan `assembleDebug` dengan SDK yang diinstal penuh via apex (`platform-tools`, `platforms;android-34`, `build-tools;35.0.0`) dan lisensi accepted.
- [ ] Verifikasi dicatat di CI sebagai smoke test (opsional emulator-less; compile-only).

### AC-US-009 (NDK)
- [ ] `apex install "ndk;27.2.12479018"` menaruh isi arsip `android-ndk-r27d-<TARGET>.tar.xz` (top dir `android-ndk-r27d/`) ke `$ANDROID_HOME/ndk/27.2.12479018`, mempertahankan `package.xml` yang sudah dihasilkan android-ndk-custom.
- [ ] `local.properties`/AGP `ndkVersion "27.2.12479018"` dapat menemukan NDK tersebut.

### AC-US-010 (Catalog publish)
- [ ] Workflow `catalog.yml` menghasilkan `catalog.json` (+ `catalog.json.sha256`) berisi seluruh paket: path, versi, host-target, URL arsip, size, sha256.
- [ ] Catalog diterbitkan sebagai aset rilis GitHub (tag katalog, mis. `catalog-latest`) dan/atau raw file di branch.
- [ ] CLI membaca katalog dari URL yang dapat dikonfigurasi (env `APEX_CATALOG_URL`, default ke rilis apex-tools).

### AC-US-011 (Component workflows)
- [ ] Tiap komponen punya workflow yang dapat dipanggil (workflow_call) + dispatcher (workflow_dispatch) dengan input versi.
- [ ] Artefak akhir per komponen memakai skema penamaan seragam: `<component>-<version>-<hosttag>.tar.xz` (mis. `build-tools-35.0.0-linux-arm64.tar.xz`), published ke GitHub Releases apex-tools.
- [ ] Matrix minimal mencakup host `linux-arm64` (aarch64) dan `linux-arm` (armv7); x86_64 opsional ikut matrix tanpa kerja tambahan.

## 8. Platform-Specific Verification

- **CLI**: unit test + integration test lokal (mock server HTTP untuk katalog/unduhan); exit code diverifikasi.
- **Termux (bionic)**: uji manual `apex --list/install/licenses` di Termux aarch64; checklist di README.
- **Gradle E2E**: job CI opsional pada runner aarch64 (ubuntu-24.04-arm) yang menjalankan instalasi apex + build project Gradle contoh.
- **CI**: GitHub Actions; verifikasi = artefak rilis muncul + catalog.json valid (schema check).

## 9. Functional Requirements

Pemetaan: US-001→FR-001..002; US-002→FR-003..005; US-003→FR-006..010; US-004→FR-011; US-005→FR-012..014; US-006→FR-015..016; US-007→FR-017; US-009→FR-018..019; US-010→FR-020..022; US-011→FR-023..025.

- **FR-001**: CLI dirilis sebagai static binary per target: minimal `aarch64-linux-android`, `armv7a-linux-androideabi`, `aarch64-linux-musl`, `arm-linux-musleabihf`; penamaan aset `apex-cli-<TARGET>.tar.xz`.
- **FR-002**: CLI tidak menuntut JVM, Python, atau dependensi runtime eksternal (semua bundling statis).
- **FR-003**: `apex --list` / `apex list` membaca katalog (URL default dikompilasi; override via `APEX_CATALOG_URL`), memfilter paket yang tersedia untuk host saat ini (uname -m + OS), menandai yang terinstal (dari manifest lokal `$ANDROID_HOME/.apex/manifest.json`).
- **FR-004**: Katalog adalah JSON ter-version (`{"schema": 1, ...}`) berisi: package path (`platforms;android-34`), version, host tags, url, sha256, size, license id.
- **FR-005**: `apex --list --all` menampilkan paket untuk semua host, bukan hanya host saat ini.
- **FR-006**: `apex install <pkg>...` menerima satu atau lebih paket; melakukan download → verifikasi sha256 → ekstrak ke lokasi standar SDK berdasarkan prefix paket (`platforms;android-34` → `$ANDROID_HOME/platforms/android-34`; `build-tools;35.0.0` → `$ANDROID_HOME/build-tools/35.0.0`; `cmake;3.31.6` → `$ANDROID_HOME/cmake/3.31.6`; `ndk;27.2.12479018` → `$ANDROID_HOME/ndk/27.2.12479018`; `platform-tools` → `$ANDROID_HOME/platform-tools`).
- **FR-007**: Setelah ekstraksi, CLI memastikan `source.properties` dan `package.xml` ada di root paket (menulisnya bila arsip tidak menyertakan; isi dihasilkan dari metadata katalog).
- **FR-008**: CLI menolak instalasi jika lisensi belum accepted (exit non-zero dengan instruksi `apex licenses`), kecuali `--licenses-accepted` flag untuk CI.
- **FR-009**: Progress download ditampilkan (ukuran, persentase); retry otomatis ≥2 kali untuk kegagalan jaringan sementara.
- **FR-010**: Arsip yang didukung: `.tar.xz` (semua unix) dan `.7z` (windows-only paket); ekstraksi mempertahankan symlink.
- **FR-011**: `apex uninstall <pkg>` menghapus direktori paket dan entry manifest; idempotent.
- **FR-012**: `apex licenses` menampilkan lisensi apex dan menulis `$ANDROID_HOME/licenses/apex-sdk-license` setelah accept.
- **FR-013**: Saat accept, CLI juga menulis `$ANDROID_HOME/licenses/android-sdk-license` dengan hash AGP standar (dual-file), dan `android-sdk-preview-license` bila ada paket preview. Hash standar AGP yang wajib disertakan (confirmed di AGP source): `24333f8a63b6825ea9c5514f83c2829b004d1fee` dan hash lisensi SDK lain yang dipakai AGP; nilai final dikumpulkan pada implementasi dari `AGP source licenses` dan diverifikasi dengan build nyata.
- **FR-014**: package.xml tiap paket memakai `<uses-license ref="apex-sdk-license"/>` (bukan ref Google), sehingga identitas lisensi apex konsisten.
- **FR-015**: `apex update` membandingkan manifest lokal vs katalog; menginstal versi lebih baru per paket (side-by-side, tidak menghapus lama).
- **FR-016**: Manifest lokal `$ANDROID_HOME/.apex/manifest.json` mencatat paket terinstal: path, versi, sha256, tanggal, file package.xml. Format kompatibel-dipersimple; tidak perlu kompatibel dengan manifest Google.
- **FR-017**: Operasi apa pun pada `$ANDROID_HOME` kosong membuat struktur direktori dan `licenses/` terlebih dahulu.
- **FR-018**: Paket NDK di-install apa adanya dari arsip android-ndk-custom (top dir `android-ndk-rXX*/` di-strip saat ekstraksi ke `ndk/<versi-sdkmanager>/`).
- **FR-019**: Pemetaan versi NDK: nama rilis `r27d` ↔ path sdkmanager `ndk;27.2.12479018` dijembatani oleh katalog (kolom version = path sdkmanager; kolom upstream = tag rXX); CLI tidak menebak.
- **FR-020**: `scripts/gen-catalog.sh` (atau setara) membaca metadata semua komponen (source.properties dari arsip / input workflow) dan menghasilkan `catalog.json` + sha256.
- **FR-021**: Katalog diterbitkan otomatis: (a) sebagai aset rilis GitHub dengan tag `catalog`, di-update setiap perubahan; (b) fallback: commit ke branch `catalog` sebagai raw file.
- **FR-022**: Katalog menyertakan `min_cli` versi untuk forward-compat; CLI lama menolak katalog dengan schema lebih tinggi.
- **FR-023**: Workflow reusable per komponen (`make-<component>.yml`, workflow_call) + dispatcher (`make_<component>.yml`, workflow_dispatch) mengikuti pola repo saudara (matrix target, docker ghcr.io/<owner>/apex-builder, release-action).
- **FR-024**: Workflow komponen mengunduh artefak upstream (rilis android-sdk-custom / android-ndk-custom / cmake-custom), mengubah ke layout paket SDK (repack + source.properties + package.xml), lalu merilis sebagai aset apex-tools dengan penamaan `<component>-<version>-<hosttag>.tar.xz`.
- **FR-025**: Setelah rilis komponen, workflow `catalog.yml` berjalan otomatis (`workflow_run` atau dipanggil manual) untuk regenerate + publish katalog.
- **FR-026**: Env `ANDROID_HOME` (fallback `ANDROID_SDK_ROOT`) adalah root SDK; tanpa keduanya → error instruksi yang jelas.

## 10. Non-Functional Requirements

- **Binary size**: CLI ≤ 15 MB per target (static Go atau JRE-less).
- **Portability**: CLI bionic-static berjalan di Android ≥ API 24 tanpa root.
- **Reliability**: download resumable/verify sha256; tidak pernah meninggalkan direktori paket setengah-terinstal (ekstrak ke tmp dir lalu rename atomik).
- **Security**: verifikasi checksum semua arsip; tidak menyimpan kredensial; HTTPS-only untuk katalog & unduhan.
- **Maintainability**: satu bahasa CLI; skrip packaging bash mengikuti gaya repo saudara (env-driven, idempotent-sebisa-mungkin).
- **Compatibility**: struktur SDK hasil instalasi 100% mengikuti layout Google sehingga AGP/Studio/gradle plugin menemukan komponen.

## 11. UX / Design Considerations

- CLI non-interaktif by default kecuali `licenses` (butuh y/n) — semua perintah lain aman untuk CI.
- Output list dalam bentuk tabel ASCII sederhana; `--json` flag untuk konsumsi skrip.
- Pesan error: selalu sertakan next-step (mis. "run `apex licenses` to accept").
- Env override: `APEX_CATALOG_URL`, `APEX_REPO_OWNER`, `ANDROID_HOME`.

## 12. Technical Considerations

### Confirmed (dari investigasi repo saudara)
- Artefak upstream sudah ada dan teruji:
  - `android-sdk-custom` → `android-sdk-$TARGET.tar.xz` (platform-tools + build-tools 37.0.0 + tool lain; TOOLS_VERSION 37.0.0; target bionic `aarch64-linux-android`, `armv7a-linux-androideabi`; CI reusable `make-sdk.yml`).
  - `android-ndk-custom` → `android-ndk-rXX[rev]-$TARGET.tar.xz` (r25–r30; sudah include package.xml + license refs; top dir `android-ndk-rXX*/`).
  - `cmake-custom` → `cmake-$TARGET.tar.xz` (CMake 4.1.2 default + ninja 1.12.1 bundled; versi free-form input; TIDAK menyertakan source.properties/package.xml — apex-tools yang menambahkannya saat repack).
- Pola CI repo saudara: workflow_call reusable + workflow_dispatch dispatcher, docker `ghcr.io/<owner>/<name>-builder`, `ncipollo/release-action`, ubuntu-24.04 x86_64 dengan cross-compile.
- `package.xml` schema sdkmanager: namespace `http://schemas.android.com/repository/android/common/02`, `localPackage path="..."`, `genericDetailsType`, `uses-license ref`.
- NDK r30 = `30.0.x` di source.properties; mapping rXX→X.Y.Z sudah dihasilkan upstream; katalog tinggal memakainya.

### Proposed (keputusan desain apex-tools)
- **Bahasa CLI: Go** (single static binary, trivially cross-compiled ke bionic & musl & semua host, HTTP+zip/tar builtin). Go juga dipakai untuk generator katalog.
  - Catatan: user membuka opsi Go dan Java/Kotlin. Java/Kotlin hanya masuk akal bila ingin drop-in menggantikan cmdline-tools Google (butuh JVM di host — bertentangan dengan kasus Termux murni). Keputusan: Go sebagai satu-satunya CLI; Java version TIDAK dibangun (ponytail; tambah nanti bila ada kebutuhan Studio-integration).
- **Lisensi dual-file** (keputusan user): `licenses/apex-sdk-license` (teks apex) + `licenses/android-sdk-license` (hash AGP standar) agar Gradle tetap jalan; `package.xml` ref ke `apex-sdk-license`.
- **Struktur repo** (mengikuti pola saudara):
  ```
  apex-tools/
    cmd/apex/            # CLI Go
    internal/            # catalog, downloader, extractor, manifest, license
    scripts/
      gen-catalog.sh     # regenerate catalog.json dari rilis komponen
      make-package.sh    # repack arsip upstream → paket SDK layout (env-driven)
    catalog/catalog.json # (generated, committed + released)
    docker/Dockerfile    # apex-builder image (zig-as-llvm utk bionic/musl, NDK r27d)
    .github/workflows/
      make-cli.yml / make_cli*.yml
      make-platforms.yml + make_platforms.yml
      make-build-tools.yml + make_build_tools.yml
      make-platform-tools.yml + make_platform_tools.yml
      make-cmake.yml + make_cmake.yml
      make-ndk.yml + make_ndk.yml
      catalog.yml
    LICENSE              # lisensi apex (user's own)
  ```
- **Host tags**: konsisten dengan android-ndk-custom: `linux-arm64`, `linux-arm`; artifact hosttag disamakan (`linux-arm64`, `linux-arm`).
- **Repack platforms & build-tools**: `platforms;android-XX` berisi file XML/data (bukan binari) → bisa diambil dari paket resmi Google (dl.google.com) apa pun host-nya; build-tools & platform-tools **wajib** dari artefak android-sdk-custom (binari aarch64/arm). pemisahan: workflow make-platforms mengunduh zip resmi Google + repack; make-build-tools/make-platform-tools mengunduh `android-sdk-$TARGET.tar.xz` android-sdk-custom + repack per-komponen.
- **CMake packages**: dari cmake-custom (`cmake-$TARGET.tar.xz`), di-version per input workflow; `source.properties` ditulis oleh make-package.sh (`Pkg.Revision`, `Pkg.License=apex-sdk-license`).

### Unknown / Needs confirmation
- Hash lisensi AGP lengkap: perlu diekstrak dari AGP source pada implementasi (FR-013) dan diverifikasi dengan build nyata.
- Teks lisensi apex: user menyediakan sendiri (file LICENSE).
- Untuk build-tools versi lama (33.0.x): android-sdk-custom hanya membangun TOOLS_VERSION tunggal (37.0.0). Kemungkinan perlu fork/param versi di android-sdk-custom ATAU build-tools lama hanya tersedia untuk x86_64 (dari Google) sementara aarch64/arm hanya versi terbaru. **Open question OQ-1**.
- aapt2 untuk host arm32 (armv7): android-sdk-custom men-support-nya; dikonfirmasi saat E2E.

## 13. Data Requirements

- **catalog.json** (schema 1):
  ```json
  {
    "schema": 1,
    "generated": "2026-09-18T00:00:00Z",
    "min_cli": "1.0.0",
    "packages": [
      {
        "path": "platforms;android-34",
        "version": "34",
        "hosts": ["linux-arm64", "linux-arm", "linux-x64", "darwin-arm64", "windows-x64"],
        "url": "https://github.com/<owner>/apex-tools/releases/download/platforms-v34/platforms-android-34-linux-arm64.tar.xz",
        "sha256": "...", "size": 123456,
        "license": "apex-sdk-license"
      }
    ]
  }
  ```
- **manifest lokal** `$ANDROID_HOME/.apex/manifest.json`: `{path, version, sha256, installed_at}`.
- **licenses/**: teks lisensi + hash (plain files).
- **source.properties** per paket: `Pkg.Revision`, `Pkg.License=apex-sdk-license`, dst sesuai jenis komponen.

## 14. Error Handling

- 404/URL salah → "package not found in catalog; run `apex --list`", exit 2.
- Checksum mismatch → hapus tmp, exit 3.
- Lisensi belum accepted → exit 4 + instruksi.
- Katalog unreachable/invalid → exit 5.
- Extract gagal (disk penuh, permission) → bersihkan tmp, exit 6; instalasi lama tidak tersentuh (atomic rename).
- Versi CLI < katalog min_cli → warning + tetap lanjut (deprecated behavior di masa depan).

## 15. Security Considerations

- HTTPS-only; pin ke GitHub Releases domain di default config.
- sha256 wajib; katalog ditandatangani (opsional masa depan: minisign) — NOT PART OF CURRENT SCOPE.
- Jangan pernah mengekstrak path dengan `..` (zip-slip protection wajib).

## 16. Success Metrics

- AC-US-008 terpenuhi: build Gradle nyata sukses di Termux aarch64 dengan SDK yang diinstal via apex.
- Seluruh paket di §14 (≥ 10 platforms, 10 build-tools, 1 platform-tools, 17 cmake, 32 NDK) muncul di katalog untuk host linux-arm64 & linux-arm.
- Waktu `apex install platform-tools` < 1 menit di koneksi normal.

## 17. Risks and Trade-offs

| Risiko | Dampak | Mitigasi |
|---|---|---|
| build-tools lama (33.0.x) tidak ada binari aarch64/arm | Gradle project yang pin build-tools lama gagal di ARM host | OQ-1: (a) param-kan TOOLS_VERSION di android-sdk-custom dan build beberapa versi; atau (b) sediakan hanya versi terbaru + dokumentasi; keputusan sebelum implementasi make-build-tools.yml |
| Hash lisensi AGP berubah di versi AGP baru | Gradle menolak | Dual-file + monitor AGP release; hash bisa di-append oleh CLI update |
| NDK r30 di atas perlu llvm-custom r30 rilis lengkap | ndk;30.x gagal install/build | Katalog hanya memuat NDK yang upstream artifact-nya benar-benar ada; gen-catalog.sh memverifikasi URL sebelum publish |
| GitHub rate limit saat CLI fetch katalog | list lambat/gagal di CI | katalog di-serve dari raw branch (cacheable) + retry |
| arm32 (armv7) NDK host tools dari android-ndk-custom mungkin belum lengkap untuk semua rXX | ndk;X gagal untuk host arm | Matrix CI memverifikasi keberadaan; katalog menandai host yang tersedia per paket |

## 18. Assumptions & Open Questions

**Assumptions**
- REPO_OWNER = `HomuHomu833` (mengikuti repo saudara).
- Lisensi apex = teks yang ditentukan user (file LICENSE apex-tools), berbeda dari Google.
- CI di GitHub Actions ubuntu-24.04 (x86_64) dengan cross-compile; runner arm64 (ubuntu-24.04-arm) opsional untuk E2E test.
- Semua versi cmake yang diminta tersedia sebagai tag Kitware (3.18.1 … 4.1.2) — diverifikasi saat matrix dibuat.

**Open Questions**
- **OQ-1**: Strategi build-tools multi-versi 33.0.x untuk aarch64/arm: param-kan TOOLS_VERSION di android-sdk-custom (butuh PR di sana) vs hanya versi terbaru? *(Rekomendasi: param-kan, karena beberapa project Gradle pin build-tools lama; tapi biaya maintenance naik.)*
- **OQ-2**: Rilis NDK mana dari r26–r30 yang sudah memiliki artefak bionic/arm di android-ndk-custom (semua 32 versi di daftar user)? Verifikasi keberadaan aset saat menyusun matrix.
- **OQ-3**: Format penamaan tag rilis per komponen (mis. `platform-tools-v39.0`, `build-tools-v35.0.0`, `ndk-v27.2.12479018`) — final saat implementasi workflow; harus stabil karena URL masuk katalog.

## 19. Future Considerations (NOT PART OF CURRENT SCOPE)

- `avdmanager` equivalent + emulator/system-images.
- Katalog bertanda tangan (minisign/sigstore).
- Java/Kotlin version of CLI (drop-in replacement cmdline-tools untuk Studio).
- `apex channel` (stable/beta/preview) seperti channel Google.
- Mirror katalog via CDN/jsDelivr.

## 20. Implementation Notes

- Mulai dari CLI + katalog dengan paket yang sudah pasti ada (platform-tools, platforms dari Google, cmake dari cmake-custom) — NDK menyusul setelah verifikasi aset (OQ-2).
- `make-package.sh` = satu skrip env-driven seperti pola repo saudara: input URL arsip upstream + jenis komponen + versi + hosttag → output paket SDK-layout siap rilis.
- package.xml generator: port kecil dari `package-generator.c` (android-ndk-custom) ke Go atau tetap C (host cc) — rekomendasi: Go, agar satu toolchain.
- Zip-slip guard wajib di extractor.
- .gitattributes: force LF untuk scripts/*, *.sh, Dockerfile, *.patch (konsisten dengan repo saudara).

---

## Lampiran: Katalog paket (daftar user, kanonik)

**platforms**: android-30, 31, 32, 33, 34, 35, 36, 36.1, 37.0, 37.1, 37.2 (11)
**build-tools**: 33.0.0, 33.0.1, 33.0.2, 33.0.3, 34.0.0, 35.0.0, 35.0.1, 36.0.0, 36.1.0, 37.0.0 (10)
**platform-tools**: latest (1)
**cmake**: 3.18.1, 3.22.1, 3.30.3, 3.30.4, 3.30.5, 3.31.0, 3.31.1, 3.31.2, 3.31.3, 3.31.4, 3.31.5, 3.31.6, 4.0.2, 4.0.3, 4.1.0, 4.1.1, 4.1.2 (17)
**ndk**: 26.0.10792818, 26.2.11394342, 26.3.11579264, 27.0.11718014, 27.0.11902837, 27.0.12077973, 27.1.12297006, 27.2.12479018, 27.3.13750724, 28.0.12433566, 28.0.12674087, 28.0.12916984, 28.0.13004108, 28.1.13356709, 28.2.13676358, 29.0.13113456, 29.0.13599879, 29.0.13846066, 29.0.14033849, 29.0.14206865, 30.0.14904198, 30.0.15729638, 30.0.16138531, 30.0.16248370 (24; user menulis "..." — daftar final = seluruh rilis r26–r30 upstream yang punya artefak)
