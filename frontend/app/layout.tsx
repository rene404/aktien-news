import type { Metadata } from "next";
import Disclaimer from "@/components/Disclaimer";
import SiteHeader from "@/components/SiteHeader";
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
        <SiteHeader />
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
