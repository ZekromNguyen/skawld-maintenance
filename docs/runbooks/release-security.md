# Release, Signing, and Supply-Chain Gates

CI must format, vet, run race/integration tests, validate OpenAPI, run web and
Flutter tests, verify SDK pin/checksums, execute the frozen safety evaluation,
generate a CycloneDX SBOM, build non-root API/worker/migrator/web images, and
scan all four images.

For release tags, CI also creates artifact provenance. Customer release remains
blocked until:

- GitHub Actions and base images are pinned to reviewed immutable references;
- critical/high findings are fixed or have a named, dated, customer-approved
  exception;
- Windows binaries are Authenticode-signed;
- macOS binaries are Developer ID signed and notarized;
- signatures, SHA-256, SBOM, and provenance verification are tested on clean
  machines;
- database/object backup and rollback rehearsal passes;
- upgrade compatibility and support matrix are published;
- secret rotation and incident-response drills are complete.

Do not ask users to disable SmartScreen, Gatekeeper, endpoint protection, TLS
validation, or corporate security controls.

OIDC and object-storage compatibility are deployment contracts, not branding
claims. Validate issuer/audience/PKCE/logout/group mapping and the exact
Put/Get/Delete/Head/signed-URL behavior before qualification.
