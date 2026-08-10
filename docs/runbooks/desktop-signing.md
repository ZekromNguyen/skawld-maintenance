# Desktop Signing and Clean-Machine Install

The `.github/workflows/release.yml` workflow rebuilds and packages the
desktop app on Windows and macOS, then signs it. Every signing step is
guarded by a `secrets.XXX != ''` condition, so the workflow is green before
credentials exist and performs signing once the secrets are configured. The
packages are unsigned until then.

## Windows Authenticode

Provision an Authenticode code-signing certificate (an EV certificate is
recommended for the strongest SmartScreen reputation). Export it as a PFX
with a strong password and store the certificate base64 and the password as
repository or environment secrets:

- `WINDOWS_CERT_BASE64` — the PFX as base64 (`base64 -w0 cert.pfx` on GNU
  Linux, `base64 -b 0 cert.pfx` or `openssl base64 -in cert.pfx` on macOS)
- `WINDOWS_CERT_PASSWORD`

The workflow writes the PFX to the runner temp directory, signs the
built executable at `build/windows/x64/runner/Release/` before packaging,
and then packages the zip. Verify with:

```powershell
Get-AuthenticodeSignature .\skawld_maintenance_mobile.exe
```

## macOS Developer ID and notarization

Provision a Developer ID Application certificate and an App Store Connect
API key (or app-specific notary key) for `notarytool`:

- `MACOS_SIGNING_IDENTITY` — the Developer ID identity name used with
  `codesign --sign`
- `MACOS_NOTARY_KEY_ID` — the API key id (e.g. `ABCD123456`)
- `MACOS_NOTARY_ISSUER_ID` — the issuer id (e.g. `69a6de90-...`)
- `MACOS_NOTARY_PRIVATE_KEY` — the `.p8` private key contents; the workflow
  writes them to a temp file and passes the path to `notarytool --key`

The workflow codesigns the built app with hardened runtime
(`--options runtime`), packages the DMG, and submits it to `notarytool`
with `--wait` so the job fails if notarization is rejected, then staples it.

## Clean-machine install smoke test

Before calling a release "signed", install the signed package on a fresh VM
(or a clean local user) and verify:

1. Windows: no SmartScreen "Windows protected your PC" warning (or, if it
   appears, the publisher name is the certificate subject); launch the app
   and sign in with a demo account.
2. macOS: no Gatekeeper "cannot be opened because the developer cannot be
   verified" warning; the app launches and the menu bar shows the app name.
3. Load the demo data (`make seed` against the same database the app points
   at), open each console page, and confirm the signed binary behaves
   identically to the development build.

Record the OS version, package hash, and the result in the release notes.
