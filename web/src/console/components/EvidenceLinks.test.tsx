import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { I18nProvider } from "../../i18n/I18nProvider";
import { EvidenceLinks } from "./EvidenceLinks";
import type { Evidence } from "../../types";

const documentEvidence: Evidence = {
  id: "document_chunk:c1",
  kind: "DOCUMENT_CHUNK",
  source_id: "c1",
  document_id: "d9",
  title: "LOTO Procedure for P-302",
  locator: "sop/loto",
  authority: "SITE_APPROVED",
  content: "Apply lockout before inspection.",
  content_sha256: "x",
  score: { rrf_score: 0.94 },
};

const incidentEvidence: Evidence = {
  id: "historical_incident:i7",
  kind: "HISTORICAL_INCIDENT",
  source_id: "i7",
  title: "Pump vibration incident",
  locator: "inc/7",
  authority: "SITE_APPROVED",
  content: "Vibration readings above threshold.",
  content_sha256: "y",
  score: { rrf_score: 0.5 },
};

describe("EvidenceLinks", () => {
  it("links document evidence to the knowledge page", () => {
    render(
      <MemoryRouter>
        <I18nProvider>
          <EvidenceLinks
            evidence={[documentEvidence]}
            selected={["document_chunk:c1"]}
            emptyTitle="None"
          />
        </I18nProvider>
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: /loto procedure/i });
    expect(link.getAttribute("href")).toBe("/knowledge/d9");
  });

  it("links incident evidence to the incident page", () => {
    render(
      <MemoryRouter>
        <I18nProvider>
          <EvidenceLinks
            evidence={[incidentEvidence]}
            selected={["historical_incident:i7"]}
            emptyTitle="None"
          />
        </I18nProvider>
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: /pump vibration incident/i });
    expect(link.getAttribute("href")).toBe("/incidents/i7");
  });
});
