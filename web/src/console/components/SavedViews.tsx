import { BookmarkSimple, FloppyDisk, Trash } from "@phosphor-icons/react";
import { useI18n } from "../../i18n/I18nProvider";

export interface SavedView {
  id: string;
  name: string;
  priority: string;
  query: string;
}

/**
 * SavedViews: persisted named filter combinations for the incident queue.
 * One click applies a view; the trash control removes it; the action chip
 * saves the current filter combination.
 */
export function SavedViews({
  views,
  activeId,
  onSave,
  onApply,
  onDelete,
}: {
  views: SavedView[];
  activeId: string | null;
  onSave: () => void;
  onApply: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const { t } = useI18n();
  return (
    <div className="saved-views" role="group" aria-label={t("incidents.views.label")}>
      <button type="button" className="chip chip-action" onClick={onSave}>
        <FloppyDisk size={12} aria-hidden="true" />
        {t("incidents.views.save")}
      </button>
      {views.map((view) => (
        <span key={view.id} className={`chip${view.id === activeId ? " active" : ""}`}>
          <button type="button" className="chip-apply" onClick={() => onApply(view.id)}>
            <BookmarkSimple size={12} aria-hidden="true" />
            {view.name}
          </button>
          <button
            type="button"
            className="chip-delete"
            aria-label={`${t("incidents.views.delete")}: ${view.name}`}
            onClick={() => onDelete(view.id)}
          >
            <Trash size={11} aria-hidden="true" />
          </button>
        </span>
      ))}
    </div>
  );
}
