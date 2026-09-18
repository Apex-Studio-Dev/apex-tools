# apex-tools

Custom Android SDK manager + repackaged SDK components for **aarch64 / arm (armv7)** hosts.
Google's cmdline-tools only ships x86_64 binaries; apex-tools ships a Go CLI that runs natively
on arm64/arm and installs SDK components built by this repo.

## Components (published as GitHub Releases)

| Package    | Versions                                   | Sources      |
|------------|--------------------------------------------|--------------|
| platforms  | android-30 31 32 33 34 35 36 36.1 37.0 37.1 37.2 | AOSP         |
| build-tools| 33.0.0 33.0.1 33.0.2 33.0.3 34.0.0 35.0.0 35.0.1 36.0.0 36.1.0 37.0.0 | AOSP (android-sdk-custom method) |
| platform-tools | 37.0.1                                   | AOSP (android-sdk-custom method) |
| cmake      | 3.18.1 3.22.1 3.30.3 3.30.4 3.30.5 3.31.0 3.31.1 3.31.4 3.31.5 3.31.6 4.0.2 4.0.3 4.1.0 4.1.1 4.1.2 | cmake-custom method |
| ndk        | 26.0.10792818 26.1.10909125 26.2.11394342 26.3.11579264 27.0.12077973 27.1.12297006 27.2.12479018 27.3.13750724 28.0.13004108 28.1.13356709 28.2.13676358 29.0.14206865 30.0.16248370 | android-ndk-custom method |

Host tags: `linux-arm64`, `linux-arm` (+ `linux-x64` convenience).

## CLI

```
apex --list                  list available/installed packages
apex install <pkg>...        install ("platforms;android-34", "build-tools;37.0.0", ...)
apex uninstall <pkg>...      remove
apex update                  upgrade side-by-side
apex licenses [--accept]     accept license (writes licenses/ for Gradle)
```

Requires `ANDROID_HOME` (or `ANDROID_SDK_ROOT`). Catalog URL override: `APEX_CATALOG_URL`.

Catalog is published at `github.com/Apex-Studio-Dev/apex-tools` on tag `catalog` (also served raw
from branch) as `catalog.json`.

## Building

```
scripts/build-cli.sh        # cross-build apex for linux-arm64/arm
scripts/gen-catalog.sh      # regenerate catalog.json from release assets
scripts/make-package.sh     # package a component archive from its build output
```

## CI

`.github/workflows/` — `cli.yml` (build/test CLI), reusable `make-<component>.yml` +
dispatchers, `catalog.yml` (regenerate catalog.json on release / manual).

## License

See `licenses/apex-sdk-license`.
