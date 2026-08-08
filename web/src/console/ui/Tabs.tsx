import type { KeyboardEvent } from "react";

export type TabItem = { id: string; label: string; count?: number };

/**
 * Tabs: WAI-ARIA tabs with roving tabindex and arrow-key navigation.
 * Arrow keys move focus without activating; Enter/Space activates.
 */
export function Tabs({
  label,
  tabs,
  active,
  onChange,
}: {
  label: string;
  tabs: TabItem[];
  active: string;
  onChange: (id: string) => void;
}) {
  const panelId = `tabpanel-${label.replace(/\s+/g, "-")}`;

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, id: string) => {
    const index = tabs.findIndex((tab) => tab.id === id);
    if (index === -1) return;
    let next: number | undefined;
    switch (event.key) {
      case "ArrowRight":
        next = (index + 1) % tabs.length;
        break;
      case "ArrowLeft":
        next = (index - 1 + tabs.length) % tabs.length;
        break;
      case "Home":
        next = 0;
        break;
      case "End":
        next = tabs.length - 1;
        break;
      case "Enter":
      case " ":
        event.preventDefault();
        onChange(id);
        return;
      default:
        return;
    }
    event.preventDefault();
    const target = tabs[next!];
    onChange(target.id);
    const button = document.getElementById(`${panelId}-${target.id}`);
    button?.focus();
  };

  return (
    <div className="incident-tabs" role="tablist" aria-label={label}>
      {tabs.map((tab) => (
        <button
          key={tab.id}
          id={`${panelId}-${tab.id}`}
          role="tab"
          aria-selected={active === tab.id}
          aria-controls={panelId}
          tabIndex={active === tab.id ? 0 : -1}
          className={`tab${active === tab.id ? " active" : ""}`}
          onClick={() => onChange(tab.id)}
          onKeyDown={(event) => onKeyDown(event, tab.id)}
        >
          {tab.label}
          {tab.count !== undefined ? <span className="count">{tab.count}</span> : null}
        </button>
      ))}
    </div>
  );
}
