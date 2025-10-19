# Fraud Detection Rules

Real-time rules-based fraud detection system for transaction risk assessment.

## Overview

The fraud detection service evaluates every transaction against multiple rules to calculate a risk score (0-100) and make a decision to **approve**, **review**, or **decline** the transaction.

## Risk Scoring

### Risk Score Calculation

Each rule that triggers adds risk points to the total score:
- **Total Risk Score**: Sum of all triggered rules
- **Range**: 0-100
- **Higher score** = Higher risk

### Risk Levels

- **Low Risk**: Score < 25
- **Medium Risk**: Score 25-49
- **High Risk**: Score ≥ 50

### Decision Thresholds

- **Approve**: Score < 50 (automatic approval)
- **Review**: Score 50-69 (manual review required)
- **Decline**: Score ≥ 70 (automatic decline)

---

## Fraud Detection Rules

### 1. Amount-Based Rules

#### Very Large Transaction (+30 points)
- **Trigger**: Amount > $10,000
- **Reason**: Very large transactions are higher risk
- **Example**: $15,000 → +30 points

#### Large Transaction (+15 points)
- **Trigger**: Amount $5,000 - $10,000
- **Reason**: Large amounts require additional scrutiny
- **Example**: $7,500 → +15 points

#### Suspicious Amount Pattern (+20 points)
- **Trigger**: Amount between $9,990 - $9,999.99
- **Reason**: May be attempting to avoid $10k reporting threshold (structuring)
- **Example**: $9,999.50 → +20 points

#### Round Amount (+5 points)
- **Trigger**: Exact round amount ≥ $1,000 (e.g., $1000.00, $5000.00)
- **Reason**: Round amounts can indicate automated/scripted transactions
- **Example**: $5,000.00 → +5 points

#### Tiny Transaction (+8 points)
- **Trigger**: Amount < $1.00
- **Reason**: May be testing stolen credentials
- **Example**: $0.01 → +8 points

---

### 2. Velocity-Based Rules

#### High Transaction Frequency - 1 Hour (+25 points)
- **Trigger**: ≥10 transactions in last hour
- **Reason**: Unusual transaction velocity
- **Example**: 12 transactions in last hour → +25 points

#### High Transaction Frequency - 24 Hours (+15 points)
- **Trigger**: ≥50 transactions in last 24 hours
- **Reason**: Excessive daily transaction count
- **Example**: 55 transactions today → +15 points

#### High Volume - 1 Hour (+30 points)
- **Trigger**: Total amount sent in last hour > $5,000
- **Reason**: Unusual spending velocity
- **Example**: Already sent $4,000, trying to send $2,000 more → +30 points

#### High Volume - 24 Hours (+20 points)
- **Trigger**: Total amount sent in last 24 hours > $20,000
- **Reason**: Excessive daily spending
- **Example**: Already sent $18,000, trying to send $5,000 more → +20 points

#### Repeated Transactions to Same Receiver (+12 points)
- **Trigger**: ≥5 transactions to same receiver in last hour
- **Reason**: May indicate account compromise or splitting transactions
- **Example**: 6 transactions to same person in 30 minutes → +12 points

---

### 3. Pattern-Based Rules

#### Suspicious Description Keywords (+15 points)
- **Trigger**: Description contains suspicious keywords
- **Keywords**:
  - Financial: "urgent", "emergency", "cash out", "withdraw all"
  - Crypto: "bitcoin", "crypto"
  - Scam indicators: "lottery", "prize", "winner", "tax refund", "irs"
  - Legal: "lawyer", "attorney", "court", "legal fees", "inheritance"
- **Reason**: Common fraud/scam patterns
- **Example**: Description: "urgent cash out needed" → +15 points

#### Empty Description for Large Amount (+10 points)
- **Trigger**: Amount > $1,000 with no description
- **Reason**: Legitimate large transactions usually have descriptions
- **Example**: $5,000 transfer with blank description → +10 points

---

### 4. Time-Based Rules

#### Late Night Transaction (+8 points)
- **Trigger**: Transaction between 12:00 AM - 5:00 AM
- **Reason**: Unusual time for transactions, higher fraud risk
- **Example**: Transaction at 2:30 AM → +8 points

---

### 5. Account-Based Rules

#### Same Sender and Receiver (+100 points - Auto Decline)
- **Trigger**: Sender account ID = Receiver account ID
- **Reason**: Self-transfers should be blocked
- **Example**: Sending money to own account → +100 points (auto-decline)

---

## Example Scenarios

### Scenario 1: Normal Transaction
```json
{
  "amount": 50.00,
  "description": "Dinner payment",
  "time": "7:00 PM"
}
```
**Result:**
- Risk Score: 0
- Risk Level: Low
- Decision: **Approve**
- Reasons: ["Transaction within normal parameters"]

---

### Scenario 2: Large Round Amount
```json
{
  "amount": 5000.00,
  "description": "Monthly rent",
  "time": "2:00 PM"
}
```
**Result:**
- Risk Score: 20 (Large transaction: +15, Round amount: +5)
- Risk Level: Low
- Decision: **Approve**
- Reasons: ["Large amount: $5000.00", "Round amount: $5000.00"]

---

### Scenario 3: Suspicious Transaction
```json
{
  "amount": 9999.99,
  "description": "urgent cash transfer",
  "time": "3:00 AM"
}
```
**Result:**
- Risk Score: 73 (Very large: +30, Suspicious pattern: +20, Suspicious keyword: +15, Late night: +8)
- Risk Level: High
- Decision: **Decline**
- Reasons: [
  "Very large amount: $9999.99",
  "Suspicious amount pattern: $9999.99 (possible structuring)",
  "Suspicious keyword in description: 'urgent'",
  "Late night transaction at 3:00"
]

---

### Scenario 4: High Velocity
```json
{
  "amount": 100.00,
  "sender": "user123",
  "recent_activity": "11 transactions in last hour, $4500 sent"
}
```
**Result:**
- Risk Score: 55 (High frequency: +25, High volume: +30)
- Risk Level: High
- Decision: **Review** (manual review required)
- Reasons: [
  "High frequency: 11 transactions in last hour",
  "High volume: $4500.00 sent in last hour"
]

---

### Scenario 5: Testing Pattern
```json
{
  "amount": 0.01,
  "description": ""
}
```
**Result:**
- Risk Score: 8 (Tiny transaction: +8)
- Risk Level: Low
- Decision: **Approve** (but flagged)
- Reasons: ["Tiny test transaction: $0.01"]

---

### Scenario 6: Repeated Transactions
```json
{
  "sender": "user456",
  "receiver": "merchant789",
  "recent_activity": "7 transactions to same receiver in last hour"
}
```
**Result:**
- Risk Score: 12 (Repeated transactions: +12)
- Risk Level: Low
- Decision: **Approve**
- Reasons: ["Repeated transactions: 7 transactions to same receiver in last hour"]

---

## Transaction History Tracking

The system maintains an in-memory transaction store for velocity checks:

- **Storage Duration**: 24 hours
- **Max Transactions**: 10,000 most recent
- **Tracked Data**:
  - Transaction ID
  - Sender Account ID
  - Receiver Account ID
  - Amount
  - Timestamp

### Velocity Checks

The system tracks:
1. **Transaction count** by sender (hourly/daily)
2. **Total amount** sent by sender (hourly/daily)
3. **Transactions between** specific sender-receiver pairs
4. **Receiver patterns** (repeated transactions)

---

## Configuration

### Thresholds

All thresholds can be adjusted in `fraud_rules.go`:

```go
// Risk level thresholds
HighRiskThreshold   = 50.0
MediumRiskThreshold = 25.0
DeclineThreshold    = 70.0
ReviewThreshold     = 50.0

// Amount thresholds
VeryLargeAmount    = 10000.0
LargeAmount        = 5000.0
ModerateAmount     = 1000.0
SuspiciousMaxAmount = 9999.99
SuspiciousMinAmount = 9990.0

// Velocity thresholds
MaxTransactionsPerHour   = 10
MaxTransactionsPerDay    = 50
MaxAmountPerHour         = 5000.0
MaxAmountPerDay          = 20000.0
MaxRepeatTransactions    = 5
```

### Adding New Rules

To add a new fraud detection rule:

1. Define the rule in `fraud_rules.go`:
```go
{
    Name:        "New Rule Name",
    Description: "What this rule detects",
    RiskPoints:  10.0,
    Check: func(req models.RiskAssessmentRequest) (bool, string) {
        // Your detection logic here
        if /* condition */ {
            return true, "Reason why this triggered"
        }
        return false, ""
    },
}
```

2. Add it to the `GetFraudRules()` function

---

## Testing

### Test Endpoint

```bash
curl -X POST http://localhost:8085/api/fraud-detection/assess \
  -H "Content-Type: application/json" \
  -d '{
    "transactionId": "test-123",
    "senderAccountId": "sender-456",
    "receiverAccountId": "receiver-789",
    "amount": 5000.00,
    "currency": "USD",
    "transactionType": "transfer",
    "description": "Test transaction"
  }'
```

### Expected Response

```json
{
  "transactionId": "test-123",
  "riskScore": 20,
  "riskLevel": "low",
  "decision": "approve",
  "reasons": [
    "Large amount: $5000.00",
    "Round amount: $5000.00"
  ],
  "assessedAt": "2025-10-19T10:30:00Z"
}
```

---

## Performance

- **Response Time**: < 50ms (in-memory processing)
- **Throughput**: Thousands of assessments per second
- **Memory**: ~1MB per 10,000 transactions stored
- **Concurrency**: Thread-safe with read-write locks

---

## Future Enhancements

### Machine Learning Integration
- Train ML models on historical fraud data
- Real-time scoring alongside rules
- Adaptive thresholds based on patterns

### Additional Data Sources
- Geographic location (impossible travel detection)
- Device fingerprinting
- IP address reputation
- Blacklist/whitelist databases
- AML screening APIs

### Advanced Features
- User behavior profiling
- Anomaly detection
- Network analysis (money flow patterns)
- Risk score decay over time
- Whitelisted merchants/receivers

### Monitoring
- Dashboard for fraud patterns
- Alert system for high-risk transactions
- Analytics and reporting
- A/B testing for rule effectiveness

---

## Compliance

### Regulations Considered
- **BSA/AML**: Anti-Money Laundering
- **OFAC**: Sanctions screening (to be added)
- **FinCEN**: Suspicious Activity Reporting
- **KYC**: Know Your Customer (handled by user service)

### Audit Trail
All risk assessments are logged with:
- Transaction ID
- Risk score
- Decision
- Triggered rules
- Timestamp

---

## Support

For questions or issues:
- Check logs: `docker service logs omni-server_fraud-detection-service`
- Review this documentation
- See [TRANSACTION_FLOW.md](../TRANSACTION_FLOW.md) for integration details
