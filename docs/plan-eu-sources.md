# Plan: Pflichtmeldungen aus FR, ES, NL, Nordics und UK

Entwurf vom 2026-11-05. Grundlage ist die Recherche in `docs/research-sources.md`, Abschnitt „Nachtrag 2026-11-05: Pflichtmeldungen in anderen EU-Ländern“.
Vorbild ist die OeKB-OAM-Anbindung (`ingest/oekb`, umgesetzt am 2026-11-05).

## Ausgangslage

- **Ohne Katalog keine Anzeige.** Der Feed zeigt nur Cluster mit mindestens einem `linked`-Eintrag.
  Wir brauchen also Stammdaten für die französischen, spanischen und niederländischen Firmen, sonst bleiben ihre Meldungen unsichtbar.
- **Märkte:** `sources.market` erlaubt nur `AT`, `DE`, `US` und `MULTI` (CHECK in `00001_init.sql`, `catalog.knownMarkets`).
  Die Website kennt nur XWBO, XETR, XNYS und XNAS.
- **Matching über LEI fehlt.** `companies.lei` gibt es zwar, der Matcher nutzt die Spalte aber nicht.
  `match_method` hat noch keinen Wert `lei`. Für AFM (NL) und später die FCA (UK) ist die LEI jedoch die einzige harte Kennung.
- **Abruf in einem Schritt:** Der Runner holt eine URL pro Quelle (`Fetcher.Fetch(url)`).
  DILA (FR) braucht aber zwei Schritte: erst das Verzeichnis lesen, dann die neuen Archive laden.
  Dafür gibt es schon die Möglichkeit, je Parser einen eigenen Fetcher zu setzen (`Runner.Fetchers`, wie bei SEC).

## Rahmen für alle Aufgaben

- **Öffentlich** zeigen wir weiterhin nur Titel, Quelle, Link, Zeit und Ticker (AGENTS.md, Rechtsregel 1).
- **LLM:** Es bekommt nur Titel, `text_policy = none`.
- **Quellenpflichten** übernehmen wir wörtlich:
  - AFM: Angabe „AFM“ als Quelle.
  - CNMV: Inhalte „fielmente, sin manipular“.
- **Arbeitsweise:** Jede Aufgabe bekommt einen eigenen Branch von `develop` und läuft per TDD. Gemergt wird mit `--no-ff`.
- **Gates:**
  - Backend: `docker compose --profile test run --rm backend-test sh -c "gofmt -l .; go vet ./... && go test ./..."`.
  - Frontend: `docker compose --profile test run --rm web-test`.
- **Fixtures:** Parser-Tests laufen gegen gespeicherte Antworten in `testdata/`, nie live (AGENTS.md).

## Entscheidungen (2026-11-05)

1. **Indizes:** CAC 40, IBEX 35 und AEX 25, jeder mit eigenem Tab und eigener Aktienliste.
2. **Directors' Dealings:** Keine Personennamen im Titel, also „STMicroelectronics N.V.: Eigengeschäft einer Führungskraft“.
3. **Nordics und UK werden mitgebaut.**
   - Die Seite dient derzeit dem privaten Gebrauch; die Quellen werden getestet und lassen sich jederzeit abschalten (`sources.enabled`).
   - Die Anfrage bei Nasdaq und die Rechtsprüfung der FCA müssen vor einem öffentlichen Betrieb erledigt sein.
   - Die FCA-Quelle schicken wir trotzdem nicht ans LLM, weil die Seite einen TDM-Vorbehalt erklärt.
4. **Frankreich:** Der Link führt auf die PDF-Datei der Meldung.

## A. Grundlagen

- [x] **1. Neue Märkte und Börsen.** Branch `feature/markets-eu`.
  - **Umgesetzt (2026-11-08):** Migration `00007_markets_eu.sql`; `knownMarkets` und `knownMICs` im Seed; `EXCHANGES` im Frontend.
    Die Down-Migration schlägt fehl, solange FR-, ES- oder NL-Quellen existieren, damit nichts stillschweigend verloren geht.
    Offen für Aufgabe 4/7: Der Tab „SEC-Filings“ zeigt alle Quellen der Art `filing`, also künftig auch AFM; Beschriftung dann anpassen.
  - **Dateien:**
    - neue Migration `00007_markets_eu.sql`;
    - `backend/internal/catalog/seed.go`;
    - `frontend/lib/filters.ts` und die zugehörigen Tests.
  - **Inhalt:**
    - Der CHECK auf `sources.market` erlaubt zusätzlich `FR`, `ES`, `NL`; Down-Migration zurück auf die alte Liste.
    - `knownMarkets` wird entsprechend erweitert.
    - `EXCHANGES` bekommt `XPAR` (Euronext Paris → FR), `XMAD` (Bolsa de Madrid → ES) und `XAMS` (Euronext Amsterdam → NL).
  - **Fertig, wenn:**
    - Die Migration läuft up → down → up.
    - Der Seed lehnt unbekannte Märkte weiter ab.
    - `sourcesFor` liefert auf einer Pariser Aktienseite nur FR- und MULTI-Quellen.

- [x] **2. Matching über LEI.** Branch `feature/matching-lei`.
  - **Umgesetzt (2026-11-08):** Migration `00008_match_lei.sql`; `RawArticle.IssuerLEI`; `CompanyEntry.LEI` und ein LEI-Index im Matcher.
    `Input.ISINIsIssuer` heißt jetzt `IssuerGrade`, weil das Merkmal für ISIN und LEI gilt.
    Der OeKB-Parser setzt `emittent.lei`. In der Fixture `iic.json` trägt die RLB-NÖ-Wien-Meldung (ohne ISIN) jetzt ihre echte LEI wie auf der Live-Seite.
    Die Down-Migration baut den Enum ohne `lei` neu und bricht ab, solange es Verknüpfungen mit Methode `lei` gibt.
  - **Dateien:**
    - Migration `00008_match_lei.sql` (`ALTER TYPE match_method ADD VALUE 'lei'`);
    - `ingest/raw.go` (neues Feld `IssuerLEI`);
    - `matching/match.go`, `matching/persist.go` und `ingest/runner.go`.
  - **Inhalt:**
    - Die LEI kommt in den Matching-Index, zugeordnet zur Firma.
    - Aus einer `adhoc`- oder `filing`-Quelle gilt die LEI als Emittenten-Signal mit Konfidenz 0.98, Methode `lei`.
    - Steht die LEI nicht im Katalog, hat das keine Wirkung; dann zählen die übrigen Signale wie bisher.
    - Die OeKB-OAM-Meldungen bringen `emittent.lei` mit. Der OeKB-Parser setzt sie gleich mit, das ist ein Gewinn für AT-Meldungen ohne ISIN.
  - **Fertig, wenn** Tabellentests in `matching` belegen:
    - Eine passende LEI verknüpft.
    - Eine unbekannte LEI ändert nichts.
    - In einer News-Quelle zählt die LEI nicht als Emittenten-Signal.
    - Eine OeKB-Meldung ohne ISIN, aber mit LEI wird verknüpft.

- [x] **3. Stammdaten FR, ES, NL.** Branch `feature/catalog-eu`.
  - **Umgesetzt (2026-11-08):** 102 neue Firmen, 104 Listings und 105 Indexmitgliedschaften (`valid_from` 2026-11-08); Airbus bestand schon und bekam das Pariser Listing und seine LEI.
    - **Zusammensetzung:** CAC 40 und IBEX 35 sind 2026 unverändert (Euronext und Comité Asesor Técnico, März, Juni und September). Der AEX hat seit dem 22.09.2025 30 Werte; die niederländische Wikipedia, Stand 22.06.2026, nennt 29. InPost ist mitgezählt, weil die Aktie trotz laufender Übernahme noch notiert ist. Nach dem Delisting muss InPost mit `valid_to` heraus.
    - **ISIN und LEI:** LEI aus ESMA FIRDS, rechtlicher Name aus GLEIF. Die ISIN ist die aktive Aktien-ISIN an der Heimatbörse laut FIRDS; bei Dassault Systèmes, Eurofins, Michelin, Aena und Unilever ist das schon die neue ISIN nach Split oder Umstellung.
    - **Aliasse:** Der rechtliche Name steht nur als `legal_name`. Als `issuer_name`-Alias würde ein nackter Name wie „ORANGE“ als voller Firmenname zählen und „Orange County Bancorp“ verknüpfen. Für spanische Firmen kommt die CNMV-Schreibweise „…, S.A.“ als `issuer_name` dazu.
    - **Ticker:** Kürzel von Euronext Paris, Amsterdam und Madrid sind Mnemonics („EN“, „RED“, „NN“). Als Wort im Titel zählen sie nur, wenn sie zugleich der Name sind (IAG, BBVA, ASML, ACS); Cashtags gelten weiter. Ermittelt gegen alle 15.364 Titel der Dev-Datenbank.
    - **Prüfung:** `ValidLEI` (ISO 17442, mod 97) im Seed.
    - **Offen:** Ein ADR-Duplikat aus dem SEC-Import (etwa „ASML HOLDING NV“ an der Nasdaq, uncurated) bleibt eine eigene Firma.
  - **Dateien:** `backend/seed/companies.csv`, `listings.csv`, `index_memberships.csv`, `company_aliases.csv`.
  - **Inhalt:**
    - Pro Firma: Name, rechtlicher Name, Land, **LEI**, ISIN, Ticker, MIC und Index (CAC40, IBEX35, AEX).
    - **Aliasse:** die Schreibweise der CNMV („VIDRALA, S.A.“, „MAPFRE, S.A.“) als `issuer_name`, die AFM-Schreibweise ebenso.
    - **Woher die LEI kommt:** einmalig über ESMA FIRDS
      (`registers.esma.europa.eu/solr/esma_registers_firds/select?q=isin:<ISIN>&wt=json`).
      Das macht ein Skript im Scratchpad, eingecheckt wird nur das Ergebnis-CSV.
    - Indexzusammensetzung wie bisher manuell, Stand mit `valid_from`.
  - **Fertig, wenn:**
    - Der Seed ist idempotent.
    - Ein Test prüft das LEI-Format (20 Zeichen, Prüfziffer nach ISO 17442, mod 97) und das ISIN-Format.
    - Jede Firma der drei Indizes hat ein Listing.

- [ ] **4. Website: Tabs und Aktienliste.** Branch `feature/web-eu-sections`.
  - **Dateien:**
    - `frontend/lib/filters.ts` (`SECTIONS`), `frontend/lib/stocklist.ts` (`LIST_INDICES`);
    - die zugehörigen Tests.
  - **Inhalt:** Tabs „CAC 40“, „IBEX 35“ und „AEX“ mit dem jeweiligen Markt; die Aktienliste bekommt dieselben Indizes.
  - **Fertig, wenn:**
    - Die Tabs zeigen nur verknüpfte Meldungen ihres Index.
    - Die Tab-Leiste scrollt mit Verlauf (vorhandener `useScrollRow`).
    - Der Gate ist grün.
    - Screenshot auf Handy und Desktop.

## B. Neue Quellen

- [ ] **5. Spanien: CNMV.** Branch `feature/ingest-cnmv`.
  - **Dateien:**
    - `ingest/cnmv/parse.go` mit Tests und `testdata/` (beide Feeds, einmal leer, einmal kaputt);
    - `ingest/parsers/parsers.go` und `seed/sources.csv`.
  - **Quellen:**
    - `cnmv_ip`: Insiderinformation, `kind = adhoc`, `…/informacion-privilegiada/RSS.asmx/GetNoticiasCNMV`.
    - `cnmv_oir`: sonstige relevante Information, `kind = filing`, `…/Otra-Informacion-Relevante/RSS.asmx/GetNoticiasCNMV`.
    - Für beide: Markt `ES`, Abruf alle 15 Minuten. Der Feed enthält nur den laufenden Handelstag und hat `ttl = 15`.
  - **Parser:**
    - Die XML-Elemente sind großgeschrieben (`<Channel>`, `<Title>`), daher ein eigener Parser statt `pressrss`.
    - `Title` ist der Emittent, also `IssuerHint`.
    - Der Betreff steht in `description` nach Uhrzeit, Datum und Kategorie.
    - Titel = „Emittent: Betreff“, beides unverändert übernommen.
    - `ExternalID` = `nreg` aus dem Link; `URL` = Link der CNMV.
  - **Rechtlicher Hinweis zur Doku:** In `description` steht bei der CNMV der Betreff der Meldung, kein Fließtext oder Agenturtext.
    Der Parser übernimmt nur den Betreff, nichts sonst; das kommt als Kommentar an `RawArticle`.
  - **Fertig, wenn:**
    - Die Fixture-Tests sind grün.
    - Im Live-Lauf werden IBEX-Werte über den Namen verknüpft (Alias aus Aufgabe 3).
    - `sources.csv` nennt die Nutzungsbedingungen in `terms_url`.

- [ ] **6. Frankreich: AMF über DILA Open Data.** Branch `feature/ingest-dila-amf`.
  - **Dateien:**
    - `ingest/dila/fetch.go`: Fetcher, liest das Verzeichnis, lädt die Archive und entpackt die XML-Dateien.
    - `ingest/dila/parse.go`, Tests mit einem kleinen nachgebauten `tar.gz` in `testdata/`.
    - `cmd/worker/main.go` (`Fetchers["dila_…"]`), `parsers.go`, `sources.csv`.
  - **Abruf:**
    - Verzeichnis `https://echanges.dila.gouv.fr/OPENDATA/AMF/` lesen.
    - Geladen werden die Archive `amf-gz_DDMMYYYY_HHMM.tar.gz` der letzten 26 Stunden.
    - Aus dem `tar.gz` liest der Fetcher nur `xml/…`; PDFs überspringt er beim Entpacken.
    - Archive ändern sich nicht. Deshalb behält der Fetcher die entpackten XML-Dateien der letzten 26 Stunden im Speicher, nach Archivnamen.
      So laden die beiden Quellen (siehe unten) nicht doppelt; ein Archiv ist bis zu 12 MB groß.
    - Abruf alle 60 Minuten. DILA veröffentlicht alle 1–3 Stunden.
  - **Quellen:**
    - `amf_adhoc`: Insiderinformation, `kind = adhoc`.
    - `amf_filings`: übrige Typen, `kind = filing`.
    - Für beide: Markt `FR`. Die Typcodes (`INF_STP_PRI`) ordnen wir anhand der Typtabellen bzw. des XSD von DILA zu; diese Zuordnung ist Teil der Aufgabe.
  - **Parser:**
    - `ExternalID` = `UIN_IDT_UIN`.
    - Titel = `INF_TIT_INF`; nennt er den Emittenten nicht, kommt dieser davor (wie bei OeKB).
    - `PublishedAt` = `INF_DAT_EMT`.
    - Kennungen: `ISO_CD_ISI` (ISIN), `ISO_CD_LEI` (LEI), `ISO_NOM_SOC` (Name als Hint).
    - Meldungen mit `INF_UPG_STS` ≠ `NEW` (Korrekturen, Rückzüge) werden vorerst übersprungen.
  - **Link:** die PDF-Datei der Meldung (Entscheidung 4). Wir prüfen, welche öffentliche URL das PDF zu `FichierDeContenu INF_FIC_NOM` ausliefert, und nehmen sie.
  - **Fertig, wenn:**
    - Fixture-Tests decken ab: Ad-hoc und Filing getrennt, ISIN und LEI gesetzt, ein kaputtes Archiv ergibt einen Fehler, und ein Archiv außerhalb der 26 Stunden wird nicht geladen.
    - Im Live-Lauf werden CAC-Werte verknüpft.

- [ ] **7. Niederlande: AFM Directors' Dealings.** Branch `feature/ingest-afm-dd`.
  - **Dateien:** `ingest/afm/parse.go` mit Tests und `testdata/`, `parsers.go`, `sources.csv`.
  - **Quelle:**
    - `afm_dd`: `kind = filing`, Markt `NL`, Name „AFM – Transacties leidinggevenden“ (Quellenangabe).
    - `https://www.afm.nl/export.aspx?type=0ee836dc-5520-459d-bcf4-a4a689de6614&format=xml`.
    - Komplettdump mit knapp 5 MB, daher Abruf alle 60 Minuten.
  - **Parser:**
    - Nur Meldungen der letzten 14 Tage; der Dump enthält die ganze Historie.
    - `ExternalID` = `meldingid`. Kennungen: `lei`, `uitgevendeinstelling` (Name als Hint).
    - Titel nach Entscheidung 2: „<Emittent>: Eigengeschäft einer Führungskraft“, ohne Personennamen.
    - Datum: Wir prüfen, ob der Export ein Meldedatum hat. Wenn nicht, nehmen wir `transactiedatum` (US-Format M/D/YYYY) und setzen 00:00 Uhr Europe/Amsterdam.
    - Link: Detailseite im AFM-Register, falls es eine gibt, sonst die Registerseite.
  - **Fertig, wenn:**
    - Die Fixture-Tests sind grün (inkl. US-Datumsformat und altem Eintrag).
    - Im Live-Lauf werden AEX-Werte über die LEI verknüpft.
    - Im UI ist „AFM“ als Quelle sichtbar.

## C. Nordics und UK (zum Testen, Entscheidung 3)

- [ ] **8. Nordics: Nasdaq Nordic Company Announcements.**
  - **Parallel:** schriftliche Anfrage bei Nasdaq zur Nutzung von `api.news.eu.nasdaq.com`; Pflicht vor einem öffentlichen Betrieb.
  - **Umfang:**
    - Markt `NORDIC` (bzw. SE, FI, DK).
    - Katalog für OMXS30, OMXH25 und OMXC25.
    - JSON-Parser: `disclosureId`, `headline`, `company`, `releaseTime`, `messageUrl`, `cnsCategory` → adhoc/filing.

- [ ] **9. UK: FCA National Storage Mechanism.**
  - **Rechtsprüfung parallel, vor einem öffentlichen Betrieb:** `robots.txt` erklärt einen TDM-Vorbehalt nach Art. 4 DSM-RL (`Content-Signal: ai-train=no, use=reference`).
    - Unser TDM-Check liest bisher nur `tdmrep.json`. Er muss auch `Content-Signal` in `robots.txt` erkennen.
    - Zu klären ist, ob Titel aus so einer Quelle überhaupt ans LLM dürfen. Falls nicht, braucht `sources` ein Merkmal, das die Klassifizierung abschaltet.
  - **Umfang:**
    - Markt `UK`, Katalog FTSE 100.
    - Die Quelle wird von der LLM-Klassifizierung ausgenommen.
    - Parser für den POST auf `api.data.fca.org.uk/search?index=nsm-search` (Elasticsearch-JSON, LEI).

## D. Doku

- [ ] **10. Doku nachziehen.** Branch `chore/docs-eu-sources`.
  - `README.md`: Märkte und Tabs.
  - `docs/design.md`: LEI als Matching-Signal, Fetcher in zwei Schritten (DILA).
  - `docs/research-sources.md`: Status je Quelle.
  - AGENTS.md: der Datenhinweis zu ISIN gilt auch für LEI.

## Reihenfolge

1 → 2 → 3 → 4 sind die Voraussetzung, danach 5 (am einfachsten), 6, 7, 8, 9. Die Anfragen an Nasdaq und die Rechtsprüfung der FCA laufen parallel.
Grober Aufwand:

| Aufgabe | Aufwand |
|---|---|
| 1 | klein |
| 2 | mittel |
| 3 | mittel (Datenpflege) |
| 4 | klein |
| 5 | klein |
| 6 | groß (Archiv, Typcodes, Cache) |
| 7 | klein–mittel |
