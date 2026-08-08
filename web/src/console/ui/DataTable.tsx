import { useMemo, useState, type KeyboardEvent, type ReactNode } from "react";
import { EmptyState } from "./EmptyState";
import { ErrorState } from "./ErrorState";
import { Skeleton } from "./Skeleton";

export interface Column<T> {
  key: string;
  header: string;
  render: (row: T) => ReactNode;
  sortValue?: (row: T) => string | number;
}

export function DataTable<T>({
  columns,
  rows,
  rowKey,
  emptyTitle,
  emptyBody,
  loading = false,
  error,
  onRetry,
  onRowClick,
  rowClassName,
  selectedKey,
  onTableKeyDown,
}: {
  columns: Array<Column<T>>;
  rows: T[];
  rowKey: (row: T) => string;
  emptyTitle: string;
  emptyBody?: string;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  onRowClick?: (row: T) => void;
  rowClassName?: (row: T) => string;
  selectedKey?: string | null;
  onTableKeyDown?: (event: KeyboardEvent<HTMLDivElement>) => void;
}) {
  const [sort, setSort] = useState<{ key: string; dir: 1 | -1 } | null>(null);

  const sorted = useMemo(() => {
    if (!sort) return rows;
    const column = columns.find((c) => c.key === sort.key);
    if (!column?.sortValue) return rows;
    return [...rows].sort((a, b) => {
      const av = column.sortValue!(a);
      const bv = column.sortValue!(b);
      if (av < bv) return -1 * sort.dir;
      if (av > bv) return 1 * sort.dir;
      return 0;
    });
  }, [rows, sort, columns]);

  if (error) {
    return <ErrorState message={error} onRetry={onRetry} />;
  }
  if (loading) {
    return <Skeleton height={180} />;
  }
  if (sorted.length === 0) {
    return <EmptyState title={emptyTitle} body={emptyBody} />;
  }
  const clickableRows = onRowClick != null;
  return (
    <div className="table-wrap" onKeyDown={onTableKeyDown} tabIndex={clickableRows ? 0 : undefined}>
      <table>
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                role="columnheader"
                aria-sort={
                  sort?.key === column.key
                    ? sort.dir === 1
                      ? "ascending"
                      : "descending"
                    : undefined
                }
              >
                {column.sortValue ? (
                  <button
                    className="sort-button"
                    onClick={() =>
                      setSort((prev) =>
                        prev?.key === column.key
                          ? { key: column.key, dir: prev.dir === 1 ? -1 : 1 }
                          : { key: column.key, dir: 1 },
                      )
                    }
                  >
                    {column.header}
                  </button>
                ) : (
                  column.header
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sorted.map((row) => {
            const key = rowKey(row);
            const selected = selectedKey != null && selectedKey === key;
            const clickable = onRowClick != null;
            return (
              <tr
                key={key}
                className={[
                  rowClassName?.(row),
                  selected ? "data-row selected" : undefined,
                ]
                  .filter(Boolean)
                  .join(" ")}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                aria-selected={selected || undefined}
              >
                {columns.map((column) => (
                  <td key={column.key}>{column.render(row)}</td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
