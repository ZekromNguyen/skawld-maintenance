# ADR 0003: Identity, Permission, and Approval

Status: Accepted  
Date: 2026-07-26

OIDC proves identity. Application RBAC grants ordinary actions. `ApprovalAuthority` independently grants the right to approve a subject/action within organization, site, scope, competency, risk ceiling, delegation, and time bounds.

A role such as Senior Technician never implies global approval authority. An approval requires both the ordinary approval permission and a current matching authority grant.

The local development identity provider is Keycloak. It is a development dependency, not a product-domain dependency and not the only supported OIDC provider.

