import type { ReactNode } from "react";
import type { Principal } from "../../types";
import { Breadcrumbs } from "./Breadcrumbs";
import { usePageTrail } from "./PageTrail";

export function PageHeader({
  title,
  actions,
  principal,
}: {
  title: string;
  actions?: ReactNode;
  principal?: Principal;
}) {
  const trail = usePageTrail();

  return (
    <header className="page-header">
      {trail.length > 0 ? <Breadcrumbs trail={trail} /> : null}
      <div className="page-header-row">
        <h1>{title}</h1>
        <div className="page-header-right">{actions}</div>
      </div>
    </header>
  );
}
