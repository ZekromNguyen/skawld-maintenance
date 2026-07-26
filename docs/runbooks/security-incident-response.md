# Security and Safety Incident Response

## Severity

- Critical: suspected cross-tenant disclosure, credential compromise, escaped
  critical/industrial-control proposal, audit tampering, destructive data loss.
- High: unauthorized privilege/approval use, malicious document execution,
  confirmed sensitive export, backup compromise, material unsafe guidance.
- Medium/Low: contained service abuse, availability degradation, failed job or
  provider issues without safety/data impact.

## Immediate actions

1. Preserve human and equipment safety; Skawld never substitutes for site
   emergency procedures.
2. Disable affected ingress/provider/connector capability. Do not disable
   industrial safeguards.
3. Revoke sessions, service credentials, signed URLs, and provider keys in the
   authoritative systems when compromise is plausible.
4. Preserve append-oriented audit, logs, database snapshot, job state, object
   metadata/checksums, build provenance, and involved model/prompt versions.
5. Notify named product, customer IT/OT security, maintenance authority, and
   safety contacts according to the pilot agreement.

## Investigation and recovery

- establish tenant/site/time/entity scope;
- compare audit export checksum and database/object provenance;
- identify exact recommendation/evidence/workflow/provider versions;
- quarantine malicious documents/media and stop related ingestion retries;
- rotate secrets using dual-key overlap where supported;
- restore only into an isolated target and verify before cutover;
- add a frozen regression case for every AI/safety failure;
- require human review before republishing affected workflows/documents.

No incident response action authorizes PLC/SCADA/DCS access or equipment
control. Record timeline, decisions, evidence, customer notifications, root
cause, corrective actions, and closure authority.
