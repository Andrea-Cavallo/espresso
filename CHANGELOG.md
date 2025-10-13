# Changelog

Tutte le modifiche importanti a questo progetto saranno documentate in questo file.

Il formato è basato su [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
e questo progetto aderisce al [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Pianificato
- Supporto per HTTP/3
- Plugin system per estensioni custom
- Dashboard web per monitoring
- Generatore automatico di documentazione API

## [1.0.0] - 2025-10-13

### Added
- Client HTTP type-safe con supporto completo ai generics Go 1.21+
- Sistema di retry configurabile con exponential backoff e jitter
- Circuit breaker con rilevamento automatico dei fallimenti
- Cache integrata con TTL configurabile
- Metrics collector per monitoraggio delle performance
- Rate limiting per protezione API
- Sistema di autenticazione (Bearer, Basic, Custom)
- Chain middleware estensibile
- Pattern di orchestrazione avanzati:
  - Chain Pattern per sequenze di chiamate
  - Fan-Out Pattern per esecuzione parallela
  - Saga Pattern per transazioni distribuite
  - Batch Pattern per elaborazione massiva
  - Conditional Pattern per logica condizionale
- Response switching con branching condizionale
- Supporto completo per context propagation
- Request builder fluent API
- Gestione automatica degli errori con callback personalizzabili

### Features per Orchestrazione
- Esecuzione sequenziale e parallela di step
- Context sharing tra step
- Step condizionali con predicati
- Step opzionali e required
- Callback OnSuccess, OnFail, OnComplete, OnError
- Status code specific handlers
- Salvataggio automatico delle response nel context

### Resilience Features
- Retry automatico su errori temporanei
- Circuit breaker states (Closed, Open, Half-Open)
- Timeout configurabili per request
- Backoff strategies (Exponential, Linear, Fixed)
- Jitter per evitare thundering herd
- Status code configurabili per retry

### Developer Experience
- API type-safe con zero type assertions
- Documentazione completa con esempi
- Error messages descrittivi
- Support per Go modules
- Esempi end-to-end completi

## [0.9.0] - 2025-09-15

### Added
- Versione beta con feature core
- Client base con configurazione
- Request builder iniziale
- Retry mechanism base
- Test suite iniziale

### Changed
- Refactoring architettura interna
- Miglioramento gestione errori

### Fixed
- Memory leak in connection pooling
- Race condition nel circuit breaker
- Panic su response nil

## [0.5.0] - 2025-07-20

### Added
- Proof of concept iniziale
- Basic HTTP client wrapper
- Configurazione minimale

### Known Issues
- Circuit breaker non thread-safe
- Cache non supporta TTL
- Mancano test di integrazione

## [0.1.0] - 2025-06-01

### Added
- Setup progetto iniziale
- Struttura base repository
- CI/CD pipeline
- README e documentazione base

---

## Tipi di Cambiamenti

- `Added` per nuove funzionalità
- `Changed` per modifiche a funzionalità esistenti
- `Deprecated` per funzionalità che saranno rimosse
- `Removed` per funzionalità rimosse
- `Fixed` per bug fix
- `Security` per vulnerabilità corrette

## Versioning

Il progetto segue [Semantic Versioning](https://semver.org/):

- **MAJOR** (1.x.x): Modifiche incompatibili con versioni precedenti
- **MINOR** (x.1.x): Nuove funzionalità backward-compatible
- **PATCH** (x.x.1): Bug fix backward-compatible
