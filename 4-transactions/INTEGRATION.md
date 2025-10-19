# Transaction Service Integration

## Docker & Nginx Setup

The transaction service is now fully integrated with the Omni system.

### Access Points

**Via Nginx (Recommended):**

```
http://localhost/api/transactions/*
```

**Direct Access (for debugging):**

```
http://localhost:8084/api/transactions/*
```

### Service Architecture

```
                          ┌─────────────┐
                          │    Nginx    │
                          │   Port 80   │
                          └──────┬──────┘
                                 │
                    ┌────────────┼────────────┬────────────┐
                    │            │            │            │
              ┌─────▼─────┐ ┌───▼────┐ ┌────▼─────┐ ┌───▼─────────┐
              │   Users   │ │Wallets │ │Notifictn │ │Transactions │
              │  :8080    │ │ :8080  │ │  :8080   │ │    :8083    │
              └─────┬─────┘ └───┬────┘ └────┬─────┘ └──────┬──────┘
                    │           │            │              │
              ┌─────▼─────┐ ┌───▼────┐ ┌────▼─────┐ ┌─────▼──────┐
              │User Redis │ │Wallet  │ │Notif     │ │Transaction │
              │  :6379    │ │Redis   │ │Redis     │ │Redis :6379 │
              │           │ │ :6380  │ │  :6381   │ │            │
              └───────────┘ └────────┘ └──────────┘ └────────────┘
```

### Nginx Routes

```nginx
/api/users           → user-service:8080
/api/wallets         → wallet-service:8080
/api/notifications   → notification-service:8080
/api/transactions    → transaction-service:8083
```

### Docker Services

**transaction-service:**

- Container: `transaction-service`
- External Port: `8084`
- Internal Port: `8083`
- Depends on: Kafka (broker), transaction-redis

**transaction-redis:**

- Container: `transaction-redis`
- External Port: `6382`
- Internal Port: `6379`
- Password protected
- Persistent volume: `transaction_redis_data`

### Environment Variables

```bash
TRANSACTION_SERVICE_PORT=8084
TRANSACTION_SERVICE_INTERNAL_PORT=8083
TRANSACTION_REDIS_PASSWORD=transaction_redis_pass_2024
TRANSACTION_REDIS_PORT=6382
```

### Service Communication

**Transaction Service → User Service:**

- Lookup users by OmniTag: `GET http://user-service:8080/api/users/search/{omniTag}`

**Transaction Service → Wallet Service:**

- Get wallet: `GET http://wallet-service:8080/api/wallets/{walletId}`
- List wallets: `GET http://wallet-service:8080/api/wallets/list/{accountId}`
- Get card: `GET http://wallet-service:8080/api/wallets/cards/{cardId}`

**Transaction Service → Kafka:**

- Publishes events to topics:
  - `transaction-created`
  - `transaction-completed`
  - `transaction-failed`
  - `money-received`
  - `money-sent`
  - `card-purchase`
  - `card-refund`

**Notification Service ← Kafka:**

- Consumes transaction events to send notifications

## Usage Examples

### Transfer Money (via Nginx)

```bash
curl -X POST http://localhost/api/transactions/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "senderWalletId": "wallet-uuid-123",
    "receiverOmniTag": "john",
    "amount": 100.00,
    "description": "Payment for dinner",
    "idempotencyKey": "unique-key-123"
  }'
```

### Card Purchase (via Nginx)

```bash
curl -X POST http://localhost/api/transactions/purchase \
  -H "Content-Type: application/json" \
  -d '{
    "cardId": "card-uuid-456",
    "merchantName": "Amazon",
    "merchantCategory": "retail",
    "amount": 49.99,
    "currency": "USD",
    "description": "Book purchase",
    "idempotencyKey": "unique-key-456"
  }'
```

### Get Transaction History (via Nginx)

```bash
# By account
curl http://localhost/api/transactions/account/account-uuid-123?limit=20&offset=0

# By wallet
curl http://localhost/api/transactions/wallet/wallet-uuid-123?limit=20&offset=0

# With filters
curl "http://localhost/api/transactions/account/account-uuid-123?type=transfer&status=completed&limit=10"
```

### Get Specific Transaction (via Nginx)

```bash
curl http://localhost/api/transactions/transaction-uuid-789
```

## Starting the Service

### Build and Start All Services

```bash
# From project root
docker-compose up --build -d
```

### Start Only Transaction Service

```bash
docker-compose up --build -d transaction-service
```

### View Logs

```bash
# Transaction service logs
docker-compose logs -f transaction-service

# All services logs
docker-compose logs -f
```

### Stop Services

```bash
docker-compose down
```

## Health Check

```bash
curl http://localhost:8084/health
# or via nginx (if nginx is configured with /health route)
```

## Database Mode

The service supports two modes (configured via `MODE` env var):

**Memory Mode (Development):**

```bash
ENVIRONMENT=local
MODE=memcached
```

- No persistence
- Fast startup
- Good for testing

**Redis Mode (Production-like):**

```bash
ENVIRONMENT=local
MODE=redis  # or any value except "memcached"
```

- Persistent storage
- Survives container restarts
- Closer to production

## Troubleshooting

### Service not accessible via nginx

1. Check nginx is running: `docker ps | grep nginx`
2. Check transaction-service is running: `docker ps | grep transaction-service`
3. Verify nginx config: `docker exec nginx cat /etc/nginx/nginx.conf`
4. Check nginx logs: `docker-compose logs nginx`

### Service can't connect to Redis

1. Check Redis is running: `docker ps | grep transaction-redis`
2. Verify Redis password matches: Check `.env` file
3. Check service logs: `docker-compose logs transaction-service`

### Service can't connect to other services

1. Verify all services are on same Docker network
2. Check service URLs in environment variables
3. Test connectivity: `docker exec transaction-service ping user-service`

### Port conflicts

If port 8084 is already in use, update `.env`:

```bash
TRANSACTION_SERVICE_PORT=8085  # or any available port
```

## Next Steps

1. Integrate with fraud detection service
2. Implement atomic balance updates with wallet service
3. Add daily/monthly spending limit tracking
4. Implement transaction reversal functionality
5. Add currency conversion support
