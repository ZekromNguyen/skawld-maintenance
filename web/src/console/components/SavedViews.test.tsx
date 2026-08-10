import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { SavedViews } from "./SavedViews";
import { I18nProvider } from "../../i18n/I18nProvider";

const views = [{ id: "v1", name: "My critical queue", priority: "CRITICAL", query: "" }];

function renderViews(props: {
  onSave: () => void;
  onApply: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  return render(
    <I18nProvider>
      <SavedViews
        views={views}
        activeId={null}
        onSave={props.onSave}
        onApply={props.onApply}
        onDelete={props.onDelete}
      />
    </I18nProvider>,
  );
}

describe("SavedViews", () => {
  it("lists saved views and applies on click", () => {
    const onApply = vi.fn();
    renderViews({ onSave: vi.fn(), onApply, onDelete: vi.fn() });
    fireEvent.click(screen.getByText("My critical queue"));
    expect(onApply).toHaveBeenCalledWith("v1");
  });
  it("offers save current filter", () => {
    const onSave = vi.fn();
    renderViews({ onSave, onApply: vi.fn(), onDelete: vi.fn() });
    fireEvent.click(screen.getByRole("button", { name: /Save view/ }));
    expect(onSave).toHaveBeenCalled();
  });
});
