# apex-tools

Custom Android SDK manager + repackaged SDK components for **aarch64 / arm (armv7)** hosts.
Google's cmdline-tools only ships x86_64 binaries; apex-tools ships a Go CLI that runs natively
on arm64/arm and installs SDK components built by this repo.

## Components (published as GitHub Releases)

| Package    | Versions                                   | Sources      |
|------------|--------------------------------------------|--------------|
| platforms  | android-30 .. android-37.2                 | AOSP         |
| build-tools| 33.0.0 .. 37.0.0                           | AOSP (android-sdk-custom method) |
| platform-tools | latest                                 | AOSP (android-sdk-custom method) |
| cmake      | 3.18.1 .. 4.1.2                            | cmake-custom method |
| ndk        | r26 .. r30                                 | android-ndk-custom method |

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
