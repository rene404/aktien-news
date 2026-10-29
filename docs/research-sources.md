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
- FinanzNachrichten: ob die Übernahme ausdrücklich auch ein kommerzielles Portal abdeckt; TDM-Status (tdmrep) noch nicht geprüft.
- Alpha Vantage: interne kommerzielle Nutzung als Baseline.
- Business Wire / PR Newswire: Nutzungsbedingungen für die Anzeige auf einem kommerziellen Portal.
