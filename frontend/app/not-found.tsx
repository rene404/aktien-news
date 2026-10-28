import Link from "next/link";

export default function NotFound() {
  return (
    <>
      <h1>Nicht gefunden</h1>
      <p className="sub">Diese Seite oder Aktie gibt es (noch) nicht.</p>
      <Link href="/">Zur Startseite</Link>
    </>
  );
}
