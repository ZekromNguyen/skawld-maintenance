import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { StatusBadge } from "./StatusBadge";
import { EmptyState } from "./EmptyState";
import { ErrorState } from "./ErrorState";
import { Skeleton } from "./Skeleton";
import { FormField } from "./FormField";
import { DataTable } from "./DataTable";

describe("console UI primitives", () => {
  it("renders a tone badge with the label", () => {
    render(<StatusBadge tone="high" label="HIGH" />);
    const badge = screen.getByText("HIGH");
    expect(badge.className).toContain("tone-high");
  });

  it("renders distinct classes for each severity tone", () => {
    const { container, rerender } = render(<StatusBadge tone="low" label="a" />);
    const classes = new Set<string>();
    for (const tone of ["low", "medium", "high", "critical"] as const) {
      rerender(<StatusBadge tone={tone} label={tone} />);
      classes.add(container.querySelector(".tone")?.className ?? "");
    }
    expect(classes.size).toBe(4);
  });

  it("renders an empty state with title and action", () => {
    render(<EmptyState title="Nothing here" action={<button>Add</button>} />);
    expect(screen.getByText("Nothing here")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Add" })).toBeTruthy();
  });

  it("renders an error state with retry", () => {
    let clicks = 0;
    render(<ErrorState message="boom" onRetry={() => { clicks += 1; }} />);
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(clicks).toBe(1);
  });

  it("renders a skeleton with aria-busy", () => {
    render(<Skeleton height={40} />);
    const status = screen.getByRole("status");
    expect(status.getAttribute("aria-busy")).toBe("true");
  });

  it("associates form labels with inputs and shows errors", () => {
    render(
      <FormField label="Tag" htmlFor="tag" error="Required">
        <input id="tag" />
      </FormField>,
    );
    expect(screen.getByLabelText("Tag")).toBeTruthy();
    expect(screen.getByText("Required")).toBeTruthy();
  });

  it("sorts DataTable rows on header click", () => {
    type Row = { id: string; n: number };
    const rows: Row[] = [
      { id: "a", n: 2 },
      { id: "b", n: 1 },
    ];
    render(
      <DataTable
        columns={[{ key: "n", header: "N", render: (r) => r.n, sortValue: (r) => r.n }]}
        rows={rows}
        rowKey={(r) => r.id}
        emptyTitle="Empty"
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "N" }));
    const cells = screen.getAllByRole("cell");
    expect(cells[0].textContent).toBe("1");
  });
});
