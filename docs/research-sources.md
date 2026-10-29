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
| Deutsche Telekom | `telekom.com/en/investor-relations.rss.feed.427676.xml` (IR), `…/newsroom/latest-updates/media-information.rss.feed.418728.xml` (Presse) | aktuell | ✅ |
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
- OeKB: maschinenlesbarer Bezug der OAM-Daten.
- FinanzNachrichten: ob die Übernahme ausdrücklich auch ein kommerzielles Portal abdeckt (kein tdmrep, geprüft 2026-10-29).
- Alpha Vantage: interne kommerzielle Nutzung als Baseline.
- Business Wire / PR Newswire: Nutzungsbedingungen für die Anzeige auf einem kommerziellen Portal.
- Nasdaq: Lizenz für Schlagzeilen und Deep Links (die Terms erlauben nur private Nutzung).
- Medien (derStandard, Handelsblatt, CNBC): Erlaubnis zur Anzeige von Schlagzeilen mit Link.
- news aktuell (presseportal.de): Antrag auf automatisierte Einbindung der ots-Meldungen.
- Nasdaq Trade Halts: Anzeige ohne Original-Link, Verlinkung auf die Halts-Seite.
