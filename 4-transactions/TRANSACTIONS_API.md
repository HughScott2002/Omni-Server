# Transaction Service API

## POST /api/transactions/transfer

Transfer money between wallets using OmniTag.

**Input:**

```json
{
  "senderWalletId": "uuid",
  "receiverOmniTag": "string (max 5 chars)",
  "amount": 100.5,
  "description": "Payment for dinner",
  "idempotencyKey": "unique-key-123"
}
```

**Output (Success):**

```json
{
  "status": "success",
  "message": "Transfer completed successfully",
  "transactionId": "uuid",
  "reference": "TXN-20251018-abc12345",
  "senderBalance": 900.0,
  "receiverBalance": 1100.5,
  "transaction": {
    "id": "uuid",
    "reference": "TXN-20251018-abc12345",
    "senderAccountId": "uuid",
    "receiverAccountId": "uuid",
    "senderWalletId": "uuid",
    "receiverWalletId": "uuid",
    "amount": 100.5,
    "currency": "USD",
    "transactionType": "transfer",
    "transactionCategory": "debit",
    "status": "completed",
    "description": "Payment for dinner",
    "balanceBefore": 1000.0,
    "balanceAfter": 900.0,
    "createdAt": "2025-10-18T12:00:00Z",
    "completedAt": "2025-10-18T12:00:01Z",
    "metadata": {
      "receiverOmniTag": "john",
      "idempotencyKey": "unique-key-123"
    }
  }
}
```

**Output (Failure):**

```json
{
  "status": "failed",
  "message": "Insufficient balance"
}
```

**Status Codes:**

- 200: Success
- 400: Bad request (validation error, insufficient balance, etc.)
- 404: Wallet or receiver not found
- 500: Internal server error

---

## POST /api/transactions/purchase

Simulate a card purchase transaction.

**Input:**

```json
{
  "cardId": "uuid",
  "merchantName": "Amazon",
  "merchantCategory": "retail",
  "amount": 49.99,
  "currency": "USD",
  "description": "Book purchase",
  "idempotencyKey": "unique-key-456"
}
```

**Output (Success):**

```json
{
  "status": "success",
  "message": "Purchase completed successfully",
  "transactionId": "uuid",
  "reference": "TXN-20251018-def67890",
  "cardBalance": 450.01,
  "walletBalance": 950.01,
  "transaction": {
    "id": "uuid",
    "reference": "TXN-20251018-def67890",
    "senderAccountId": "uuid",
    "senderWalletId": "uuid",
    "cardId": "uuid",
    "amount": 49.99,
    "currency": "USD",
    "transactionType": "card_purchase",
    "transactionCategory": "debit",
    "status": "completed",
    "description": "Book purchase",
    "balanceBefore": 500.0,
    "balanceAfter": 450.01,
    "metadata": {
      "merchantName": "Amazon",
      "merchantCategory": "retail",
      "idempotencyKey": "unique-key-456",
      "cardType": "debit",
      "cardBrand": "visa"
    },
    "createdAt": "2025-10-18T12:00:00Z",
    "completedAt": "2025-10-18T12:00:01Z"
  }
}
```

**Output (Failure):**

```json
{
  "status": "failed",
  "message": "Insufficient card balance"
}
```

**Status Codes:**

- 200: Success
- 400: Bad request (validation error, insufficient balance, card inactive, etc.)
- 404: Card not found
- 500: Internal server error

---

## GET /api/transactions/account/{accountId}

Get transaction history for an account.

**Query Parameters:**

- `limit` (optional): Number of transactions to return (default: 20)
- `offset` (optional): Offset for pagination (default: 0)
- `type` (optional): Filter by transaction type (deposit, withdrawal, transfer, card_purchase, etc.)
- `category` (optional): Filter by category (credit, debit)
- `status` (optional): Filter by status (pending, completed, failed, reversed, cancelled)

**Output:**

```json
[
  {
    "id": "uuid",
    "reference": "TXN-20251018-abc12345",
    "senderAccountId": "uuid",
    "receiverAccountId": "uuid",
    "senderWalletId": "uuid",
    "receiverWalletId": "uuid",
    "amount": 100.5,
    "currency": "USD",
    "transactionType": "transfer",
    "transactionCategory": "debit",
    "status": "completed",
    "description": "Payment for dinner",
    "balanceBefore": 1000.0,
    "balanceAfter": 900.0,
    "createdAt": "2025-10-18T12:00:00Z",
    "completedAt": "2025-10-18T12:00:01Z"
  }
]
```

**Status Codes:**

- 200: Success
- 400: Bad request (invalid account ID)
- 500: Internal server error

---

## GET /api/transactions/wallet/{walletId}

Get transaction history for a specific wallet.

**Query Parameters:**

- Same as account endpoint

**Output:**

- Same format as account endpoint

**Status Codes:**

- 200: Success
- 400: Bad request (invalid wallet ID)
- 500: Internal server error

---

## GET /api/transactions/{transactionId}

Get details of a specific transaction.

**Output:**

```json
{
  "id": "uuid",
  "reference": "TXN-20251018-abc12345",
  "senderAccountId": "uuid",
  "receiverAccountId": "uuid",
  "senderWalletId": "uuid",
  "receiverWalletId": "uuid",
  "amount": 100.5,
  "currency": "USD",
  "transactionType": "transfer",
  "transactionCategory": "debit",
  "status": "completed",
  "description": "Payment for dinner",
  "balanceBefore": 1000.0,
  "balanceAfter": 900.0,
  "metadata": {
    "receiverOmniTag": "john"
  },
  "createdAt": "2025-10-18T12:00:00Z",
  "completedAt": "2025-10-18T12:00:01Z",
  "updatedAt": "2025-10-18T12:00:01Z"
}
```

**Status Codes:**

- 200: Success
- 400: Bad request (invalid transaction ID)
- 404: Transaction not found
- 500: Internal server error

---

## Kafka Events Published

### transaction-created

Published when a new transaction is created.

### transaction-completed

Published when a transaction completes successfully.

### transaction-failed

Published when a transaction fails.

### money-received

Published to the receiver's account when they receive money.

### money-sent

Published to the sender's account when they send money.

### card-purchase

Published when a card purchase is made.

### card-refund

Published when a card purchase is refunded.

---

## Transaction Types

- `deposit`: Money added to wallet
- `withdrawal`: Money removed from wallet
- `transfer`: Money transferred between wallets
- `card_purchase`: Purchase made with virtual card
- `card_refund`: Refund of a card purchase
- `reversal`: Transaction reversal
- `fee_charged`: Fee charged to account
- `interest_credited`: Interest credited to account

## Transaction Statuses

- `pending`: Transaction is being processed
- `completed`: Transaction completed successfully
- `failed`: Transaction failed
- `reversed`: Transaction was reversed
- `cancelled`: Transaction was cancelled

## Transaction Categories

- `credit`: Money coming into account/wallet
- `debit`: Money going out of account/wallet
