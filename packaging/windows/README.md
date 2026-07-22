# Windows installer (best-effort, unsupported)

`jennifer.iss` is an [Inno Setup](https://jrsoftware.org/isinfo.php) script that
produces `jennifer-<version>-setup.exe` - a per-user, no-admin installer for the
standard-Go `jennifer.exe`. Linux stays the only supported platform; this is the
same best-effort, **unsupported** build as the `-UNSUPPORTED.zip`, just wrapped
in an installer. Unsigned, so Windows SmartScreen warns ("More info -> Run
anyway").

## What it does

- Installs `jennifer.exe` (plus `README.md`, `JENNIFER.md`, `LICENSE.txt`,
  `UNSUPPORTED.txt`) to `%LOCALAPPDATA%\Programs\Jennifer`.
- Bundles the Jennifer-coded system modules (`modules/*.j`, minus `*_test.j`)
  under `share\jennifer\modules\` and sets `JENNIFER_SYSMODDIR` to that path, so
  a bare `import "name.j";` resolves. This is required on Windows: the
  compile-time module dir (`ResolveSysmoddir`'s default in
  `internal/module/sysmoddir.go`) is a POSIX path that does not exist here.
- Prepends the install dir to the user `PATH` (opt-out task).
- Optionally associates `.j` (opt-in task): a `Jennifer.Source` ProgId whose
  default double-click opens the source in Notepad (safe) with an explicit
  "Run with Jennifer" right-click verb.
- Registers an Apps & Features uninstaller that reverses PATH, the env var, and
  the association.

The icon is the repo's existing multi-size `docs/favicon.ico`.

## Building locally

Needs Inno Setup 6.3+ (`iscc` / `ISCC.exe`). Build `jennifer.exe` at the repo
root first (the script references `..\..\jennifer.exe`):

```
go build -trimpath -ldflags="-s -w" -o jennifer.exe ./cmd/jennifer
ISCC.exe /DAppVersion=0.20.0 packaging\windows\jennifer.iss
```

The installer lands in `dist\jennifer-0.20.0-setup.exe`. Without `/DAppVersion`
it defaults to `0.0.0-dev`. In CI this is built by the `build-windows-installer`
job in [`.github/workflows/release.yml`](../../.github/workflows/release.yml) on
a `windows-latest` runner and attached to the GitHub Release.
