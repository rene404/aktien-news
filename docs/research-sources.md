# Recherche: Datenquellen (Stand 2026-10-28)

**[V]** = selbst abgerufen und geprüft, **[U]** = aus Suchtreffern oder geschlossen, nicht geprüft.
Das ist eine technische Recherche, keine Rechtsberatung: Vor dem Livegang prüft ein österreichischer Jurist.
Hinweis: Nur im AustriaCard-Firmennetz ist `sec.gov` gesperrt, aus anderen Netzen funktioniert es (geprüft am 2026-10-29). GlobeNewswire liefert 403 (Bot-Schutz).

## Kernbefunde

1. **Wiener Börse ist nicht nutzbar.** [V] Der Disclaimer sagt: „Eine automatisierte Abfrage ist nicht gestattet.“
   Einen News- oder Ad-hoc-Feed gibt es nicht; die News-Seite zeigt APA-lizenzierte Inhalte.
2. **APA-OTS hat einen TDM-Vorbehalt.** [V] `ots.at/tdmrep.json` liefert `tdm-reservation: 1` für die ganze Website.
   RSS (Titel, Link, `author` = Aussender) ist frei abrufbar, der Meldungstext darf aber **nicht ohne Lizenz ins LLM**.
3. **EQS hat keinen öffentlichen Feed** [V], und seine robots.txt sperrt KI-Bots. Die EQS-Meldungen kommen aber
   über OTS `/rss/finanzen` (AT) und FinanzNachrichten `/rss-aktien-adhoc/` (DE) herein.
4. **FinanzNachrichten.de erlaubt die Übernahme von Überschrift und Link ausdrücklich.** [V] Die Erlaubnis ist kostenlos,
   aber widerruflich; Polling höchstens alle 10 Minuten, Links in neuem Fenster, keine Frames. Die Feeds haben ein `<fn:isin>`-Feld,
   und es gibt Feeds pro Index (DAX, MDAX, SDAX, Österreich 20, USA). `description` enthält dpa-AFX-Text, der **nicht verwendet** werden darf.
5. **SEC EDGAR 8-K** [U]: gemeinfrei und gratis, deckt alle US-Emittenten ab. Grenze 10 Requests/s,
   ein User-Agent mit Kontakt ist Pflicht. Für die USA ist das die rechtlich sicherste und breiteste Quelle.
6. **Die gratis Pläne der kommerziellen US-APIs sind für eine öffentliche Seite unzulässig:** Finnhub ist im Free-Plan nur
   nicht-kommerziell [U], NewsAPI im Developer-Plan nur für Entwicklung mit 24 h Verzögerung [V]. Bei Alpha Vantage braucht
   kommerzielle Nutzung eine Bestätigung des Anbieters [U].
7. **OeKB OAM** ist die maßgebliche Quelle für vorgeschriebene AT-Meldungen (ISIN, LEI). Sie ist öffentlich, hat aber keine
   dokumentierte API [U]. Wir fragen bei issuerinfo@oekb.at an.
8. **IR-Feeds der Emittenten:** Nur wenige haben RSS (voestalpine [V]). Erste, Verbund und Andritz haben keinen [V].

## Quellen-Matrix MVP

| Quelle | Markt | Zugang | Matching-Signal | Text fürs LLM? | Status |
|---|---|---|---|---|---|
| FinanzNachrichten `rss-aktien-adhoc`, Index-Feeds | DE, AT, US | RSS, ≥ 10 min | `fn:isin` | **nein** (nur Titel) | ✅ MVP |
| APA-OTS `/rss/finanzen`, `/rss/wirtschaft` | AT | RSS | `author` = Emittent | **nein** (TDM-Vorbehalt) | ✅ MVP |
| SEC EDGAR 8-K | US | JSON/Atom, 10 req/s, UA-Pflicht | CIK → Ticker | ja (gemeinfrei) | ✅ MVP |
| Business Wire / PR Newswire RSS | US (DE) | RSS | Ticker im Text „(NYSE: X)“ | nur Titel, bis die Bedingungen geklärt sind | ✅ MVP (Ergänzung) |
| IR-RSS der Emittenten (voestalpine …) | AT | RSS | Feed = Emittent | ja, sofern kein TDM-Vorbehalt | ✅ MVP, wo vorhanden |
| Alpha Vantage NEWS_SENTIMENT | US | API, free 25 req/Tag | Ticker | – (nur Baseline intern) | ⚠️ kommerzielle Nutzung anfragen |
| OeKB OAM | AT | keine API | ISIN, LEI | – | ⏳ Anfrage |
| APA-Comm Lizenz/API | AT | Vertrag | – | – | ⏳ Anfrage |
| EQS direkt | DE, AT | Vertrag | – | – | ⏳ Anfrage, falls nötig |
| Marketaux / Massive / Finnhub paid | US | API, 29–199 $/Monat | `tickers[]` | – | 🔜 später, falls die Breite nicht reicht |
| Wiener Börse | AT | – | – | – | ❌ ToS-Verbot |
| NewsAPI, Finnhub free | US | – | – | – | ❌ nicht kommerziell |

## Nachtrag 2026-10-29: weitere RSS/XML-Feeds

Alle Feeds am 2026-10-29 abgerufen [V]. „aktuell“ = neuester Eintrag vom selben Tag.
Keine der geprüften Domains (FN, Handelsblatt, derStandard, CNBC, onvista, Spiegel, FT, Nasdaq u. a.)
hat eine `.well-known/tdmrep.json`; damit ist der offene TDM-Punkt bei FinanzNachrichten geprüft.

| Quelle | Status | Einschätzung |
|---|---|---|
| FN `rss-aktien-nachrichten`, `rss-aktien-analysen` | ✅ aktuell, mit `fn:isin` | **aufgenommen** (`fn_aktien`, `fn_analysen`): gleiche Erlaubnis wie die übrigen FN-Feeds |
| FN MDAX (`-3`), SDAX (`-4`), TecDAX (`-2`), ~45 Branchen-Feeds | ✅ aktuell | erst sinnvoll, wenn MDAX/SDAX/TecDAX-Stammdaten im Seed sind (sonst nichts verlinkbar) |
| SEC EDGAR 6-K (ausländische Emittenten, auch europäische ADRs) | ✅ mit SEC_USER_AGENT | **aufgenommen** (`sec_6k`): gemeinfrei, gleicher Parser wie 8-K |
| Business Wire RSS | ✅ ein Branchen-Feed erreichbar | die Feed-Übersicht ist aus dem Firmennetz nicht erreichbar; Items haben weder `author` noch Ticker außerhalb der `description` → braucht eigenen Matching-Ansatz |
| **Nasdaq** `nasdaq.com/feed/rssoutbound?symbol=…` / `?category=…` | ✅ technisch, mit `<nasdaq:tickers>` je Item | ❌ **Terms of Use (Stand 2026-05-11):** Lizenz nur „personal, non-commercial use“; ohne schriftliche Vereinbarung nur Text-Link auf die Startseite `https://www.nasdaq.com` (kein Deep Link); kein automatisiertes Erfassen von Inhalten; kein KI-/Datenanalyse-Einsatz ohne Erlaubnis. Nur mit Lizenz nutzbar |
| FMA (`fma.gv.at/feed/`), EZB-, Fed-Pressemitteilungen | ✅ | Makro/Aufsicht ohne Ticker; braucht im UI einen Bereich für nicht verlinkte Meldungen |
| CNBC, MarketWatch/WSJ (Dow-Jones-Feeds), FT, Handelsblatt, tagesschau, Spiegel, FAZ, onvista, Der Aktionär, derStandard, Kurier, ORF | ✅ aktuell | ⚠️ Medien-Schlagzeilen: nur mit schriftlicher Erlaubnis |
| Yahoo Finance, Seeking Alpha pro Ticker | ✅ technisch | ❌ ToS: nicht kommerziell |
| CNN Money RSS | letzter Eintrag 2018 | ❌ eingestellt |
| GlobeNewswire, Accesswire, finanzen.net, boerse-express | 403 | ❌ Bot-Schutz |

### Nutzbare Feeds, 2. Runde (2026-10-29)

Geprüft wurde, was wir **rechtlich** nutzen können: Quellen, die Verbreitung wollen (Emittenten, Verteiler),
gemeinfreie oder zur Weiterverwendung freigegebene Behördendaten und Feeds mit eigenen Bedingungen.

**Emittenten-IR/Presse-RSS** [V] (gleiche Grundlage wie voestalpine: Kanal des Emittenten → `issuer_slug`, Titel + Link):

| Emittent | Feed | Stand | Hinweis |
|---|---|---|---|
| Deutsche Telekom | `telekom.com/en/investor-relations.rss.feed.427676.xml` (IR), `…/newsroom/latest-updates/media-information.rss.feed.418728.xml` (Presse) | aktuell | ✅ **aufgenommen** (`telekom_ir`, `telekom_presse`) |
| Henkel | `henkel.com/service/en/33922/asFeed/index.xml`, `…/39580/…` | aktuell | ✅ Feeds ohne Kanal-Titel; vor Aufnahme klären, welcher Presse und welcher IR ist |
| SAP | `news.sap.com/feed/` | aktuell | ✅ nur Titel: robots.txt sperrt KI-Bots → `text_policy = none`; viel Blog, wenig Börsenrelevantes |
| Symrise | `symrise.com/newsroom/press-releases/rss.xml` | aktuell | ✅ |
| DO & CO | `doco.com/feed/` | 2019 | ❌ tot |

Bei den übrigen ATX- und DAX-Emittenten fand der automatische Scan (Startseite, Presse- und IR-Seite) keinen Feed.
Viele dieser Seiten rendern per JavaScript oder blocken Bots; das ist kein Beweis, dass es keinen Feed gibt.

**Pressemitteilungs-Verteiler**

| Quelle | Status | Einschätzung |
|---|---|---|
| presseportal.de (news aktuell, ots DE) `/rss/wirtschaft.rss2`, `/rss/finanzen.rss2` | ✅ aktuell; `<source>` = Absender (Emittenten-Signal wie bei OTS) | ⚠️ Nutzungsbedingungen: Nutzung „für eigene Geschäftszwecke … nur mit schriftlicher Genehmigung der news aktuell GmbH“; es gibt ein Antragsverfahren für die automatisierte Einbindung in Websites Dritter. robots.txt sperrt KI-Bots |
| GlobeNewswire | 403 | ❌ Bot-Schutz |
| pressetext.com | 404 | Feed-URL nicht gefunden |
| EQS News | kein Feed | ❌ |

**SEC EDGAR, weitere Formulare** [V] (gemeinfrei, gleicher Endpunkt `getcurrent`):

| Formular | Inhalt | Parser |
|---|---|---|
| 10-Q, 10-K | Quartals- und Jahresberichte | ✅ funktioniert mit dem bestehenden `sec`-Parser (Filer-Zeilen) |
| 4 | Insider-Geschäfte (Directors' Dealings) | ⚠️ sehr viele Meldungen; der Emittent steht in der `(Issuer)`-Zeile, der Parser liest nur `(Filer)` |
| SCHEDULE 13D, 13D/A | Beteiligungen über 5 % mit Einflussabsicht | ⚠️ der Emittent steht in der `(Subject)`-Zeile → Parser-Erweiterung |
| 144 | geplante Verkäufe von Insidern | ⚠️ ebenso viel Rauschen |
| `SC 13D`/`SC 13G` | – | liefern 0 Einträge; EDGAR nennt die Formulare jetzt `SCHEDULE 13D/13G` |

**Behörden, Zentralbanken** [V] (keine Ticker; brauchen im UI einen Bereich „Markt & Politik“ für nicht verlinkte Meldungen):

| Quelle | Status | Nutzung |
|---|---|---|
| EZB `ecb.europa.eu/rss/press.html` | ✅ | Weiterverwendung mit Quellenangabe erlaubt |
| Fed `federalreserve.gov/feeds/press_all.xml` | ✅ | gemeinfrei |
| Bundesbank `bundesbank.de/service/rss/de/633286/feed.rss` | ✅ | Quellenangabe |
| EU-Kommission Presse `ec.europa.eu/commission/presscorner/api/rss?language=en&type=IP` | ✅ | Weiterverwendung nach Beschluss 2011/833/EU; Fusionskontrolle und Beihilfen nennen Unternehmen → Matcher kann verlinken |
| FDA Pressemitteilungen `fda.gov/…/press-releases/rss.xml` | ✅ | gemeinfrei; Zulassungen nennen Pharmafirmen |
| ESMA `esma.europa.eu/rss.xml` | ✅ | Quellenangabe |
| FMA `fma.gv.at/feed/` | ✅ | AT-Aufsicht |
| BaFin, OeNB | 404 | aktuelle Feed-URLs nicht gefunden |
| BLS, US Treasury | 403 | Bot-Schutz |

**Nasdaq Trade Halts** `nasdaqtrader.com/rss.aspx?feed=tradehalts` [V]: Handelsaussetzungen mit Symbol, Name, Grund (`ndaq:ReasonCode`).
Die eigenen Bedingungen („Trade Halt RSS Feed Terms and Conditions“) verbieten keine kommerzielle Nutzung, aber „editing, modifying,
misrepresenting“. Die Items haben **keinen Link** → wir müssten auf die Halts-Seite verlinken. ⚠️ vor Nutzung bei Nasdaq nachfragen.

## Nachtrag 2026-11-04: 3. Runde

In dieser Runde ging es um vier Fragen: Welche Pflichtmeldungen gibt es offiziell? Welche Emittenten haben eigene Feeds? Welche Lizenzbedingungen haben News-APIs? Welche Presseverteiler gibt es?
Was mit [V] markiert ist, haben wir am 2026-11-04 selbst abgerufen. Bei Lizenzen bedeutet [V], dass wir die Klausel auf der Seite des Anbieters gelesen haben.

### Wichtigster Fund: OeKB OAM hat einen öffentlichen JSON-Endpunkt [V]

`GET https://my.oekb.at/issuer-info/rest/public/meldedaten/iic?startPosition=0&offset=50`

Der Endpunkt braucht kein Login. Er liefert `{"anzahlTreffer": 54956, "dokumente": [...]}`, die neuesten Meldungen zuerst. Die jüngste Meldung war vom selben Tag.

- **Felder je Meldung:**
  - `emittent.name` und `emittent.lei`;
  - `isinBezug`, z. B. `["AT0000741301 (Aktie),"]` (muss bereinigt werden);
  - `meldetypCode`, `titel`, `uploadzeitpunkt` (in ms) und `sprachcode`;
  - `dateien[].id` (PDF über `…/iic/download?id=`).
- **Meldetypen:**
  - `EP_AD_HOC` (Ad-hoc-Meldungen);
  - `EP_EIGENGESCHAEFT_VON_FUEHRUNGSKRAFT` (Directors' Dealings);
  - `EP_AEND_WESENTL_STIMMRECHTSSCHWELLEN` (Stimmrechte);
  - `EP_JFB`, `EP_HJFB`, `EP_QUARTALBER` (Berichte);
  - `EP_HV_*` (Hauptversammlung);
  - `EP_BEKANNTGABEN_ZU_RUECKKAUFPROGRAMMEN` (Rückkäufe).
- **Sprache:** Meldungen kommen oft doppelt, auf Deutsch und auf Englisch. Wir deduplizieren deshalb über Emittent, Typ und Zeitpunkt.
- **Filter:** Nach Meldetyp filtert der Server nicht, das machen wir selbst.
- **Nutzung:** Laut *OeKB-OAM-Nutzungsgrundlagen V5.2* ist die Abfrage für Endnutzer „unentgeltlich“, und die Dokumente sind „an Dritte ohne Einschränkung“ weiterzugeben.
- **Einschränkung:** Die API ist **nicht dokumentiert**. Eine `robots.txt` gibt es nicht.
- **Konsequenz:** Die offene OeKB-Anfrage (siehe unten) bezieht sich jetzt konkret auf diesen Endpunkt.
- **Bewertung: hoch.** Das ist die offizielle Quelle für AT-Pflichtmeldungen mit ISIN. Sie schließt die Lücke, die die Wiener Börse lässt.
- **Umsetzung:** Wir brauchen einen eigenen Parser (JSON → RawArticle). Die Quelle bekommt `kind = adhoc` bzw. `filing`, damit gilt die ISIN als Emittenten-Signal.
- **Link:** Ein Deep Link auf die einzelne Meldung muss noch geklärt werden. Bis dahin verlinken wir auf das PDF.

### Behörden und Pflichtmeldungen

| Quelle | URL | Kennung | Lizenz | Wert |
|---|---|---|---|---|
| BaFin RSS (neue URLs) [V] | `bafin.de/DE/service/rss/_function/rssnewsfeed.xml?nn=150166`, außerdem `RSS_Presse`, `RSS_Massnahmen`, `RSS_Aufsicht` | Name im Titel | Weitergabe mit deutlicher Quellenangabe erlaubt; Links in neuem Fenster, keine Frames | mittel: `RSS_Massnahmen` nennt Firmen (meist Banken und Versicherer); generischer RSS-Parser |
| BaFin Directors' Dealings (CSV-Export) [V] | `portal.mvp.bafin.de/database/DealingsInfo/` | ISIN | ❌ robots.txt `Disallow: /` | hoch, aber nur mit Genehmigung der BaFin |
| Netto-Leerverkaufspositionen DE (Bundesanzeiger) [V] | `bundesanzeiger.de/pub/de/nlp` | ISIN | ❌ robots.txt `Disallow: /nlp` | nur mit Genehmigung |
| Unternehmensregister / Bundesanzeiger | – | – | ❌ keine API, robots.txt sperrt | nicht nutzbar, auf ESAP warten (EU, ca. 2027) [U] |
| filings.xbrl.org (ESEF) [V] | `filings.xbrl.org/api/filings?filter[country]=AT&sort=-date_added` | LEI | „no restrictions on the ways that the data can be used“ | niedrig: 601 AT-Filings, **keine deutschen**, Wochen Verzug; eher für Kennzahlen |
| FMA Leerverkäufe | fma.gv.at | – | Cloudflare-Challenge (403) | bei der FMA anfragen |
| OeNB RSS | – | – | – | nicht mehr vorhanden |
| SEC Press Releases [V] | `sec.gov/news/pressreleases.rss` | Name | gemeinfrei | mittel: Enforcement gegen Firmen |
| SEC Litigation, Admin. Proceedings, Trading Suspensions [V] | `sec.gov/enforcement-litigation/{litigation-releases,administrative-proceedings,trading-suspensions}/rss` | Name | gemeinfrei | niedrig: meist Einzelpersonen und Micro-Caps |
| SEC XBRL RSS [V] | `sec.gov/Archives/edgar/xbrlrss.all.xml` | CIK | gemeinfrei | niedrig: doppelt zu `getcurrent` 10-K/10-Q |
| FTC Competition [V] | `ftc.gov/feeds/press-release-competition.xml` | Name | gemeinfrei | mittel: Fusionsklagen gegen gelistete Firmen |
| NYSE Trade Halts [V] | `nyse.com/api/trade-halts/current` (JSON) | Ticker | ToS nicht geprüft | niedrig; Nasdaq Trade Halts (siehe oben) bevorzugen. Dort nur nachrichtenbezogene Codes (T1, T2, H10) nehmen, keine LULD-Pausen |
| EU-Kommission Wettbewerb, EMA, Federal Register, DOJ, CFTC [V] | – | Name | Weiterverwendung erlaubt bzw. gemeinfrei | niedrig: selten börsenrelevant oder ohne Firmennamen im Titel |

### Emittenten-Feeds, 2. Scan [V]

Geprüft wurden alle ATX- und DAX-Werte: die Seiten auf `rel=alternate` und RSS-Links abgesucht und 33 übliche Pfade abgefragt. Bei keinem der Treffer gibt es `tdmrep.json` oder eine Sperre für KI-Bots, sofern nicht anders vermerkt.

| Emittent | Feed | Neuester Eintrag | Art | Hinweis |
|---|---|---|---|---|
| Continental | `continental.com/en/general/rss/investors/`, `…/rss/press-releases/` | 2026-11-04 | IR + Presse | ✅ eigener IR-Feed |
| Fresenius | `fresenius.com/news/rss?type=investor`, `?type=media` (`/de/…` für Deutsch) | 2026-11-04 | IR + Presse | ✅ wird als `text/html` ausgeliefert, ist aber gültiges RSS |
| Heidelberg Materials | `heidelbergmaterials.com/en/rss/press-release` (auch `/de/`) | 2026-11-04 | Presse inkl. Ergebnisse | ✅ `/rss.xml` ist veraltet |
| BMW | `newsroom.bmwgroup.com/feed/` | 2026-11-04 | Presse + IR | ✅ |
| Volkswagen Konzern | `volkswagen-group.com/en/feeds/press-releases` | 2026-10-29 | Presse inkl. Ergebnisse | ✅ |
| Siemens | `press.siemens.com/global/en/rss.xml` | 2026-11-02 | Presse global | ✅ |
| adidas | `adidas-group.com/en/rss/news`, `/en/rss/publications` (auch `/de/`) | 2026-10-29 | Presse + IR | ✅ |
| Airbus | `airbus.com/en/generate-rss-feeds?type=all-press-releases` | 2026-11-02 | Presse | ✅ `pubDate` hat das Format `Mon, 11/02/2026 - 13:28` → Datumsparser erweitern |
| Qiagen | `corporate.qiagen.com/rss/pressrelease.aspx` (IR-Anbieter Q4) | 2026-10-21 | IR | ✅ |
| Infineon | `presseagentur.com/infineon/feed.php?lang=de` (RSS 1.0) | 2026-11-04 | Presse | ⚠️ auf der Domain der PR-Agentur, nicht auf infineon.com verlinkt; vorher bestätigen lassen |
| Covestro | `covestro.com/press//tagfeed/en/tags/corporate__news` | 2026-11-04 | Corporate News | ⚠️ robots.txt sperrt `/tagfeed/*`; dem Sinn nach nicht nutzen |
| CPI Europe (ex-Immofinanz) | `cpi-europe.com/de/rss.xml` | 2026-08-28 | IR | ✅ einziger brauchbarer ATX-Feed; Datum im JS-Format `Date.toString()` |
| Mayr-Melnhof | `mm.group/de/feed/` | 2026-11-04 | Blog/Stories | niedrig, kein IR |
| A1 Group | `a1.group/de/feed/` | 2025-12-16 | Presse | ❌ veraltet |

- **Ohne Feed:**
  - ATX: OMV, RBI, BAWAG, Wienerberger, Lenzing, Strabag, CA Immo, Palfinger, SBO, Porr, Uniqa, AT&S, Post, Erste, Verbund, Andritz.
  - DAX: Siemens Energy, Siemens Healthineers, Munich Re, Hannover Rück, BASF, Porsche, Daimler Truck, Deutsche Bank, Commerzbank, Beiersdorf, Brenntag, FMC, MTU, Rheinmetall, Sartorius, Vonovia, Zalando, Scout24, GEA, Deutsche Börse.
- **Bot-Schutz (403):** Allianz, Bayer, E.ON, RWE, Merck KGaA, Mercedes-Benz. Wer automatisierte Abrufe blockt, wird nicht gepollt.
- **Nicht prüfbar:** VIG, EVN, DHL.
- **Fazit:** Für ATX-Pflichtmeldungen führt der Weg über OeKB OAM, nicht über Emittenten-Feeds.

### News-APIs und Presseverteiler (Lizenz)

| Quelle | Preis | Ticker | Kommerzielle Anzeige? | Empfehlung |
|---|---|---|---|---|
| **Stock News API** Premium [V] | 49,99 $/Monat (Basic nur nicht-kommerziell) | US-Ticker; enthält PR Newswire, GlobeNewswire, Business Wire, Accesswire, Newsfile | **ja**: FAQ „Can I display the news on my site/app? Yes, as long as you are in compliance with the terms … of the news source“; Links in neuem Fenster, nur Text | ✅ beste günstige US-Option, umgeht den 403 von GlobeNewswire |
| **Marketaux** [V] | 29–199 $/Monat | Entities mit Symbol und Börse; Länderfilter de, at, us; deutsche Artikel | unklar: Es gibt nur Website-AGB („no commercial endeavors except … approved by us“) | ⏳ schriftlich anfragen; technisch die beste Lösung für DE/AT |
| **pressetext** (pta = Kapitalmarktmeldungen) [V] | Weiterverwendung gratis; XML-Feed 990 €/Jahr je Paket, kein freies RSS | ISIN in OAM | „Übernahme und Verwendung von Presse- und Kapitalmarktmitteilungen (pts, ptp, pta) ist kostenfrei“ | ✅ optional; die pta-Meldungen kommen gratis über OeKB OAM |
| **GDELT DOC 2.0** [V] | gratis | keine (Matcher) | GDELT-Daten: „unlimited and unrestricted use … commercial“ mit Zitat und Link; Schlagzeilen bleiben bei den Verlagen (EU-Ausnahme für Links und sehr kurze Auszüge) | ⚠️ Discovery für DE/AT-Medien; rechtlich vorher klären |
| GNews (49,99 €) / Mediastack (24,99 $) [V] | – | keine | kommerziell erlaubt, aber „no rights in third-party content“ | niedrig |
| NewsData.io [V] | ab ca. 200 $ | keine | erlaubt, auf eigenes Risiko, Credit „Powered by NewsData.io“ | niedrig |
| Event Registry / NewsAPI.ai | 90 $/Monat | Entities | Anzeigebedingungen nicht gelesen [U] | ⏳ prüfen |
| EQS News (DGAP) | Vertrag | ISIN | keine öffentlichen Bedingungen | ⏳ Angebot anfragen; größte DE-Ad-hoc-Lücke |
| Finnhub paid, Alpaca, Tiingo, EODHD, FMP, Intrinio, Massive [V] | 50–3.500 $ | US | Anzeige nur mit Enterprise- oder Display-Lizenz bzw. Freigabe; Alpaca und Finnhub verbieten Weitergabe | ❌ |
| Google News RSS [U], Bing News API | – | – | Google: nur persönlich; Bing: am 2025-08-11 eingestellt | ❌ |
| IRW-Press, Common Crawl News | – | – | keine Nutzungsrechte bzw. eigenes Risiko | ❌ |

### Empfohlene Reihenfolge

1. **OeKB OAM** (AT-Pflichtmeldungen mit ISIN). Neuer JSON-Parser; parallel die Anfrage bei der OeKB.
2. **Emittenten-Feeds DAX**: Continental, Fresenius, Heidelberg Materials, BMW, VW, Siemens, adidas, Qiagen, Airbus. Dafür reicht der vorhandene `pressrss`-Parser mit `issuer_slug`; für Airbus und CPI Europe muss der Datumsparser erweitert werden.
3. **BaFin `RSS_Massnahmen`/`RSS_Presse`, FTC Competition, SEC Press Releases.** Generischer Parser; Meldungen ohne Firma im Titel landen im Bereich „Markt & Politik“.
4. **Stock News API Premium** (49,99 $/Monat) für die US-Breite inkl. GlobeNewswire. Sentiment des Anbieters nur intern verwenden.
5. Anfragen: Marketaux (DE/AT), EQS (DE-Ad-hoc), BaFin (Directors' Dealings CSV).

## Stammdaten

| Quelle | Inhalt | Lizenz | Status |
|---|---|---|---|
| Nasdaq Trader `nasdaqlisted.txt`, `otherlisted.txt` [V] | ca. 13k US-Symbole, Name, Börse | frei | ✅ |
| SEC `company_tickers_exchange.json` [U] | CIK, Name, Ticker, Börse | gemeinfrei | ✅ (CIK für 8-K) |
| OpenFIGI [V] | ISIN ↔ Ticker/MIC; mit gratis Key 250 req/min × 100 Jobs | frei mit Key | ✅ |
| Xetra „All tradable instruments“ CSV [U] | ISIN, WKN, Mnemonic | frei | ✅ (den neuen URL-Pfad ermitteln) |
| GLEIF Golden Copy [U] | LEI, rechtlicher Name, LEI↔ISIN | CC0 | ✅ (Namen fürs Matching) |
| ATX-, DAX-, MDAX-, SDAX-Mitglieder | Indexzusammensetzung | manuell gepflegt | ✅ Seed-CSV |

## Kosten MVP

0 $/Monat für Daten, zuzüglich OpenRouter. Ein lizenzierter US-News-Feed mit Ticker-Tags, falls nötig, kostet 30–450 $/Monat.

## Schriftlich zu klären (vor dem Livegang)

- APA-Comm: Lizenz für OTS-Text für TDM/LLM bzw. eine API.
- OeKB: maschinenlesbarer Bezug der OAM-Daten, konkret die Nutzung von `my.oekb.at/issuer-info/rest/public/meldedaten/iic` und Deep Links auf einzelne Meldungen.
- BaFin: automatisierter Bezug der Directors'-Dealings-Datenbank (robots.txt sperrt das Portal).
- Marketaux: kommerzielle Anzeige von Schlagzeile + Link und interne LLM-Verarbeitung.
- EQS: Lizenzangebot für den News-Feed (DE-Ad-hoc).
- Infineon: ob der Feed auf presseagentur.com offiziell ist.
- FinanzNachrichten: ob die Übernahme ausdrücklich auch ein kommerzielles Portal abdeckt (kein tdmrep, geprüft 2026-10-29).
- Alpha Vantage: interne kommerzielle Nutzung als Baseline.
- Business Wire / PR Newswire: Nutzungsbedingungen für die Anzeige auf einem kommerziellen Portal.
- Nasdaq: Lizenz für Schlagzeilen und Deep Links (die Terms erlauben nur private Nutzung).
- Medien (derStandard, Handelsblatt, CNBC): Erlaubnis zur Anzeige von Schlagzeilen mit Link.
- news aktuell (presseportal.de): Antrag auf automatisierte Einbindung der ots-Meldungen.
- Nasdaq Trade Halts: Anzeige ohne Original-Link, Verlinkung auf die Halts-Seite.
