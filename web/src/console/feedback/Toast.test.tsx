import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ToastProvider, useToast } from "./Toast";

function Probe() {
  const toast = useToast();
  return (
    <button onClick={() => toast.success("Saved")}>fire</button>
  );
}

describe("ToastProvider", () => {
  it("shows a success toast", () => {
    render(
      <ToastProvider>
        <Probe />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "fire" }));
    expect(screen.getByText("Saved")).toBeTruthy();
  });

  it("surfaces an error toast with retry", () => {
    const retry = vi.fn();
    function ProbeRetry() {
      const toast = useToast();
      return <button onClick={() => toast.error("Failed", { retry })}>go</button>;
    }
    render(
      <ToastProvider>
        <ProbeRetry />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "go" }));
    expect(screen.getByText("Failed")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(retry).toHaveBeenCalled();
  });
});
