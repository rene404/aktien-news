import { sourceIcon } from "@/lib/sources";

type Props = { sourceKey: string; name: string };

/** The publisher's icon before the source name; decoration only, the name follows as text. */
export default function SourceIcon({ sourceKey, name }: Props) {
  const src = sourceIcon(sourceKey);
  if (!src) {
    // The letter comes from CSS (::before), so it adds no text to the row.
    return <span className="src-icon" data-letter={name.charAt(0).toUpperCase()} aria-hidden="true" />;
  }
  // A plain <img>: a 64 px PNG from public/ needs no image optimisation route.
  // eslint-disable-next-line @next/next/no-img-element
  return <img className="src-icon" src={src} alt="" width={16} height={16} loading="lazy" decoding="async" />;
}
