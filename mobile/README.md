# Skawld Maintenance Mobile

Offline-first Flutter technician client. Phase 1 packaging prioritizes Windows
and macOS field/workstation deployments on approved devices. Android remains a
future field target; the React web remains the supervisor surface.

Generate Drift code before the first build:

```bash
dart run build_runner build
flutter test
flutter build apk
```

Generate and package a local desktop host:

```bash
# Run on macOS
flutter create --platforms=macos --org com.skawld --project-name skawld_maintenance_mobile .
../scripts/configure-flutter-macos.sh
flutter build macos --release
../scripts/package-desktop-macos.sh v0.1.0
```

```powershell
# Run on Windows
flutter create --platforms=windows --org com.skawld --project-name skawld_maintenance_mobile .
flutter build windows --release
../scripts/package-desktop-windows.ps1 -Version v0.1.0
```

Artifacts are written to `dist/desktop/` with SHA-256 files. Push builds create
an unsigned portable Windows ZIP plus macOS DMG/ZIP on native GitHub Actions
runners. Production distribution still requires Authenticode signing on
Windows and Developer ID signing/notarization on macOS. MSI/MSIX, Apple PKG,
auto-update, and certificate secret handling remain release engineering gates,
not Phase 1 runtime features.

Each package includes `skawld-config.example.json`. Copy it to
`skawld-config.json` beside the Windows executable or beside the macOS `.app`
bundle, then set the deployment API and OIDC endpoints. Environment variables
with the same names take precedence, and `SKAWLD_CONFIG_FILE` can point to a
managed configuration file:

```json
{
  "api_url": "https://maintenance.example.internal",
  "oidc_issuer": "https://identity.example.internal/realms/skawld",
  "oidc_client_id": "skawld-mobile"
}
```

Runtime endpoints are supplied at build time:

```bash
flutter build macos --release \
  --dart-define=SKAWLD_API_URL=https://maintenance.example.com \
  --dart-define=SKAWLD_OIDC_ISSUER=https://identity.example.com/realms/skawld \
  --dart-define=SKAWLD_OIDC_CLIENT_ID=skawld-mobile
```

The client never treats local permit or isolation input as authoritative.
Prerequisite verification is a privileged online command and must reference the
external PTW/LOTO authority.
