import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { MagnifyingGlass } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";
import type { Principal } from "../../types";

interface Command {
  id: string;
  label: string;
  run: () => void;
}

/**
 * CommandPalette: Ctrl/Cmd+K quick navigation and actions. Filterable,
 * arrow-key navigable, Enter to run, Esc to close.
 */
export function CommandPalette({ principal }: { principal?: Principal }) {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [index, setIndex] = useState(0);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setOpen((current) => !current);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const commands = useMemo<Command[]>(() => {
    const nav: Command[] = [
      { id: "overview", label: t("nav.overview"), run: () => navigate("/") },
      { id: "incidents", label: t("nav.incidents"), run: () => navigate("/incidents") },
      { id: "executions", label: t("nav.executions"), run: () => navigate("/executions") },
      { id: "handover", label: t("nav.handover"), run: () => navigate("/handovers") },
      { id: "knowledge", label: t("nav.knowledge"), run: () => navigate("/knowledge") },
      { id: "search", label: t("nav.search"), run: () => navigate("/search") },
      { id: "reports", label: t("nav.reports"), run: () => navigate("/reports") },
      { id: "demonstrations", label: t("nav.demonstrations"), run: () => navigate("/demonstrations") },
      { id: "workflows", label: t("nav.workflows"), run: () => navigate("/workflows") },
      { id: "quality", label: t("nav.quality"), run: () => navigate("/quality") }
    ];
    if (principal?.permissions.includes("incident:create")) {
      nav.push({
        id: "create-incident",
        label: t("form.createIncident"),
        run: () => navigate("/incidents?create=1")
      });
    }
    return nav;
  }, [navigate, t, principal]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return q ? commands.filter((command) => command.label.toLowerCase().includes(q)) : commands;
  }, [commands, query]);

  useEffect(() => {
    setIndex(0);
  }, [filtered.length, open]);

  const runCommand = (command: Command) => {
    setOpen(false);
    setQuery("");
    command.run();
  };

  const onKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setIndex((current) => Math.min(current + 1, filtered.length - 1));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setIndex((current) => Math.max(current - 1, 0));
    } else if (event.key === "Enter" && filtered[index]) {
      event.preventDefault();
      runCommand(filtered[index]);
    }
  };

  return (
    <DialogPrimitive.Root open={open} onOpenChange={setOpen}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="dialog-overlay" />
        <DialogPrimitive.Content
          className="palette"
          onKeyDown={onKeyDown}
        >
          <DialogPrimitive.Title className="visually-hidden">
            {t("cmd.paletteTitle")}
          </DialogPrimitive.Title>
          <div className="palette-input-row">
            <MagnifyingGlass size={16} weight="duotone" aria-hidden="true" />
            <input
              autoFocus
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t("cmd.placeholder")}
              aria-label={t("cmd.placeholder")}
              className="palette-input"
            />
          </div>
          <div className="palette-list" role="listbox" aria-label={t("cmd.paletteTitle")}>
            {filtered.length === 0 ? (
              <div className="palette-empty">{t("cmd.noResults")}</div>
            ) : (
              filtered.map((command, commandIndex) => (
                <button
                  key={command.id}
                  type="button"
                  role="option"
                  aria-selected={commandIndex === index}
                  className={`palette-item${commandIndex === index ? " active" : ""}`}
                  onMouseEnter={() => setIndex(commandIndex)}
                  onClick={() => runCommand(command)}
                >
                  {command.label}
                </button>
              ))
            )}
          </div>
          <small className="palette-hint">{t("cmd.hint")}</small>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
