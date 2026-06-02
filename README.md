# Pokémon Binder Finder

MVP locale per trovare una stampa Pokémon specifica nelle foto di binder e lotti pubblicati su eBay. Il frontend consente di scegliere una carta dal catalogo, avviare una ricerca live e ispezionare i match con evidenza visiva.

## Prerequisiti

- Docker Desktop
- account eBay Developer con accesso production alla Browse API
- Ollama sull'host con un modello vision:

```sh
ollama pull qwen2.5vl:7b
```

## Avvio

```sh
cp .env.example .env
# valorizzare EBAY_CLIENT_ID e EBAY_CLIENT_SECRET
docker compose up --build
```

Aprire `http://localhost:5173`, premere `Sincronizza catalogo`, cercare una stampa e avviare una scansione. Il primo import del catalogo può richiedere tempo. I job e i risultati restano in `data/app.db`; le immagini scaricate sono in `data/images`.

Marketplace aggiuntivi richiedono soltanto una modifica alla configurazione:

```env
EBAY_MARKETPLACE_IDS=EBAY_IT,EBAY_DE,EBAY_GB
```

## Architettura

Il backend segue un'impostazione esagonale pragmatica:

```text
backend/internal/
  domain/            entità e regole
  application/       porte e casi d'uso
  adapters/in/       API HTTP
  adapters/out/      catalogo, annunci, storage e vision
  platform/          configurazione
```

eBay è un adapter di `ListingSource`, Pokémon TCG API è un adapter di `CardCatalog`, SQLite è un adapter di `Store` e Ollama è un adapter opzionale di `VisionVerifier`.

## Matcher Locale

Il matcher iniziale è implementato interamente in Go e non invia immagini a servizi remoti. Esegue una ricerca multi-scala locale e produce coordinate e punteggio. Ollama interviene soltanto sui candidati ambigui e il job continua in modalità degradata se non è disponibile.

Il matcher è volutamente isolato dietro `ImageMatcher`: prima di usare il prodotto per scansioni ad alto volume va sostituito o affiancato con l'adapter GoCV/OpenCV pianificato, usando feature locali e omografia per gestire prospettiva, riflessi e carte parzialmente coperte. Il matcher corrente rende eseguibile il flusso completo, ma non offre ancora quella robustezza.

## Verifica

```sh
cd backend
GOCACHE=/private/tmp/pokemon-binder-go-build go test ./...

cd ../frontend
npm install
npm run build
```

