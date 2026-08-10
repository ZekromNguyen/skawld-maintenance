# Pilot Readiness — Decision and Drill Checklist

This runbook is **blocked on a named pilot customer**. None of the items can
be completed (or the drills measured) until the customer, deployment
topology, data classification, and support model are agreed. The checklist
records the decisions and the drills to execute once the targets exist; each
row names the evidence required for completion.

## Decisions (require the named customer)

| # | Item | Decision required | Evidence of completion |
|---|---|---|---|
| D1 | Deployment topology | Single-server/private deployment shape (host, sizing, network) built from `deployments/pilot/compose.yaml` | Written topology with host sizing and the compose profile |
| D2 | Data classification | Classification of maintenance, evidence, measurement, and personal data | Signed data-classification record |
| D3 | Retention targets | Retention window for each data class and the deletion procedure | Retention table with owner sign-off |
| D4 | Identity provider | IdP selection and the OIDC/SSO deployment guide steps executed (see `docs/runbooks/identity-storage-support.md`) | Configured IdP + successful demo-account login |
| D5 | Object storage | S3/on-prem object-storage selection for attachments | Configured bucket + a stored attachment |
| D6 | Support contacts | Support contacts and escalation path for the pilot | Contact list with response-time agreement |
| D7 | RPO/RTO targets | Recovery point and recovery time objectives agreed with the customer | Written RPO/RTO targets |
| D8 | Safety review | Safety review of the guided-workflow and recommendation behavior | Signed safety review |
| D9 | Customer acceptance | Customer sign-off of scope and acceptance criteria | Signed acceptance record |

## Drills (execute once D1–D9 are agreed)

### Restore drill (measured against the agreed RPO/RTO)

1. Take a fresh backup: `scripts/backup-postgres.sh` (see
   `docs/runbooks/backup-restore.md`).
2. Create a new empty/isolated target database and restore the backup into
   it with `scripts/restore-postgres.sh` (never rehearse into the live pilot
   database; the restore script refuses a non-empty target).
3. Verify schema and row counts: `scripts/verify-restore.sh`.
4. Record: elapsed time from failure to verified restore, and compare with
   the agreed RPO/RTO. The drill passes only when the measured time meets
   the targets; record the log as the completion evidence.

### Object-storage backup/restore drill

1. Confirm the selected object storage implements versioning or an explicit
   backup policy for the attachment bucket.
2. Delete an attachment object, restore it from the backup/version, and
   confirm the attachment manifest still resolves.
3. Record the procedure used and the result.

## Status

| | Item | Evidence |
|---|---|---|
| [ ] | D1 topology agreed | topology write-up with sizing |
| [ ] | D2 data classification signed | signed classification record |
| [ ] | D3 retention targets set | retention table with owner sign-off |
| [ ] | D4 IdP deployed | IdP config + demo-account login |
| [ ] | D5 object storage selected | bucket + stored attachment |
| [ ] | D6 support contacts named | contact list + response-time agreement |
| [ ] | D7 RPO/RTO agreed | written targets |
| [ ] | D8 safety review signed | signed review |
| [ ] | D9 customer acceptance | signed acceptance record |
| [ ] | Restore drill measured within RPO/RTO | drill log with elapsed time |
| [ ] | Object-storage drill executed | drill log + restored object |

The word "pilot-ready" may not be used until every box is checked and every
drill is measured against the agreed targets.
