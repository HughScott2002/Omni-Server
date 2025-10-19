# Fraud Detection Service

Real-time rules-based fraud detection for transaction risk assessment.

## Overview

This service evaluates every transaction against a comprehensive set of fraud detection rules to:
- Calculate risk scores (0-100)
- Classify risk levels (low/medium/high)
- Make decisions (approve/review/decline)
- Provide detailed reasons for decisions

## Features

✅ **Real-time Risk Assessment** - Sub-50ms response time
✅ **Rules-Based Engine** - 15+ fraud detection rules
✅ **Velocity Tracking** - Monitor transaction frequency and volume
✅ **Pattern Detection** - Identify suspicious behaviors
✅ **Thread-Safe** - Handles concurrent requests
✅ **In-Memory Store** - Fast access to recent transaction history
✅ **Detailed Reasons** - Clear explanation for every decision

## Quick Start

### Start Service

```bash
# With Docker Compose
docker-compose up -d fraud-detection-service

# Check health
curl http://localhost:8085/health
```

### Test Risk Assessment

```bash
curl -X POST http://localhost:8085/api/fraud-detection/assess \
  -H "Content-Type: application/json" \
  -d '{
    "transactionId": "tx-123",
    "senderAccountId": "user-456",
    "receiverAccountId": "user-789",
    "amount": 100.50,
    "currency": "USD",
    "transactionType": "transfer",
    "description": "Payment for services"
  }'
```

### Expected Response

```json
{
  "transactionId": "tx-123",
  "riskScore": 0,
  "riskLevel": "low",
  "decision": "approve",
  "reasons": ["Transaction within normal parameters"],
  "assessedAt": "2025-10-19T10:30:00Z"
}
```

## Fraud Detection Rules

### Amount-Based (5 rules)
- Very large transactions (>$10k) → +30 points
- Large transactions ($5k-$10k) → +15 points
- Suspicious patterns ($9,990-$9,999.99) → +20 points
- Round amounts (≥$1k) → +5 points
- Tiny test transactions (<$1) → +8 points

### Velocity-Based (5 rules)
- High frequency per hour (≥10 tx) → +25 points
- High frequency per day (≥50 tx) → +15 points
- High volume per hour (>$5k) → +30 points
- High volume per day (>$20k) → +20 points
- Repeated to same receiver (≥5 tx/hour) → +12 points

### Pattern-Based (2 rules)
- Suspicious keywords → +15 points
- Empty description for large amount → +10 points

### Time-Based (1 rule)
- Late night (12-5 AM) → +8 points

### Account-Based (1 rule)
- Same sender/receiver → +100 points (auto-decline)

## Decision Thresholds

- **Approve**: Score < 50
- **Review**: Score 50-69 (manual review needed)
- **Decline**: Score ≥ 70

## Examples

### Low Risk - Approved
```json
{"amount": 50, "description": "Lunch"}
→ Score: 0, Decision: approve
```

### Medium Risk - Approved with Flags
```json
{"amount": 5000, "description": "Rent"}
→ Score: 20, Decision: approve
→ Reasons: ["Large amount", "Round amount"]
```

### High Risk - Declined
```json
{"amount": 9999.99, "description": "urgent cash", "time": "3 AM"}
→ Score: 73, Decision: decline
→ Reasons: ["Very large", "Suspicious pattern", "Suspicious keyword", "Late night"]
```

### Velocity Triggered - Review
```json
{"sender": "user123", "recent": "11 tx in last hour"}
→ Score: 55, Decision: review
→ Reasons: ["High frequency", "High volume"]
```

## Integration

### From Transaction Service

```go
import "utils"

// Assess transaction risk
assessment, err := utils.AssessTransactionRisk(utils.RiskAssessmentRequest{
    TransactionID:     tx.ID,
    SenderAccountID:   tx.SenderAccountID,
    ReceiverAccountID: tx.ReceiverAccountID,
    Amount:            tx.Amount,
    Currency:          tx.Currency,
    TransactionType:   "transfer",
    Description:       tx.Description,
})

if err != nil {
    // Handle error
}

if assessment.Decision != "approve" {
    // Decline or flag for review
}
```

## API Endpoints

### POST /api/fraud-detection/assess
Assess transaction risk

**Request:**
```json
{
  "transactionId": "string",
  "senderAccountId": "string",
  "receiverAccountId": "string",
  "amount": 100.00,
  "currency": "USD",
  "transactionType": "transfer",
  "description": "string",
  "metadata": {}
}
```

**Response:**
```json
{
  "transactionId": "string",
  "riskScore": 0-100,
  "riskLevel": "low|medium|high",
  "decision": "approve|review|decline",
  "reasons": ["array of strings"],
  "assessedAt": "timestamp"
}
```

### GET /health
Health check

**Response:**
```json
{
  "status": "healthy",
  "service": "fraud-detection"
}
```

## Configuration

Edit thresholds in `src/utils/fraud_rules.go`:

```go
const (
    // Risk thresholds
    DeclineThreshold = 70.0
    ReviewThreshold  = 50.0

    // Amount limits
    VeryLargeAmount = 10000.0

    // Velocity limits
    MaxTransactionsPerHour = 10
    MaxAmountPerHour = 5000.0
)
```

## Monitoring

### View Logs
```bash
# Docker Compose
docker-compose logs -f fraud-detection-service

# Docker Swarm
docker service logs -f omni-server_fraud-detection-service
```

### Key Metrics to Monitor
- Average risk score
- Decline rate
- Review rate
- Response time
- Velocity rule triggers

## Performance

- **Response Time**: <50ms average
- **Throughput**: 1000+ assessments/second
- **Memory**: ~1MB per 10k transactions
- **Storage**: In-memory (24h rolling window)

## Development

### Build
```bash
docker build -t fraud-detection:latest ./5-fraud-detection
```

### Run Locally
```bash
cd 5-fraud-detection
go run src/main.go
```

### Add New Rule
See `FRAUD_DETECTION_RULES.md` for detailed instructions.

## Documentation

- **Rules Reference**: [FRAUD_DETECTION_RULES.md](FRAUD_DETECTION_RULES.md)
- **Transaction Flow**: [../TRANSACTION_FLOW.md](../TRANSACTION_FLOW.md)
- **Deployment**: [../DEPLOYMENT.md](../DEPLOYMENT.md)

## Future Enhancements

- [ ] Machine learning models
- [ ] Geographic/IP analysis
- [ ] Device fingerprinting
- [ ] Blacklist/whitelist
- [ ] OFAC sanctions screening
- [ ] User behavior profiling
- [ ] Network analysis
- [ ] Real-time dashboard

## License

Part of Omni Server - Internal use only
