import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { ToastProvider } from "./console/feedback/Toast";
import { PrincipalProvider } from "./console/state/PrincipalProvider";
import { ConsoleLayout } from "./console/layout/ConsoleLayout";
import { DashboardPage } from "./console/pages/DashboardPage";
import { QualityPage } from "./console/pages/QualityPage";
import { KnowledgePage } from "./console/pages/KnowledgePage";
import { HandoverPage } from "./console/pages/HandoverPage";
import { DemonstrationsPage } from "./console/pages/DemonstrationsPage";
import { WorkflowsPage } from "./console/pages/WorkflowsPage";
import { AssetsPage } from "./console/pages/AssetsPage";
import { AssetDetailPage } from "./console/pages/AssetDetailPage";
import { IncidentsPage } from "./console/pages/IncidentsPage";
import { IncidentDetailPage } from "./console/pages/IncidentDetailPage";
import { ExecutionWorkbenchPage } from "./console/pages/ExecutionWorkbenchPage";
import { ReportsPage } from "./console/pages/ReportsPage";
import { ReportDetailPage } from "./console/pages/ReportDetailPage";
import { DocumentDetailPage } from "./console/pages/DocumentDetailPage";
import { SearchPage } from "./console/pages/SearchPage";
import { ExecutionsPage } from "./console/pages/ExecutionsPage";

export function App() {
  return (
    <PrincipalProvider>
      <ToastProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<ConsoleLayout />}>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/assets" element={<AssetsPage />} />
              <Route path="/assets/:assetId" element={<AssetDetailPage />} />
              <Route path="/incidents" element={<IncidentsPage />} />
              <Route path="/incidents/:incidentId" element={<IncidentDetailPage />} />
              <Route path="/executions" element={<ExecutionsPage />} />
              <Route path="/executions/:executionId" element={<ExecutionWorkbenchPage />} />
              <Route path="/reports" element={<ReportsPage />} />
              <Route path="/reports/:reportId" element={<ReportDetailPage />} />
              <Route path="/handovers" element={<HandoverPage />} />
              <Route path="/knowledge" element={<KnowledgePage />} />
              <Route path="/knowledge/:documentId" element={<DocumentDetailPage />} />
              <Route path="/search" element={<SearchPage />} />
              <Route path="/quality" element={<QualityPage />} />
              <Route path="/demonstrations" element={<DemonstrationsPage />} />
              <Route path="/workflows" element={<WorkflowsPage />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ToastProvider>
    </PrincipalProvider>
  );
}
