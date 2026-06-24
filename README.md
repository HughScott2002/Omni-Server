# Omni-Server

<p align="center">
  <strong>Experimental microservices backend for digital payments.</strong><br/>
  Users, wallets, notifications, transactions, fraud checks. One gateway, mixed sync + async flows, Docker-first local setup.
</p>

<p align="center">
  <a href="#quick-start"><img src="https://img.shields.io/badge/-Quick_Start-111111?style=for-the-badge" alt="Quick Start" /></a>&nbsp;
  <a href="#architecture"><img src="https://img.shields.io/badge/-Architecture-111111?style=for-the-badge" alt="Architecture" /></a>&nbsp;
  <a href="#services"><img src="https://img.shields.io/badge/-Services-111111?style=for-the-badge" alt="Services" /></a>&nbsp;
  <a href="#status"><img src="https://img.shields.io/badge/-Status-111111?style=for-the-badge" alt="Status" /></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/State-Experimental-bf360c?style=flat-square" alt="Experimental" />
  <img src="https://img.shields.io/badge/Phase-In_Development-6a1b9a?style=flat-square" alt="In Development" />
  <img src="https://img.shields.io/badge/Architecture-Microservices-1565c0?style=flat-square" alt="Microservices" />
  <img src="https://img.shields.io/badge/Go-Services-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go Services" />
  <img src="https://img.shields.io/badge/FastAPI-Notifications-009688?style=flat-square&logo=fastapi&logoColor=white" alt="FastAPI" />
  <img src="https://img.shields.io/badge/Nginx-Gateway-009639?style=flat-square&logo=nginx&logoColor=white" alt="Nginx" />
  <img src="https://img.shields.io/badge/Kafka-Events-231F20?style=flat-square&logo=apachekafka&logoColor=white" alt="Kafka" />
  <img src="https://img.shields.io/badge/Redis-Per_Service-DC382D?style=flat-square&logo=redis&logoColor=white" alt="Redis" />
  <img src="https://img.shields.io/badge/Docker-Local_Dev-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker" />
</p>

> **Status:** Experimental and in development. Core flows run locally today. Testing, CI coverage, horizontal scaling, and some cross-service behavior still need hardening.

## What This Repo Is

Omni-Server is microservices backend for wallet-style product. It is built to show real backend system design, not single CRUD app. Go services handle users, wallets, transactions, and fraud detection. FastAPI service handles notification storage and WebSocket fanout. Nginx fronts public routes, Kafka carries domain events, Redis backs per-service state.

## Problem

Digital payments backend gets hard fast. Auth, wallets, transfers, fraud checks, notifications, session state, event delivery, local orchestration all need to work together. Single service gets messy fast.

## Solution

Omni-Server splits system into focused services behind one gateway:

| # | Capability | What it does |
| --- | --- | --- |
| 1 | **User service** | Registration, login, sessions, contacts, OmniTag search, KYC |
| 2 | **Wallet service** | Wallet creation, wallet lookup, virtual card lifecycle |
| 3 | **Transaction service** | Transfer and purchase orchestration, history, fraud check handoff |
| 4 | **Fraud detection** | Rules-based risk scoring before transaction completion |
| 5 | **Notification service** | Kafka consumers + WebSocket push for account and card events |

## Architecture

```text
Client
  -> Nginx gateway
      -> User Service
      -> Wallet Service
      -> Notification Service
      -> Transaction Service -> Fraud Detection Service

Kafka broker
  -> account-created
  -> contact-request-*
  -> virtual-card-*
  -> transaction-*

Redis
  -> user-service state
  -> wallet-service state
  -> transaction-service state
  -> notification-service state
```

### Key runtime flows

```text
Register user
  -> User Service creates account
  -> Kafka publishes account-created
  -> Wallet Service creates default wallet + card
  -> Notification Service stores + pushes welcome events

Transfer money
  -> Transaction Service validates sender + receiver
  -> calls Wallet Service and User Service
  -> calls Fraud Detection Service
  -> publishes transaction events
```

## Services

| Path | Stack | Role |
| --- | --- | --- |
| `0-nginx/` | Nginx | Public gateway and route fanout |
| `1-users/` | Go | Auth, sessions, profiles, contacts, OmniTag, KYC |
| `2-notification/` | Python / FastAPI | Notification persistence, Kafka consume, WebSocket delivery |
| `3-wallet/` | Go | Wallets, virtual cards, wallet bootstrap on account creation |
| `4-transactions/` | Go | Transfers, purchases, transaction history, fraud handoff |
| `5-fraud-detection/` | Go | Rules-based risk assessment with amount, velocity, pattern checks |

## What Works Today

- user registration, login, refresh, logout, session checks
- OmniTag lookup and contact workflow in user service
- wallet bootstrap from `account-created` event
- virtual card event publishing from wallet service
- transaction service calling user, wallet, and fraud services
- rules-based fraud scoring with amount, velocity, time, and description rules
- notification service consuming Kafka events and pushing WebSocket notifications
- local Docker Compose stack
- local Docker Swarm-style stack for production-like testing

## Quick Start

### Prerequisites

- Docker
- Docker Compose
- ~4 GB RAM for local stack

### Start stack

```bash
make build
```

Direct Docker command if you want it:

```bash
docker compose --env-file .env.example up --build -d
```

### Check stack

```bash
docker compose --env-file .env.example ps
curl http://localhost/api/users/health
curl http://localhost:8083/api/notifications/health
curl http://localhost:8084/health
curl http://localhost:8085/health
```

### Useful commands

| Command | What it does |
| --- | --- |
| `make build` | Build and start full local stack |
| `make logs` | Follow Compose logs |
| `make ps` | List running services |
| `make test` | Run simple smoke checks against running stack |
| `make down` | Stop local stack |
| `make swarm-init` | Init local Docker Swarm |
| `make swarm-deploy` | Deploy local Swarm stack |

### Default local ports

| Service | URL |
| --- | --- |
| Nginx gateway | `http://localhost` |
| User service | `http://localhost:8081` |
| Wallet service | `http://localhost:8082` |
| Notification service | `http://localhost:8083` |
| Transaction service | `http://localhost:8084` |
| Fraud detection service | `http://localhost:8085` |

## Why This Repo Is Interesting

- polyglot backend: Go + FastAPI
- mixed traffic model: request/response + event-driven + WebSocket push
- gateway pattern with clean service boundaries
- explicit transaction risk scoring path
- Docker Compose for dev, Swarm path for production-like deployment
- room to improve and evolve

## Status

### Testing and quality

- automated Go tests exist in `1-users/`
- repo-level smoke checks exist through health endpoints and `make test`
- no automated test suites found yet for `2-notification/`, `3-wallet/`, `4-transactions/`, `5-fraud-detection/`
- current CI workflow is deploy-focused and only builds `1-users/`

### Known gaps

- full multi-service automated test matrix missing
- some transaction ownership edges still need hardening
- notification coverage for transaction events looks incomplete
- fraud detection transaction history is still in-memory, not shared store
- service contracts are still moving

## Project Structure

```text
Omni-Server/
├── 0-nginx/              # gateway config
├── 1-users/              # auth, sessions, contacts, KYC
├── 2-notification/       # notifications, Kafka consumer, WebSocket push
├── 3-wallet/             # wallets and cards
├── 4-transactions/       # transfer and purchase flows
├── 5-fraud-detection/    # rules-based risk scoring
├── docker-swarm/         # Swarm deployment files
├── docker-compose.yaml   # local multi-service stack
├── makefile              # dev + deployment shortcuts
└── CHANGELOG.md
```

## Roadmap

- add automated tests across all services
- add full CI build/test matrix
- finish transaction event coverage in notifications
- harden balance update ownership in transaction flow
- move more deployment and operational knowledge into stable service docs

## Further Reading

- `4-transactions/TRANSACTIONS_API.md`
- `5-fraud-detection/FRAUD_DETECTION_RULES.md`
- `docker-swarm/README.md`

## License

See `LICENSE`.
