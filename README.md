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

Aprire `http://localhost:5173`, premere `Sincronizza catalogo`, cercare una stampa e avviare una scansione. Il primo import del catalogo può richiedere tempo. Job, risultati e immagini restano nel volume Docker `app-data`.

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

Il container usa GoCV/OpenCV con feature ORB, BFMatcher e omografia RANSAC per produrre coordinate e punteggio senza inviare immagini a servizi remoti. Ollama interviene soltanto sui candidati ambigui e il job continua in modalità degradata se non è disponibile.

Il matcher resta isolato dietro `ImageMatcher`. I test host usano un fallback Go puro per non richiedere OpenCV installato localmente; l'immagine Docker abilita l'adapter GoCV tramite build tag.

## Verifica

```sh
cd backend
GOCACHE=/private/tmp/pokemon-binder-go-build go test ./...

cd ../frontend
npm install
npm run build
```
