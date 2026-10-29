# Plan Phase 2: mehr Quellen, US-Stammdaten, Clustering, Text-Regeln

Entwurf vom 2026-10-28, freigegeben am 2026-10-29. Entschieden: US-Universum gestuft (alle NYSE/Nasdaq), öffentliche Liste ein Eintrag pro Cluster. Grundlage: `docs/design.md`, `docs/research-sources.md`
und ein Erreichbarkeits-Check am 2026-10-28:

- OTS `/rss/finanzen` und `/rss/wirtschaft`: 200, Aussender im Feld `author`. `tdmrep.json` liegt im
  **Root** (`/tdmrep.json`, `tdm-reservation: 1`), nicht unter `/.well-known/`.
- PR Newswire `news-releases-list.rss`: 200, Aussender im Feld `dc:contributor`, keine Ticker im Feed.
- **Business Wire: Der RSS-Kanal ist abgeschaltet** („The RSS channel you requested was deactivated“).
  Die Quelle entfällt.
- SEC (`sec.gov`) ist hinter der Firmen-Firewall nicht erreichbar. Die Testdaten werden nach dem
  dokumentierten Format gebaut und später in einem anderen Netz gegen echte Antworten geprüft.

Jede Aufgabe läuft auf einem eigenen `feature/phase2-*`-Branch per TDD. Für alle gilt der Gate
`docker compose --profile test run --rm backend-test sh -c "go vet ./... && go test ./..."`.

- [x] **1. TDM-Prüfung und wirksame `text_policy`**
  Files: `backend/internal/ingest/fetch/tdm.go`, `backend/internal/ingest/fetch/tdm_test.go`, `backend/internal/ingest/policy.go`, `backend/internal/ingest/policy_test.go`, `backend/internal/store/queries/ingest.sql`, `backend/internal/ingest/runner.go`, `backend/internal/ingest/runner_test.go`
  What: Der Worker prüft pro Quelle höchstens einmal in 24 h den Header `TDM-Reservation` sowie `/.well-known/tdmrep.json` und `/tdmrep.json` und schreibt das Ergebnis nach `sources.tdm_reserved`/`tdm_checked_at`. `ingest.EffectivePolicy` liefert `none`, sobald ein Vorbehalt besteht.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/ingest/...`
  Done: Die httptest-Fälle Root-Datei mit Vorbehalt (wie bei OTS), `.well-known`, Header, 404 (= kein Vorbehalt) und Netzwerkfehler (= Vorbehalt bleibt unverändert) sind grün. `EffectivePolicy(full_text, reserved)` ergibt `none`.

- [x] **2. Presse-RSS: APA-OTS und PR Newswire**
  Files: `backend/internal/ingest/pressrss/parse.go`, `backend/internal/ingest/pressrss/parse_test.go`, `backend/internal/ingest/pressrss/testdata/ots_finanzen.xml`, `backend/internal/ingest/pressrss/testdata/prn.xml`, `backend/seed/sources.csv`, `backend/cmd/worker/main.go`
  What: Ein Parser für RSS-Feeds, die den Aussender in `author` bzw. `dc:contributor` liefern, setzt diesen als `IssuerHint`. `description` wird ignoriert. Neu sind die Quellen `ots_finanzen`, `ots_wirtschaft` und `prn_all`.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/ingest/pressrss/` · `docker compose up -d --build worker && docker compose logs worker --tail 30`
  Done: Fixture-Tests mit echten Feeds sind grün, ebenso die Fälle leer, kaputt und ohne Aussender. `RawArticle` bleibt ohne Textfeld. Der Live-Lauf zeigt `ok` für alle drei Quellen, und eine Lenzing-Meldung von OTS ist über `issuer` verlinkt.

- [x] **3. IR-Feeds: Quelle = Emittent**
  Files: `backend/internal/store/migrations/00005_source_issuer.sql`, `backend/internal/store/queries/catalog.sql`, `backend/internal/catalog/seed.go`, `backend/internal/catalog/seed_test.go`, `backend/seed/sources.csv`, `backend/internal/ingest/runner.go`, `backend/internal/ingest/runner_test.go`
  What: Die neue Spalte `sources.issuer_company_id` wird im Seed über eine Spalte `issuer_slug` gesetzt. Ist sie belegt, verlinkt der Runner jeden Artikel der Quelle mit `issuer` (1.0). Erster Feed ist voestalpine.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/store/ ./internal/catalog/ ./internal/ingest/`
  Done: up → down → up ist grün. Ein Artikel ohne Firmennamen im Titel aus einer IR-Quelle ist `linked`. Ein unbekannter `issuer_slug` lässt den Seed mit Fehler abbrechen.

- [x] **4. SimHash-Clustering**
  Files: `backend/internal/dedupe/simhash.go`, `backend/internal/dedupe/simhash_test.go`, `backend/internal/store/queries/clusters.sql`, `backend/internal/ingest/runner.go`, `backend/internal/ingest/runner_test.go`
  What: Jeder neue Artikel bekommt einen SimHash über den normalisierten Titel. In derselben Transaktion schließt er sich dem nächsten Cluster im 72-h-Fenster an (Hamming ≤ 3) oder gründet einen neuen. Repräsentant ist die Quelle mit dem höchsten `weight`, bei Gleichstand die früheste.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/dedupe/ ./internal/ingest/`
  Done: Dieselbe EQS-Meldung über FN und OTS landet in einem Cluster mit `size` 2, und der Repräsentant ist die Ad-hoc-Quelle. Ähnlich lautende, aber verschiedene Meldungen bleiben getrennt (z. B. die Korrekturen vom 23.10. und 27.10.). Ein Artikel außerhalb von 72 h gründet einen neuen Cluster.

- [ ] **5. Öffentliche Liste: ein Eintrag pro Cluster**
  Files: `backend/internal/store/queries/news.sql`, `backend/internal/service/news.go`, `backend/internal/service/news_test.go`, `backend/internal/httpapi/news_test.go`
  What: `ListNews` liefert pro Cluster nur den Repräsentanten (Titel, Link, Quelle) mit den Tickern aller verlinkten Artikel des Clusters. Filter und Cursor beziehen sich auf den Repräsentanten.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/service/ ./internal/httpapi/`
  Done: Zwei Artikel eines Clusters ergeben einen Eintrag. Der Ticker-Filter findet den Cluster auch dann, wenn nur das Nicht-Repräsentanten-Mitglied verlinkt ist. Die Pagination bleibt ohne Duplikate. Der DTO-Rechtstest bleibt unverändert grün.

- [ ] **6. US-Stammdaten aus SEC `company_tickers_exchange.json`**
  Files: `backend/internal/store/migrations/00006_company_cik.sql`, `backend/internal/catalog/sec.go`, `backend/internal/catalog/sec_test.go`, `backend/internal/catalog/testdata/company_tickers_exchange.json`, `backend/cmd/seed/main.go`, `backend/internal/matching/match.go`, `backend/internal/matching/match_test.go`
  What: `companies` bekommt `cik` (unique, nullable) und `curated` (Default true). `seed -sec-file <pfad>` ergänzt bestehende US-Firmen um den CIK und legt fehlende NYSE-/Nasdaq-Titel als `curated = false` an. Für nicht kuratierte Firmen nutzt der Matcher nur starke Signale: Emittent/CIK, ISIN, Cashtag, vollen Namen mit Rechtsform. Ticker als Wort und Kernname zählen dort nicht.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/store/ ./internal/catalog/ ./internal/matching/`
  Done: Der Import ist idempotent, und vorhandene kuratierte Firmen behalten Namen und Aliase. Nach dem Import wird „A“, „ON“ oder „IT“ im Titel nicht verlinkt. „Agilent Technologies, Inc.“ wird über den vollen Namen verlinkt.

- [ ] **7. SEC EDGAR 8-K**
  Files: `backend/internal/ingest/raw.go`, `backend/internal/ingest/sec/parse.go`, `backend/internal/ingest/sec/parse_test.go`, `backend/internal/ingest/sec/testdata/current_8k.xml`, `backend/internal/config/config.go`, `backend/internal/matching/match.go`, `backend/internal/matching/match_test.go`, `backend/seed/sources.csv`, `backend/cmd/worker/main.go`
  What: Ein Parser für den Atom-Feed „latest filings“ (8-K) liest CIK, Firmenname, Item-Nummern und Link. Der CIK wird als Emittent mit 1.0 zugeordnet. Abgerufen wird nur mit `SEC_USER_AGENT` (Kontakt-E-Mail); ohne ihn wird die Quelle `skipped`.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/ingest/... ./internal/matching/`
  Done: Fixture-Tests sind grün (CIK mit führenden Nullen, mehrere Items, leer, kaputt). Ein bekannter CIK ist `linked` (issuer), ein unbekannter erzeugt keinen Link. Ohne `SEC_USER_AGENT` steht der Lauf in `ingest_runs` auf `skipped`.

- [ ] **8. Text-Speicherung nur nach Policy, plus Retention-Purge**
  Files: `backend/internal/ingest/texts.go`, `backend/internal/ingest/texts_test.go`, `backend/internal/store/queries/texts.sql`, `backend/cmd/worker/main.go`
  What: `ingest.StoreText` schreibt nach `article_texts` nur bei wirksamer Policy `full_text`, sonst nicht. `purge_after` = jetzt + `TEXT_RETENTION` (Default 30 Tage). Der Worker löscht einmal pro Stunde abgelaufene Texte. Den Abruf von 8-K-Volltexten erledigt Phase 3 zusammen mit dem Classifier, weil erst er Text braucht.
  Command: `docker compose --profile test run --rm backend-test go test ./internal/ingest/`
  Done: Mit Policy `none` bzw. bei TDM-Vorbehalt wird nichts gespeichert. Bei `full_text` ist `purge_after` korrekt gesetzt. Der Purge löscht nur abgelaufene Zeilen, und Artikel sowie Links bleiben erhalten.

## Bewusst nicht in Phase 2

- Business Wire (RSS abgeschaltet). Ersatz wäre GlobeNewswire (403/Bot-Schutz) oder ein bezahlter Feed, siehe Recherche.
- OpenFIGI (braucht einen Key) und ISINs für importierte US-Titel. Der CIK reicht für SEC, für den Rest genügen Name und Ticker.
- Abruf von 8-K-Volltexten, Classifier, MCP: Phase 3.
- Frontend: keine Änderung nötig, weil der Vertrag von `/api/v1/news` gleich bleibt.
