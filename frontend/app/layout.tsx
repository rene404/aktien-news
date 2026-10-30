import type { Metadata } from "next";
import Link from "next/link";
import Disclaimer from "@/components/Disclaimer";
import "./theme.css";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "aktien-news.at – Börsennews zu ATX, DAX und US-Aktien", template: "%s – aktien-news.at" },
  description: "Aktuelle Meldungen und Ad-hoc-Mitteilungen zu österreichischen, deutschen und US-Aktien – mit Link zur Originalquelle.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="de">
      <body>
        <header className="site">
          <div className="wrap">
            <Link className="brand" href="/">
              aktien-news<span>.at</span>
            </Link>
            <form className="search" action="/suche" method="get" role="search">
              <input name="q" type="search" placeholder="Aktie, Ticker oder ISIN" aria-label="Aktie suchen" required maxLength={64} />
              <button type="submit">Suchen</button>
            </form>
          </div>
        </header>
        <main className="wrap">{children}</main>
        <footer className="site">
          <div className="wrap">
            <Disclaimer />
          </div>
        </footer>
      </body>
    </html>
  );
}
