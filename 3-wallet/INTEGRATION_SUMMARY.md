# Wallet & Virtual Card Integration Summary

## Overview

The wallet and virtual card systems are fully integrated with automatic card creation and event-driven notifications.

## Data Flow

### 1. Account Creation → Wallet Creation → Card Creation

```
User Service (account-created event)
    ↓
Wallet Service Consumer (account_created_events.go)
    ↓
Creates Wallet + Default Debit Card
    ↓
Publishes virtual-card-created event
    ↓
Notification Service
    ↓
Sends notifications to user via WebSocket
```

### 2. Card Operations → Notifications

All card operations automatically trigger notifications:

- **Card Created**: Notification sent with last 4 digits
- **Card Blocked**: Notification sent with block reason
- **Card Topped Up**: Notification sent with amount and new balance
- **Physical Card Requested**: Notification sent with delivery details
- **Card Deleted**: Notification sent with last 4 digits

## Data Model Relationships

```
Account (User Service)
    ↓
Wallet (Wallet Service)
    - walletId
    - accountId
    - currency
    - balance
    - dailyLimit
    - monthlyLimit
    ↓
Virtual Card (Wallet Service)
    - id
    - walletId  ← Links to wallet
    - cardType (debit/credit)
    - currency (inherits from wallet)
    - dailyLimit (inherits from wallet)
    - monthlyLimit (inherits from wallet)
    - availableBalance
    - cardNumber (masked in responses)
    - cvvHash (never exposed)
```

## Default Card Creation

When a wallet is created, a default debit card is automatically created with:

- **Card Type**: Debit
- **Card Brand**: Visa
- **Currency**: Inherited from wallet
- **Limits**: Same as wallet (daily: 5000, monthly: 20000)
- **Status**: Active (if KYC approved) or Pending (if KYC pending)
- **Name on Card**: "Card Holder" (user should update)
- **Auto-generated**: Card number (16 digits), CVV (hashed), Expiry date (3 years)

## Event System

### Published Events

**From Wallet Service:**

- `virtual-card-created`
- `virtual-card-blocked`
- `virtual-card-topped-up`
- `physical-card-requested`
- `virtual-card-deleted`

**Consumed by Notification Service:**
All card events are consumed and converted to user notifications.

## API Endpoints

### Wallet Endpoints

- `GET /api/wallets/{walletId}` - Get wallet
- `GET /api/wallets/list/{accountId}` - Get all wallets

### Card Endpoints

All card endpoints are under `/api/wallets/cards` (nginx routes `/api/wallets` to this service).
All card endpoints use `walletId` (not `bankAccountId`) to avoid confusion:

- `POST /api/wallets/cards` - Create new card
- `GET /api/wallets/cards/{cardid}` - Get card details
- `GET /api/wallets/cards/account/{accountid}` - Get all cards for account
- `PUT /api/wallets/cards/{cardid}` - Update card limits/status
- `DELETE /api/wallets/cards/{cardid}` - Delete card
- `POST /api/wallets/cards/{cardid}/block` - Block card
- `POST /api/wallets/cards/{cardid}/topup` - Top up card balance
- `POST /api/wallets/cards/{cardid}/request-physical` - Request physical card

## Security

- **CVV**: Only returned once during card creation, then hashed with Argon2
- **Card Number**: Masked in all responses except creation
- **Sensitive Fields**: CVVHash never exposed in JSON responses

## Files Changed

**Models:**

- `3-wallet/src/models/virtual_card.go` - Changed `BankAccountID` to `WalletId`
- `3-wallet/src/models/events/card_events.go` - Updated event struct

**Handlers:**

- `3-wallet/src/server/handlers/virtual_card.go` - All handlers updated

**Database:**

- `3-wallet/src/db/implementations/memory.go` - Updated references
- `3-wallet/src/db/implementations/redis.go` - Updated references

**Events:**

- `3-wallet/src/events/consumer/account_created_events.go` - Added default card creation
- `3-wallet/src/events/producer/card_events.go` - Kafka event producers

**Documentation:**

- `3-wallet/WALLET_API.md` - Wallet API documentation
- `3-wallet/VIRTUAL_CARDS_API.md` - Virtual card API documentation
- `3-wallet/INTEGRATION_SUMMARY.md` - This file
