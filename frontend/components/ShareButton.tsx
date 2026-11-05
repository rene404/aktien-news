"use client";

import { useEffect, useRef, useState, useSyncExternalStore } from "react";

const noop = () => () => {};
// Both need a secure context (https or localhost); on plain http neither exists.
const canShare = () => typeof navigator.share === "function" || typeof navigator.clipboard?.writeText === "function";
/** How long "Link kopiert" stays. */
const NOTICE_MS = 2000;

// Shares the original link of a news item: the phone's share sheet where
// there is one, else the link goes to the clipboard. Hidden without
// JavaScript and where the browser offers neither (plain http).
export default function ShareButton({ title, url }: { title: string; url: string }) {
  const usable = useSyncExternalStore(noop, canShare, () => false);
  const [notice, setNotice] = useState("");
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => () => clearTimeout(timer.current), []);

  const say = (text: string) => {
    setNotice(text);
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setNotice(""), NOTICE_MS);
  };

  const share = async () => {
    try {
      if (navigator.share) {
        await navigator.share({ title, url });
        return;
      }
      await navigator.clipboard.writeText(url);
      say("Link kopiert");
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return; // cancelled by the reader
      say("Teilen nicht möglich");
    }
  };

  return (
    <>
      <button type="button" className="share" hidden={!usable} aria-label={`Teilen: ${title}`} onClick={share}>
        Teilen
      </button>
      <span className="share-notice" role="status">
        {notice}
      </span>
    </>
  );
}
