# Pokémon Binder Finder

MVP locale per trovare una stampa Pokémon specifica nelle foto di binder e lotti pubblicati su eBay. Il frontend consente di scegliere una carta dal catalogo, avviare una ricerca live e ispezionare i match con evidenza visiva.

## Prerequisiti

- Docker Desktop
- account eBay Developer con accesso production alla Browse API
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

Il backend usa layer Go piatti sotto `backend`:

```text
backend/
  cmd/server/        entrypoint dell'eseguibile
  web/               router e handler HTTP
  service/           orchestrazione dei job di ricerca
  model/             modelli di dominio e costanti
  repository/        SQLite e cache immagini locale
  ebay/              OAuth e Browse API eBay
  pokemontcg/        client e sync Pokemon TCG API
  vision/            matcher GoCV
  config/            lettura env e default runtime
```

`web` dipende da `service`, `pokemontcg` e `model`; `service` coordina repository, eBay, Pokemon TCG, cache immagini e vision.

## Matcher Locale

Il container usa GoCV/OpenCV con feature ORB, BFMatcher e omografia RANSAC per produrre coordinate e punteggio senza inviare immagini a servizi remoti.

OpenCV e CGO sono requisiti del backend anche durante lo sviluppo locale; l'immagine Docker installa le librerie necessarie.

## Verifica

```sh
docker build -t pokemon-binder-finder-backend ./backend

cd frontend
npm install
npm run build
```
