import { FormEvent, useCallback, useEffect, useMemo, useState } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { api } from "./api";
import { relativeTime, severityTone } from "./presentation";
import { useI18n } from "./i18n/I18nProvider";
import type { Locale, MessageKey } from "./i18n/messages";
import { ConsoleLayout } from "./console/layout/ConsoleLayout";
import { PlaceholderPage } from "./console/pages/PlaceholderPage";
import { DashboardPage } from "./console/pages/DashboardPage";
import { QualityPage } from "./console/pages/QualityPage";
import { KnowledgePage } from "./console/pages/KnowledgePage";
import { HandoverPage } from "./console/pages/HandoverPage";
import { DemonstrationsPage } from "./console/pages/DemonstrationsPage";
import { WorkflowsPage } from "./console/pages/WorkflowsPage";
import { EmptyRow } from "./console/components/EmptyRow";
import { EvidenceLinks } from "./console/components/EvidenceLinks";
import { AssetsPage } from "./console/pages/AssetsPage";
import { AssetDetailPage } from "./console/pages/AssetDetailPage";
import { IncidentsPage } from "./console/pages/IncidentsPage";
import { IncidentDetailPage } from "./console/pages/IncidentDetailPage";
import { ExecutionWorkbenchPage } from "./console/pages/ExecutionWorkbenchPage";
import type {
  Asset,
  Demonstration,
  Evidence,
  EvaluationSummary,
  Execution,
  Incident,
  KnowledgeDocument,
  MaintenanceReport,
  Principal,
  Recommendation,
  ShiftHandover,
  Step,
  WorkflowVersion
} from "./types";

type View =
  | "overview"
  | "assets"
  | "incidents"
  | "knowledge"
  | "handover"
  | "demonstrations"
  | "workflows"
  | "quality";

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<ConsoleLayout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/assets" element={<AssetsPage />} />
          <Route path="/assets/:assetId" element={<AssetDetailPage />} />
          <Route path="/incidents" element={<IncidentsPage />} />
          <Route path="/incidents/:incidentId" element={<IncidentDetailPage />} />
          <Route path="/executions/:executionId" element={<ExecutionWorkbenchPage />} />
          <Route path="/reports" element={<PlaceholderPage titleKey="nav.reports" />} />
          <Route path="/reports/:reportId" element={<PlaceholderPage titleKey="nav.reports" />} />
          <Route path="/handovers" element={<HandoverPage />} />
          <Route path="/knowledge" element={<KnowledgePage />} />
          <Route path="/knowledge/:documentId" element={<PlaceholderPage titleKey="nav.knowledge" />} />
          <Route path="/search" element={<PlaceholderPage titleKey="nav.search" />} />
          <Route path="/quality" element={<QualityPage />} />
          <Route path="/demonstrations" element={<DemonstrationsPage />} />
          <Route path="/workflows" element={<WorkflowsPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

function signOut() {
  // Submit a POST form so the browser follows the full 303 → Keycloak → SPA
  // redirect chain natively. Using fetch here would force it through a
  // cross-origin redirect that Keycloak does not CORS-allow, leaving the page
  // frozen on the SPA for several seconds before the catch block could fall
  // back to /auth/login.
  const form = document.createElement("form");
  form.method = "POST";
  form.action = "/auth/logout";
  form.style.display = "none";
  document.body.appendChild(form);
  form.submit();
}

const viewTitle: Record<View, MessageKey> = {
  overview: "pageTitle.overview",
  assets: "pageTitle.assets",
  incidents: "pageTitle.incidents",
  knowledge: "pageTitle.knowledge",
  handover: "pageTitle.handover",
  demonstrations: "pageTitle.demonstrations",
  workflows: "pageTitle.workflows",
  quality: "pageTitle.quality"
};

