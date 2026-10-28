"use client";

export default function ErrorPage({ reset }: { error: Error; reset: () => void }) {
  return (
    <>
      <h1>Meldungen gerade nicht verfügbar</h1>
      <p className="sub">Bitte versuche es in einem Moment erneut.</p>
      <button type="button" onClick={reset}>
        Neu laden
      </button>
    </>
  );
}
