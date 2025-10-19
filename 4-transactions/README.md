# Transaction Service

Go-based microservice for handling financial transactions in the Omni banking system.

## Features

- **Wallet-to-Wallet Transfers**: Transfer money between wallets using OmniTag
- **Card Purchases**: Simulate card purchase transactions
- **Transaction History**: Query transaction history by account or wallet
- **Idempotency**: Prevents duplicate transactions using idempotency keys
- **Event-Driven**: Publishes Kafka events for all transaction activities
- **Redis Storage**: Fast transaction storage and retrieval

## Architecture

### Database

- **Redis**: Primary storage for transactions (fast reads/writes)
- **Memory**: In-memory storage for local development

### External Services

- **User Service**: Lookup users by OmniTag
- **Wallet Service**: Fetch wallet and card information

### Events Published (Kafka)

- `transaction-created`: New transaction created
- `transaction-completed`: Transaction completed successfully
- `transaction-failed`: Transaction failed
- `money-received`: Money received notification
- `money-sent`: Money sent notification
- `card-purchase`: Card purchase notification
- `card-refund`: Card refund notification

## API Endpoints

### POST /api/transactions/transfer

Transfer money between wallets using OmniTag.

### POST /api/transactions/purchase

Simulate a card purchase transaction.

### GET /api/transactions/account/{accountId}

Get transaction history for an account (supports filtering).

### GET /api/transactions/wallet/{walletId}

Get transaction history for a wallet (supports filtering).

### GET /api/transactions/{transactionId}

Get details of a specific transaction.

See [TRANSACTIONS_API.md](./TRANSACTIONS_API.md) for detailed API documentation.

## Environment Variables

- `ENVIRONMENT`: Environment (local, prod)
- `MODE`: Database mode (memcached, redis)
- `REDIS_HOST`: Redis host (default: localhost)
- `REDIS_PORT`: Redis port (default: 6379)
- `REDIS_PASSWORD`: Redis password (if required)
- `USER_SERVICE_URL`: User service URL (default: http://users:8080)
- `WALLET_SERVICE_URL`: Wallet service URL (default: http://wallets:8082)
- `PORT`: Service port (default: 8083)

## Running Locally

```bash
# Install dependencies
go mod download

# Run the service
go run src/main.go
```

## Building

```bash
# Build binary
go build -o transaction-service src/main.go

# Run binary
./transaction-service
```

## Docker Build

```bash
# Build Docker image
docker build -t transaction-service .

# Run container
docker run -p 8083:8083 \
  -e ENVIRONMENT=local \
  -e MODE=redis \
  -e REDIS_HOST=redis \
  transaction-service
```

## Transaction Flow

### Transfer Flow

1. Validate request (amount, idempotency key, wallet ID, OmniTag)
2. Fetch sender wallet from wallet service
3. Check idempotency (return cached response if duplicate)
4. Validate sender wallet status and balance
5. Lookup receiver by OmniTag from user service
6. Fetch receiver's default wallet
7. Validate currency match
8. Create transaction in database
9. Publish `transaction-created` event
10. Update balances (TODO: integrate with wallet service)
11. Mark transaction as completed
12. Publish `transaction-completed`, `money-sent`, `money-received` events
13. Return response

### Card Purchase Flow

1. Validate request (amount, card ID, merchant info)
2. Fetch card from wallet service
3. Fetch wallet from wallet service
4. Check idempotency (return cached response if duplicate)
5. Validate card and wallet status
6. Validate currency and balance
7. Create transaction in database
8. Publish `transaction-created` event
9. Update balances (TODO: integrate with wallet service)
10. Mark transaction as completed
11. Publish `transaction-completed` and `card-purchase` events
12. Return response

## TODO

- [ ] Integrate with wallet service for atomic balance updates
- [ ] Implement daily/monthly spending limit tracking
- [ ] Add transaction reversal functionality
- [ ] Implement currency conversion for cross-currency transfers
- [ ] Add fraud detection integration
- [ ] Implement transaction receipts/statements
- [ ] Add webhook notifications for transaction events
- [ ] Implement batch transaction processing
- [ ] Add transaction analytics and reporting
- [ ] Implement scheduled/recurring transactions

## Integration with Fraud Detection

The transaction service is designed to integrate with the fraud detection service (8-fraud-detection) in the future. The current implementation publishes all necessary events that the fraud detection service can consume to:

- Analyze transaction patterns
- Detect suspicious activity
- Flag high-risk transactions
- Block fraudulent transactions in real-time

## Notes

- All monetary values are stored as float64 (should be changed to decimal for production)
- Idempotency keys expire after 24 hours
- Transaction references follow format: `TXN-YYYYMMDD-{8-char-uuid}`
- Balance updates are currently not atomic (TODO: implement proper wallet service integration)
