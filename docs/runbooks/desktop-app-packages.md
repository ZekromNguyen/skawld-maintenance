# Windows and macOS Desktop Application Packages

The Phase 2 source tree defines Flutter technician/workstation packages as:

- a portable Windows x64 ZIP;
- a macOS DMG and ZIP for the native CI runner architecture;
- adjacent SHA-256 checksum files.

The package definitions are not a release claim: the current Fedora
verification environment does not have Flutter installed. Native CI must still
run analysis/tests and build each platform artifact before distribution.

The React application remains the supervisor web surface. The Go native
archives documented in `native-packages.md` are backend tools and are not the
desktop user application.

## Windows

Prerequisites:

- Flutter stable with Windows desktop enabled;
- Visual Studio 2022 with **Desktop development with C++**;
- PowerShell.

From the repository root:

```powershell
Set-Location mobile
flutter create --platforms=windows --org com.skawld `
  --project-name skawld_maintenance_mobile .
flutter pub get
dart run build_runner build
flutter analyze
flutter test
flutter build windows --release
..\scripts\package-desktop-windows.ps1 -Version v0.1.0
```

The ZIP contains the executable, required Flutter DLLs/data, and
`skawld-config.example.json`. Keep the extracted directory together.

## macOS

Prerequisites:

- Flutter stable with macOS desktop enabled;
- current Xcode command-line tools;
- CocoaPods when required by Flutter plugins.

From the repository root:

```bash
cd mobile
flutter create --platforms=macos --org com.skawld \
  --project-name skawld_maintenance_mobile .
../scripts/configure-flutter-macos.sh
flutter pub get
dart run build_runner build
flutter analyze
flutter test
flutter build macos --release
../scripts/package-desktop-macos.sh v0.1.0
```

The entitlement step enables outbound API/OIDC traffic, the localhost callback
listener used by Authorization Code + PKCE, and Keychain-backed credential
storage. Do not remove these entitlements without replacing the authentication
flow.

## Runtime configuration

Copy `skawld-config.example.json` to `skawld-config.json` beside the Windows
executable or beside the macOS `.app` bundle:

```json
{
  "api_url": "https://maintenance.example.internal",
  "oidc_issuer": "https://identity.example.internal/realms/skawld",
  "oidc_client_id": "skawld-mobile"
}
```

Environment variables `SKAWLD_API_URL`, `SKAWLD_OIDC_ISSUER`, and
`SKAWLD_OIDC_CLIENT_ID` override the file. `SKAWLD_CONFIG_FILE` selects an
explicit managed file. Endpoint configuration is not a secret.

## Release boundary

CI packages are engineering artifacts:

- Windows is portable ZIP only, not MSI/MSIX;
- Windows Authenticode signing is not configured;
- macOS Developer ID signing and notarization are not configured;
- no auto-update mechanism exists.

Before an external pilot, pin Flutter and runner versions, add protected signing
secrets, produce SBOM/provenance, test clean-machine installation, and document
upgrade/rollback. Do not instruct users to disable Gatekeeper, SmartScreen, or
enterprise endpoint controls.
