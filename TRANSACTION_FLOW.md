# Transaction Flow with Risk Scoring

This document describes the complete flow for processing a transfer transaction with integrated fraud detection.

## Architecture Overview

```
┌─────────────┐      ┌─────────────────┐      ┌──────────────────┐
│   Client    │─────▶│  Transaction    │─────▶│  Fraud Detection │
│             │      │    Service      │      │     Service      │
└─────────────┘      └─────────────────┘      └──────────────────┘
                             │
                             ▼
                     ┌───────────────┐
                     │  Kafka Broker │
                     └───────────────┘
                             │
                     ┌───────┴───────┐
                     ▼               ▼
              ┌──────────┐    ┌─────────────┐
              │  Wallet  │    │Notification │
              │ Service  │    │   Service   │
              └──────────┘    └─────────────┘
```

## Transaction Flow Steps

### 1. Client Initiates Transfer

**Endpoint:** `POST /api/transactions/transfer`

**Request:**
```json
{
  "senderWalletId": "uuid",
  "receiverOmniTag": "john",
  "amount": 100.50,
  "description": "Payment for dinner",
  "idempotencyKey": "unique-key-123"
}
```

### 2. Transaction Service: Validation Phase

The transaction service performs the following validations:

1. **Input Validation**
   - Validate amount is positive
   - Validate idempotency key is provided
   - Validate sender wallet ID exists
   - Validate receiver OmniTag is provided

2. **Idempotency Check**
   - Check if this idempotency key was already processed
   - If yes, return cached response (prevents duplicate transactions)

3. **Wallet Validation**
   - Fetch sender wallet from wallet service
   - Check wallet is active
   - Verify sufficient balance

4. **Receiver Lookup**
   - Look up receiver by OmniTag from user service
   - Prevent self-transfers
   - Get receiver's default wallet
   - Verify currency match

### 3. Transaction Creation

The service creates a transaction record with status `pending`:

```go
transaction := &Transaction{
    ID:                  "generated-uuid",
    Reference:           "TXN-20251019-abc123",
    SenderAccountID:     "sender-account-id",
    ReceiverAccountID:   "receiver-account-id",
    SenderWalletID:      "sender-wallet-id",
    ReceiverWalletID:    "receiver-wallet-id",
    Amount:              100.50,
    Currency:            "USD",
    TransactionType:     "transfer",
    TransactionCategory: "debit",
    Status:              "pending",
    Description:         "Payment for dinner",
    BalanceBefore:       1000.00,
    BalanceAfter:        899.50,
    CreatedAt:           now,
}
```

**Kafka Event Published:** `transaction-created`

### 4. Risk Assessment Phase

The transaction service calls the fraud detection service:

**Request to Fraud Detection:**
```json
{
  "transactionId": "generated-uuid",
  "senderAccountId": "sender-account-id",
  "receiverAccountId": "receiver-account-id",
  "amount": 100.50,
  "currency": "USD",
  "transactionType": "transfer",
  "description": "Payment for dinner",
  "metadata": {
    "receiverOmniTag": "john",
    "idempotencyKey": "unique-key-123"
  }
}
```

**Response from Fraud Detection:**
```json
{
  "transactionId": "generated-uuid",
  "riskScore": 5.0,
  "riskLevel": "low",
  "decision": "approve",
  "reasons": ["Transaction within normal parameters"],
  "assessedAt": "2025-10-19T12:00:00Z"
}
```

### 5. Decision Point

Based on the fraud detection response:

#### 5a. If Transaction is APPROVED (decision = "approve")

1. Add risk assessment metadata to transaction
2. Update balances (debit sender, credit receiver)
3. Mark transaction as `completed`
4. Publish Kafka events:
   - `transaction-completed`
   - `money-sent` (to sender's account)
   - `money-received` (to receiver's account)
5. Store idempotency response
6. Return success response to client

**Success Response:**
```json
{
  "status": "success",
  "message": "Transfer completed successfully",
  "transactionId": "generated-uuid",
  "reference": "TXN-20251019-abc123",
  "senderBalance": 899.50,
  "receiverBalance": 1100.50,
  "transaction": {
    "id": "generated-uuid",
    "status": "completed",
    "metadata": {
      "riskScore": 5.0,
      "riskLevel": "low",
      "riskDecision": "approve"
    }
  }
}
```

#### 5b. If Transaction is DECLINED (decision = "decline" or "review")

1. Add risk assessment metadata to transaction
2. Mark transaction as `failed`
3. Publish Kafka event: `transaction-failed`
4. Store idempotency response
5. Return failure response to client

**Failure Response:**
```json
{
  "status": "failed",
  "message": "Transaction declined due to risk assessment: high"
}
```

### 6. Downstream Processing

#### Notification Service (Kafka Consumer)

Listens for transaction events and sends notifications:

- `transaction-completed` → Send success notifications to sender and receiver
- `transaction-failed` → Send failure notification to sender
- `money-sent` → Update sender's notification feed
- `money-received` → Update receiver's notification feed

#### Wallet Service (Future Enhancement)

In production, the wallet service should:
- Listen for `transaction-completed` events
- Atomically update wallet balances
- Publish balance update confirmations

## Risk Scoring Details

### Current Implementation (v1)

The fraud detection service currently **approves all transactions** with:
- Risk Score: 5.0 (out of 100)
- Risk Level: "low"
- Decision: "approve"

This is a placeholder implementation for initial testing.

### Future Enhancements

The fraud detection service can be enhanced with:

1. **Transaction Velocity Checks**
   - Track number of transactions per hour/day
   - Flag unusual spikes in activity

2. **Amount Analysis**
   - Compare against user's historical transaction amounts
   - Flag amounts significantly above normal

3. **Geographic Patterns**
   - Check for impossible travel (e.g., transactions from different countries minutes apart)
   - Flag transactions from high-risk regions

4. **Machine Learning Models**
   - Train models on historical fraud data
   - Real-time scoring based on multiple features

5. **AML Screening**
   - Check against sanctions lists
   - Monitor for suspicious patterns

6. **Blacklist/Whitelist**
   - Maintain lists of known bad actors
   - Whitelist trusted users for faster processing

## API Endpoints

### Transaction Service

- `POST /api/transactions/transfer` - Transfer money between wallets
- `POST /api/transactions/purchase` - Card purchase transaction
- `GET /api/transactions/account/{accountId}` - Get transaction history
- `GET /api/transactions/wallet/{walletId}` - Get wallet transactions
- `GET /api/transactions/{transactionId}` - Get transaction details

### Fraud Detection Service

- `POST /api/fraud-detection/assess` - Assess transaction risk
- `GET /health` - Health check

## Service URLs

### Via Nginx (Production)

- Transactions: `http://localhost/api/transactions/*`
- Fraud Detection: `http://localhost/api/fraud-detection/*`

### Direct Access (Development/Debugging)

- Transactions: `http://localhost:8084/api/transactions/*`
- Fraud Detection: `http://localhost:8085/api/fraud-detection/*`

## Environment Variables

```bash
# Transaction Service
TRANSACTION_SERVICE_PORT=8084
TRANSACTION_SERVICE_INTERNAL_PORT=8083
TRANSACTION_REDIS_PASSWORD=transaction_redis_pass_2024
TRANSACTION_REDIS_PORT=6382
USER_SERVICE_URL=http://user-service:8080
WALLET_SERVICE_URL=http://wallet-service:8080
FRAUD_DETECTION_URL=http://fraud-detection-service:8085

# Fraud Detection Service
FRAUD_DETECTION_SERVICE_PORT=8085
FRAUD_DETECTION_SERVICE_INTERNAL_PORT=8085
```

## Testing the Flow

### 1. Start All Services

```bash
docker-compose up --build -d
```

### 2. Create Test Users and Wallets

You'll need to create test users and wallets first using the user and wallet services.

### 3. Execute a Transfer

```bash
curl -X POST http://localhost/api/transactions/transfer \
  -H "Content-Type: application/json" \
  -d '{
    "senderWalletId": "your-wallet-id",
    "receiverOmniTag": "receiver-tag",
    "amount": 50.00,
    "description": "Test transfer",
    "idempotencyKey": "test-key-1"
  }'
```

### 4. Check Logs

```bash
# Transaction service logs
docker logs -f transaction-service

# Fraud detection service logs
docker logs -f fraud-detection-service

# Expected log entries:
# Transaction Service: "Risk assessed for transaction..."
# Fraud Detection: "Risk assessment for transaction..."
```

### 5. Verify Transaction

```bash
curl http://localhost/api/transactions/{transactionId}
```

Look for risk metadata in the response:
```json
{
  "metadata": {
    "riskScore": 5.0,
    "riskLevel": "low",
    "riskDecision": "approve"
  }
}
```

## Error Handling

### Fraud Detection Service Unavailable

If the fraud detection service is unavailable:
- Error is logged: "Failed to assess transaction risk"
- Transaction continues (graceful degradation)
- In production, you may want to:
  - Fail the transaction
  - Queue for manual review
  - Use a fallback risk scoring mechanism

### Timeout Handling

The fraud detection call has a 5-second timeout. If exceeded:
- Request fails with timeout error
- Transaction processing handles the error gracefully

## Idempotency

Idempotency is critical for preventing duplicate transactions:

1. Client provides `idempotencyKey` with each request
2. Before processing, check if key was already used
3. If yes, return cached response (same transaction won't process twice)
4. If no, process transaction and cache response

This ensures:
- Network retries don't create duplicate transactions
- Client can safely retry failed requests
- Same result returned for same idempotency key

## Security Considerations

1. **Authentication/Authorization** (To be implemented)
   - Verify sender owns the wallet
   - Validate user permissions

2. **Rate Limiting** (To be implemented)
   - Prevent abuse
   - Limit requests per user/IP

3. **Data Encryption**
   - Use HTTPS in production
   - Encrypt sensitive data at rest

4. **Audit Logging**
   - All transactions logged
   - Immutable audit trail

## Monitoring and Observability

### Key Metrics to Track

1. **Transaction Metrics**
   - Total transactions per minute
   - Success rate
   - Failure rate by reason
   - Average transaction amount

2. **Fraud Detection Metrics**
   - Risk score distribution
   - Approval/decline rates
   - False positive rate (if known)
   - Service response time

3. **System Metrics**
   - Service uptime
   - Response times
   - Error rates
   - Kafka lag

### Logging

Both services log key events:
- Transaction creation
- Risk assessment results
- Transaction completion/failure
- External service calls

## Troubleshooting

### Transaction fails with "Sender wallet not found"

- Ensure wallet exists in wallet service
- Check wallet ID is correct
- Verify wallet service is running

### Transaction fails with "Receiver not found"

- Ensure receiver user exists
- Check OmniTag is correct (max 5 characters)
- Verify user service is running

### Risk assessment always fails

- Check fraud detection service is running: `docker ps | grep fraud-detection`
- Verify network connectivity: `docker exec transaction-service ping fraud-detection-service`
- Check fraud detection logs: `docker logs fraud-detection-service`

### Transactions stuck in "pending" state

- Check if fraud detection service responded
- Look for errors in transaction service logs
- Verify transaction was updated after risk assessment
