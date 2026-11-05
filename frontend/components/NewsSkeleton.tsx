// Placeholder while the first page of news loads; same row layout as NewsList.
export default function NewsSkeleton({ rows = 8 }: { rows?: number }) {
  return (
    <div className="skeleton" role="status" aria-busy="true">
      <span className="sr-only">Meldungen werden geladen</span>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="skeleton-row" aria-hidden="true">
          <span />
          <span />
        </div>
      ))}
    </div>
  );
}
