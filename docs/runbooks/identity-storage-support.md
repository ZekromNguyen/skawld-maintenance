# OIDC and S3 Pilot Support Matrix

Support is qualified per customer configuration. “Protocol compatible” is not
the same as “tested and supported.”

## OIDC

Required behavior:

- Authorization Code flow for web, with server-side opaque session cookie;
- Authorization Code + PKCE for Flutter/native;
- stable `sub`, issuer validation, audience validation, clock synchronization;
- explicit organization/site membership mapping inside Skawld;
- short-lived access tokens and documented session revocation;
- separate Skawld `ApprovalAuthority`; IdP role/group membership alone never
  grants workflow, report, safety, or criticality approval.

| Provider | Development | Pilot |
|---|---:|---:|
| Included Keycloak realm | Yes | Only after customer hardening/operations review |
| Microsoft Entra ID | Not bundled | Contract-test required |
| Okta/Auth0/Authentik/other OIDC | Not bundled | Contract-test required |
| No identity provider | No | Unsupported |

## S3 API

Skawld's V1 object port uses Put, Get, Delete, Head, constrained signed upload,
and signed download. Multipart, object lock, retention, replication, and
versioning are not assumed.

| Target | Development | Pilot |
|---|---:|---:|
| Adobe S3Mock | Contract tests only | Unsupported |
| AWS S3 | Not bundled | Contract-test, security, region, backup review |
| Ceph RGW | Not bundled | Exact-operation compatibility and support review |
| Customer-selected appliance/private service | Not bundled | License, lifecycle, backup/restore, TLS, IAM, and contract-test review |
| Local filesystem | No | Unsupported |

Required pilot checks include checksum/size/MIME behavior, signed URL expiry,
service identity least privilege, TLS/certificate trust, bucket isolation,
backup/restore, unavailable-service behavior, and object/database consistency.

## Secret rotation

Inventory OIDC client secret, database owner/runtime/backup identities, S3
service identity, private AI/speech credentials, TLS keys, signing identities,
and CI credentials. Prefer overlapping key rotation:

1. create new credential with equal or narrower scope;
2. deploy consumers and verify;
3. revoke old credential;
4. verify denial and audit the change.

Database password rotation may require a controlled connection drain. Signing
keys must use the platform/vendor protected signing service; never place them in
repository variables or release archives.
