# Omni-Server

**Microservices backend for the Omni digital-payments demo.** Go services for users, wallets, transactions and fraud scoring, a FastAPI notification service with WebSocket push, Kafka for domain events, one nginx gateway in front. Pairs with [Omni-UI](https://github.com/HughScott2002/Omni-UI).

<p>
  <img src="https://img.shields.io/badge/State-Experimental-bf360c?style=flat-square" alt="Experimental" />
  <img src="https://img.shields.io/badge/Go-Services-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/FastAPI-Notifications-009688?style=flat-square&logo=fastapi&logoColor=white" alt="FastAPI" />
  <img src="https://img.shields.io/badge/Kafka-Events-231F20?style=flat-square&logo=apachekafka&logoColor=white" alt="Kafka" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker" />
</p>

## Quick start

```bash
make build     # build + start the full stack (uses .env.example)
make seed      # demo user + 6 months of transaction history
```

Log in from Omni-UI with **demo@omni.dev / DemoPass123!**

> Storage is **in-memory** (`MODE=memcached`) — data resets on every `make down`/`restart`. Re-run `make seed` afterward. Redis-backed `MODE=db` exists but isn't finished yet (#10).

## Architecture

```text
Omni-UI ──> nginx (:80) ──┬─> /api/users          user-service        :8081  Go
                          ├─> /api/wallets        wallet-service      :8082  Go
                          ├─> /api/notifications  notification-service:8083  FastAPI + WS
                          ├─> /api/transactions   transaction-service :8084  Go
                          └─> /api/fraud-detection fraud-detection    :8085  Go

Kafka: account-created · kyc-approved · contact-* · virtual-card-* · account-deletion
       (wallet bootstrap + activation, notifications fan out over WebSocket)
```

**Register** → user-service emits `account-created` → wallet-service creates a wallet + virtual card (inactive until verified). **Verify** → consent-gated KYC submit approves the account → `kyc-approved` activates wallets and cards. **Transfer** → transaction-service checks sender/receiver, calls wallet + fraud scoring, records history.

## Commands

| Command | What it does |
| --- | --- |
| `make build` | Build and start the stack |
| `make seed` | Seed demo users, wallets, and history |
| `make down` / `make restart` | Stop / rebuild (remember to reseed) |
| `make logs` / `make ps` | Follow logs / list services |
| `make test` | Smoke-check the running stack |
| `make swarm-*` | Local Docker Swarm variant |

## Status

Works today: auth + sessions, contacts/OmniTag, consent-gated KYC with wallet activation, virtual cards, transfers + card purchases with rules-based fraud scoring, transaction history, real-time notifications, dev seeding endpoints (local-only).

Known gaps (tracked in issues): persistent storage (#10), auth enforcement across services (#12, #17), real verification pipeline (#16), health endpoints + chaos targets (#11), tests + CI beyond user-service (#6).

## Docs

[`4-transactions/TRANSACTIONS_API.md`](4-transactions/TRANSACTIONS_API.md) · [`3-wallet/WALLET_API.md`](3-wallet/WALLET_API.md) · [`5-fraud-detection/FRAUD_DETECTION_RULES.md`](5-fraud-detection/FRAUD_DETECTION_RULES.md) · [`docker-swarm/README.md`](docker-swarm/README.md)

## License

See [`LICENSE`](LICENSE).
