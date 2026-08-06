export function Skeleton({
  height,
  width = "100%",
}: {
  height: number;
  width?: number | string;
}) {
  return (
    <div
      role="status"
      aria-busy="true"
      aria-label="Loading"
      className="skeleton"
      style={{ height, width }}
    />
  );
}
