# Wallet API

## GET /api/wallets/{walletId}

Get a wallet by ID, including all associated virtual cards.

**Input:** walletId (URL parameter)

**Output (200):**
```json
{
  "wallet": {
    "walletId": "string",
    "accountId": "string",
    "type": "PRIMARY|SAVINGS|ESCROW",
    "balance": 1000.00,
    "currency": "USD|EUR|GBP|JMD|TTD",
    "status": "active|inactive|suspended|disabled",
    "isDefault": true,
    "dailyLimit": 5000.00,
    "monthlyLimit": 20000.00,
    "lastActivity": "2025-01-01T00:00:00Z",
    "createdAt": "2025-01-01T00:00:00Z",
    "updatedAt": "2025-01-01T00:00:00Z"
  },
  "cards": [
    {
      "id": "string",
      "walletId": "string",
      "cardType": "debit|credit",
      "cardBrand": "visa",
      "currency": "USD",
      "cardStatus": "active|inactive|pending|blocked|expired",
      "dailyLimit": 5000.00,
      "monthlyLimit": 20000.00,
      "nameOnCard": "Card Holder",
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
  ]
}
```

---

## GET /api/wallets/list/{accountId}

Get all wallets for an account, each including its associated virtual cards.

**Input:** accountId (URL parameter)

**Output (200):**
```json
[
  {
    "wallet": {
      "walletId": "string",
      "accountId": "string",
      "type": "PRIMARY",
      "balance": 1000.00,
      "currency": "USD",
      "status": "active",
      "isDefault": true,
      "dailyLimit": 5000.00,
      "monthlyLimit": 20000.00,
      "createdAt": "2025-01-01T00:00:00Z",
      "updatedAt": "2025-01-01T00:00:00Z"
    },
    "cards": [
      {
        "id": "string",
        "walletId": "string",
        "cardType": "debit",
        "cardNumber": "**** **** **** 1234",
        ...
      }
    ]
  },
  {
    "wallet": {
      "walletId": "string",
      "accountId": "string",
      "type": "SAVINGS",
      "balance": 5000.00,
      "currency": "USD",
      "status": "active",
      "isDefault": false,
      ...
    },
    "cards": []
  }
]
```

---

## Notes

- Card numbers are always masked in wallet responses for security
- A default debit card is automatically created when a wallet is created
- Cards array will be empty `[]` if the wallet has no cards or if there's an error fetching them
- All timestamps are in ISO 8601 format
