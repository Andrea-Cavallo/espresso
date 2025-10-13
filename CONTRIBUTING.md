# Contributing to Espresso

Grazie per il tuo interesse nel contribuire a Espresso! Ogni contributo è benvenuto.

## Come Contribuire

### Segnalare Bug

Apri una issue su GitHub includendo:

- Descrizione chiara del problema
- Steps per riprodurre il bug
- Comportamento atteso vs comportamento attuale
- Versione Go utilizzata
- Snippet di codice minimale

### Proporre Nuove Features

Prima di iniziare a sviluppare:

1. Apri una issue per discutere la feature
2. Aspetta feedback dai maintainer
3. Procedi con l'implementazione se approvata

### Pull Request

1. **Fork** il repository
2. **Crea un branch** dalla `main`:
   ```bash
   git checkout -b feature/nome-feature
   ```
3. **Scrivi codice** seguendo le convenzioni
4. **Aggiungi test** per la nuova funzionalità
5. **Assicurati** che tutti i test passino:
   ```bash
   go test ./...
   ```
6. **Committa** seguendo Conventional Commits:
   ```bash
   git commit -m "feat: aggiungi supporto per HTTP/3"
   ```
7. **Pusha** sul tuo fork
8. **Apri una PR** verso `main`

## Standard di Codice

### Principi SOLID

Segui i principi SOLID in ogni componente:

- **S**ingle Responsibility
- **O**pen/Closed
- **L**iskov Substitution
- **I**nterface Segregation
- **D**ependency Inversion

### Best Practices

- Usa guard clauses invece di if-else annidati
- Mantieni funzioni sotto 50 righe
- Nomi di variabili autoesplicativi
- Zero duplicazione (DRY)
- Gestisci sempre gli errori esplicitamente
- Valida gli input all'inizio delle funzioni
- Preferisci immutabilità quando possibile
- Massimo 3-4 parametri per funzione

### Esempio Codice

```go
// ✓ Good
func ProcessOrder(ctx context.Context, orderID string) error {
    if orderID == "" {
        return ErrInvalidOrderID
    }

    order, err := fetchOrder(ctx, orderID)
    if err != nil {
        return fmt.Errorf("fetch order: %w", err)
    }

    return processOrderLogic(order)
}

// ✗ Bad
func ProcessOrder(ctx context.Context, orderID string) error {
    if orderID != "" {
        order, err := fetchOrder(ctx, orderID)
        if err == nil {
            return processOrderLogic(order)
        } else {
            return err
        }
    }
    return ErrInvalidOrderID
}
```

## Testing

### Requisiti

- Ogni nuova feature deve avere test
- Coverage minimo: 80%
- Test unitari e di integrazione

### Eseguire Test

```bash
# Tutti i test
go test ./...

# Con coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Race detector
go test -race ./...

# Benchmark
go test -bench=. -benchmem
```

### Esempio Test

```go
func TestClient_Get(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        want    int
        wantErr bool
    }{
        {
            name:    "successful request",
            path:    "/users/1",
            want:    200,
            wantErr: false,
        },
        {
            name:    "not found",
            path:    "/users/999",
            want:    404,
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Commit Convention

Usa [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` nuova feature
- `fix:` bug fix
- `docs:` modifiche alla documentazione
- `style:` formattazione codice
- `refactor:` refactoring senza cambio funzionalità
- `test:` aggiunta o modifica test
- `chore:` modifiche alla build, dipendenze

Esempi:
```
feat: aggiungi supporto per OAuth2
fix: risolvi race condition nel circuit breaker
docs: aggiorna esempi nel README
refactor: semplifica retry logic
test: aggiungi test per batch pattern
```

## Code Review

Aspettati feedback costruttivo. I maintainer valuteranno:

- Qualità del codice
- Test coverage
- Documentazione
- Performance
- Compatibilità backward

## Domande?

Apri una issue o contatta [@Andrea-Cavallo](https://github.com/Andrea-Cavallo).

Grazie per contribuire a Espresso! ☕