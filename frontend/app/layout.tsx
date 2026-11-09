import type { Metadata } from "next";
import localFont from "next/font/local";
import Disclaimer from "@/components/Disclaimer";
import Shortcuts from "@/components/Shortcuts";
import SiteHeader from "@/components/SiteHeader";
import "./theme.css";
import "./globals.css";

// Alltop's faces, self-hosted (OFL, licences in app/fonts): variable weights, Latin subset.
const inter = localFont({
  src: "./fonts/inter-latin-wght.woff2",
  weight: "100 900",
  variable: "--font-inter",
  display: "swap",
});
const mono = localFont({
  src: "./fonts/jetbrains-mono-latin-wght.woff2",
  weight: "100 800",
  variable: "--font-jetbrains-mono",
  display: "swap",
});

export const metadata: Metadata = {
  title: { default: "aktien-news.at – Börsennews zu ATX, DAX, europäischen und US-Aktien", template: "%s – aktien-news.at" },
  description: "Aktuelle Meldungen und Ad-hoc-Mitteilungen zu österreichischen, deutschen und US-Aktien – mit Link zur Originalquelle.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="de" className={`${inter.variable} ${mono.variable}`}>
      <body>
        <SiteHeader />
        <main className="wrap">{children}</main>
        <footer className="site">
          <div className="wrap">
            <Disclaimer />
            <Shortcuts />
          </div>
        </footer>
      </body>
    </html>
  );
}
