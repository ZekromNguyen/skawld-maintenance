import { useCallback, useEffect, useState } from "react";

export interface ListKeyboard {
  selectedIndex: number;
  setSelectedIndex: (index: number) => void;
  onKeyDown: (event: React.KeyboardEvent) => void;
}

/**
 * useListKeyboard: j/k row navigation for dense queues. j moves down, k
 * moves up, Enter triggers open, Escape clears the selection. Selection
 * clamps to the list bounds; open is only fired when a row is selected.
 */
export function useListKeyboard<T>({
  rows,
  onOpen,
}: {
  rows: T[];
  onOpen: (row: T, index: number) => void;
}): ListKeyboard {
  const [selectedIndex, setSelectedIndex] = useState(-1);

  useEffect(() => {
    setSelectedIndex(-1);
  }, [rows]);

  const onKeyDown = useCallback(
    (event: React.KeyboardEvent) => {
      if (rows.length === 0) return;
      const count = rows.length;
      if (event.key === "j" || event.key === "ArrowDown") {
        event.preventDefault();
        setSelectedIndex((current) => (current + 1) % count);
      } else if (event.key === "k" || event.key === "ArrowUp") {
        event.preventDefault();
        setSelectedIndex((current) => (current <= 0 ? count - 1 : current - 1));
      } else if (event.key === "Enter") {
        if (selectedIndex >= 0 && selectedIndex < count) {
          event.preventDefault();
          onOpen(rows[selectedIndex], selectedIndex);
        }
      } else if (event.key === "Escape") {
        setSelectedIndex(-1);
      }
    },
    [rows, selectedIndex, onOpen],
  );

  return { selectedIndex, setSelectedIndex, onKeyDown };
}
