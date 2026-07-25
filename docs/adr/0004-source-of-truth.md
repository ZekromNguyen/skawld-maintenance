# ADR 0004: Source of Truth

Status: Accepted  
Date: 2026-07-26

Every projection-capable record declares:

- `OWNED_BY_SKAWLD`, or
- `EXTERNAL_REFERENCE`.

External records retain source system, external identifier/version, mapping version, synchronization state, and last synchronization time. Native mutation and external write-back are separate authorized commands.

Skawld owns workflows, demonstrations, corrections, recommendations, and evidence traces. Enterprise asset masters, work orders, inventory, schedules, permits, and raw telemetry are external by default.

