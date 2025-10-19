# Virtual Cards API

## Overview

When a wallet is created via the `account-created` Kafka event, a default debit virtual card is automatically created with the following properties:
- Card Type: Debit
- Card Brand: Visa
- Currency: Same as wallet currency
- Daily/Monthly Limits: Same as wallet limits
- Status: Active (if wallet is active) or Pending (if KYC is pending)
- Name on Card: "Card Holder" (should be updated by user)

All card operations publish events to Kafka which trigger notifications via the notification service.

**Note:** All card endpoints are under `/api/wallets/cards` (not `/api/cards`) since nginx routes `/api/wallets` to this service.

## POST /api/wallets/cards

Create a new virtual card.

**Input:**
```json
{
  "walletId": "string",
  "cardType": "debit|credit",
  "cardBrand": "visa",
  "currency": "USD|EUR|GBP|JMD",
  "dailyLimit": 1000.00,
  "monthlyLimit": 5000.00,
  "nameOnCard": "string"
}
```

**Output (201):**
```json
{
  "message": "Virtual card created successfully",
  "card": { /* VirtualCard object */ },
  "cvv": "123",
  "maskedCardNumber": "**** **** **** 1234",
  "lastFourDigits": "1234"
}
```

**Note:** CVV is only returned on creation.

---

## GET /api/wallets/cards/{cardid}

Get a virtual card by ID.

**Input:** cardid (URL parameter)

**Output (200):**
```json
{
  "id": "string",
  "walletId": "string",
  "cardType": "debit|credit",
  "cardBrand": "visa",
  "currency": "USD|EUR|GBP|JMD",
  "cardStatus": "active|inactive|pending|blocked|expired",
  "dailyLimit": 1000.00,
  "monthlyLimit": 5000.00,
  "nameOnCard": "string",
  "cardNumber": "**** **** **** 1234",
  "expiryDate": "2028-01-01T00:00:00Z",
  "isActive": true,
  "availableBalance": 250.00,
  "totalToppedUp": 500.00,
  "totalSpendToday": 50.00,
  "totalSpentThisMonth": 250.00,
  "createdAt": "2025-01-01T00:00:00Z",
  "updatedAt": "2025-01-01T00:00:00Z"
}
```

---

## GET /api/wallets/cards/account/{accountid}

Get all virtual cards for an account.

**Input:** accountid (URL parameter)

**Output (200):**
```json
{
  "cards": [ /* array of VirtualCard objects with masked card numbers */ ],
  "count": 3
}
```

---

## PUT /api/wallets/cards/{cardid}

Update a virtual card.

**Input:**
```json
{
  "dailyLimit": 2000.00,
  "monthlyLimit": 10000.00,
  "isActive": true
}
```

All fields are optional.

**Output (200):**
```json
{
  "message": "Virtual card updated successfully",
  "card": { /* VirtualCard object */ }
}
```

---

## DELETE /api/wallets/cards/{cardid}

Delete a virtual card.

**Input:** cardid (URL parameter)

**Output (200):**
```json
{
  "message": "Virtual card deleted successfully",
  "deletedAt": "2025-01-01T00:00:00Z"
}
```

---

## POST /api/wallets/cards/{cardid}/block

Block a virtual card.

**Input:**
```json
{
  "blockReason": "lost|stolen|suspicious_activity|customer_request",
  "blockReasonDescription": "string"
}
```

**Output (200):**
```json
{
  "message": "Virtual card blocked successfully"
}
```

---

## POST /api/wallets/cards/{cardid}/topup

Top up a virtual card.

**Input:**
```json
{
  "accountNumber": "string",
  "amount": 100.00,
  "description": "string"
}
```

**Output (200):**
```json
{
  "status": "success",
  "message": "Card topped up successfully",
  "data": {
    "newBalance": 350.00,
    "amount": 100.00
  }
}
```

---

## POST /api/wallets/cards/{cardid}/request-physical

Request a physical card.

**Input:**
```json
{
  "deliveryAddress": "string",
  "city": "string",
  "country": "string",
  "postalCode": "string"
}
```

**Output (200):**
```json
{
  "message": "Physical card request submitted successfully",
  "status": "pending"
}
```

---

## Error Responses

All endpoints return appropriate HTTP status codes:
- 400: Bad Request (invalid input)
- 404: Not Found (card/wallet not found)
- 500: Internal Server Error

Error format:
```
Plain text error message
```
