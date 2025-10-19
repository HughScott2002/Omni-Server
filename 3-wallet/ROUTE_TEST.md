# Route Testing

Test these URLs after rebuild:

```bash
# Get card by ID
curl http://localhost:8080/api/wallets/cards/81292a64-0c07-4259-bd82-3123d7cf6ecc

# Get cards by account
curl http://localhost:8080/api/wallets/cards/account/e52f5d27-d12f-4be2-bbd8-8c473095a0b8

# Get wallet
curl http://localhost:8080/api/wallets/7e22f63e-3df8-4040-a506-e56b8b2bc777
```

Check the logs for the debug output showing what cardID is being captured.
