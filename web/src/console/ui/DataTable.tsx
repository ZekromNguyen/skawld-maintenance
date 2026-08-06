import { useMemo, useState } from "react";
import { EmptyState } from "./EmptyState";
import type { ReactNode } from "react";
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
  emptyAction,
  loading = false,
  error,
  onRetry,
  onRowClick,
}: {
  columns: Array<Column<T>>;
  rows: T[];
  rowKey: (row: T) => string;
  emptyTitle: string;
  emptyBody?: string;
  emptyAction?: ReactNode;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  onRowClick?: (row: T) => void;
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
    return <EmptyState title={emptyTitle} body={emptyBody} action={emptyAction} />;
  }
  return (
    <div className="table-wrap">
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
          {sorted.map((row) => (
            <tr
              key={rowKey(row)}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
            >
              {columns.map((column) => (
                <td key={column.key}>{column.render(row)}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
