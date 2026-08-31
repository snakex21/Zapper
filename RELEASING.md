# Releasing Zapper

The release version is defined by `appVersion` in `app/app.go`. Git tags must use the same version with a `v` prefix, for example `v10.3.6`.

## Local release check

Run:

```powershell
.\release.ps1 -Tag v10.3.6
```

or use `release.bat` for an interactive Windows build.

The release process:

1. runs Go tests and `go vet`;
2. generates all 30 firmware variants;
3. validates all 30 UI/guide/frequency language sets;
4. validates all 30 README files and their language links;
5. builds the developer executable and clean portable package;
6. verifies that release `data/` is empty;
7. creates a versioned Windows x64 ZIP and SHA-256 checksum in `dist/`.

For version 10.3.6 the expected files are:

```text
dist/Zapper-v10.3.6-Windows-x64.zip
dist/Zapper-v10.3.6-Windows-x64.zip.sha256
```

## GitHub Release

After the local release check passes:

```text
git tag v10.3.6
git push origin v10.3.6
```

The `.github/workflows/release.yml` workflow verifies that the tag matches `appVersion`, rebuilds the package on `windows-latest`, and publishes the ZIP plus SHA-256 checksum as a GitHub Release.

## Files that must never be committed

Local runtime data and build products are excluded through `.gitignore`, including:

- `data/` — profiles, progress, archive, settings, logs and backups;
- `.supercli/` and `.local/` — local development state;
- `build/` — generated and portable builds;
- `dist/` — release archives and checksums;
- generated `*.exe`, `*.zip`, `*.log` and checksum files.

The committed `locales/` directory is different: it contains the canonical public UI and guide translation packages required to build a release and must remain in the repository.
