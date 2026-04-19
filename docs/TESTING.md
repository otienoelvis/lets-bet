# Testing Guide

## Quick Start Testing

### 1. Start the Platform
```bash
# From the betting-platform directory
make dev-setup
make run-all
```

### 2. Open the Demo Frontend
```bash
# Open web/index.html in your browser
open web/index.html
# Or on Linux
xdg-open web/index.html
```

### 3. Test API Endpoints

#### Health Check
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy","service":"gateway"}
```

#### Check Wallet Balance
```bash
curl http://localhost:8080/api/v1/users/me/wallet
```

#### Place a Bet
```bash
curl -X POST http://localhost:8080/api/v1/bets \
  -H "Content-Type: application/json" \
  -d '{
    "stake": 100,
    "bet_type": "SINGLE",
    "selections": [{
      "event_id": "match_123",
      "market_id": "1X2",
      "outcome": "home",
      "odds": 2.50
    }]
  }'
```

#### Initiate M-Pesa Deposit
```bash
curl -X POST http://localhost:8080/api/v1/payments/deposit \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "254712345678",
    "amount": 500
  }'
```

### 4. Test WebSocket (Crash Game)

Using JavaScript in browser console:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/games/crash-123');

ws.onmessage = (event) => {
  console.log('Game update:', JSON.parse(event.data));
};

// Place a bet
ws.send(JSON.stringify({
  action: 'place_bet',
  amount: 100
}));

// Cashout
ws.send(JSON.stringify({
  action: 'cashout'
}));
```

## Load Testing

### Install k6
```bash
# macOS
brew install k6

# Ubuntu
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### Run Load Test
```bash
k6 run scripts/loadtest.js
```

## Database Testing

### Connect to PostgreSQL
```bash
docker exec -it betting-postgres psql -U betting_user -d betting_db
```

### Useful SQL Queries
```sql
-- Check all users
SELECT * FROM users;

-- Check wallet balances
SELECT u.phone_number, w.balance, w.currency 
FROM users u 
JOIN wallets w ON u.id = w.user_id;

-- Check recent bets
SELECT * FROM bets ORDER BY placed_at DESC LIMIT 10;

-- Check transactions
SELECT * FROM transactions ORDER BY created_at DESC LIMIT 20;

-- Get crash game history
SELECT round_number, crash_point, started_at 
FROM games 
WHERE game_type = 'CRASH' 
ORDER BY round_number DESC 
LIMIT 10;
```

## Integration Testing

### M-Pesa Sandbox Testing

1. **Get Sandbox Credentials**
   - Go to https://developer.safaricom.co.ke
   - Create an app
   - Copy Consumer Key and Consumer Secret

2. **Test STK Push**
```bash
# The phone number 254708374149 works in sandbox
curl -X POST http://localhost:8080/api/v1/payments/deposit \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "254708374149",
    "amount": 10
  }'
```

3. **Test Callback**
```bash
# Simulate M-Pesa callback
curl -X POST http://localhost:8080/api/mpesa/callback \
  -H "Content-Type: application/json" \
  -d '{
    "Body": {
      "stkCallback": {
        "MerchantRequestID": "test-123",
        "CheckoutRequestID": "test-456",
        "ResultCode": 0,
        "ResultDesc": "The service request is processed successfully."
      }
    }
  }'
```

## Crash Game Testing

### Manual Testing Steps

1. **Start the Games Service**
```bash
cd cmd/games
go run main.go
```

You should see:
```
Starting Crash Games Service
WebSocket Hub started
Crash Game Engine running
Round 1 prepared. Crash point: 3.52x (hidden)
Betting phase started for round 1
```

2. **Connect via WebSocket**
Open the demo frontend (web/index.html) and watch the multiplier increase in real-time.

3. **Verify Provably Fair**
```bash
# Run the Provably Fair verification test
go test ./internal/core/usecase -v -run TestProvablyFair
```

## Performance Benchmarks

### Expected Performance Targets

| Operation | Target Latency | Test Command |
|-----------|----------------|--------------|
| Wallet Balance Query | < 10ms | `ab -n 1000 -c 10 http://localhost:8081/internal/v1/wallets/user-id` |
| Place Bet | < 200ms | `ab -n 100 -c 5 -p bet.json http://localhost:8080/api/v1/bets` |
| WebSocket Message | < 50ms | Monitor browser dev tools Network tab |
| M-Pesa Deposit | < 30s | Manual test with real phone |

### Concurrent Connection Test

```bash
# Test 1000 concurrent WebSocket connections
node scripts/ws-stress-test.js 1000
```

## Debugging

### View Logs

```bash
# Gateway logs
docker logs betting-platform-gateway -f

# Database logs
docker logs betting-postgres -f

# Redis logs
docker logs betting-redis -f
```

### Common Issues

**Issue: Database connection failed**
```bash
# Solution: Check if PostgreSQL is running
docker ps | grep postgres

# Restart if needed
docker-compose restart postgres
```

**Issue: M-Pesa timeout**
```bash
# Solution: Verify you're using the correct environment
# Sandbox URL: https://sandbox.safaricom.co.ke
# Production URL: https://api.safaricom.co.ke
```

**Issue: WebSocket connection refused**
```bash
# Solution: Ensure Gateway service is running
curl http://localhost:8080/health

# Check if port 8080 is in use
lsof -i :8080
```

## Test Data

### Sample Users
```sql
INSERT INTO users (phone_number, email, password_hash, country_code, currency, full_name, date_of_birth, is_verified, status)
VALUES 
  ('254712345678', 'test1@example.com', '$2a$12$hashed', 'KE', 'KES', 'Test User One', '1990-01-01', true, 'ACTIVE'),
  ('254722334455', 'test2@example.com', '$2a$12$hashed', 'KE', 'KES', 'Test User Two', '1985-05-15', true, 'ACTIVE');
```

### Sample Wallets
```sql
INSERT INTO wallets (user_id, currency, balance)
SELECT id, 'KES', 5000.00
FROM users
WHERE phone_number IN ('254712345678', '254722334455');
```

## Automated Testing

### Unit Tests
```bash
# Run all unit tests
go test ./... -v

# Run specific package tests
go test ./internal/core/usecase -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests
```bash
# Requires Docker services to be running
make test
```

## Security Testing

### SQL Injection Test
```bash
# Should be prevented by parameterized queries
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "' OR 1=1--",
    "password": "anything"
  }'
```

### Rate Limiting Test
```bash
# Attempt 200 requests (should be rate limited after 100)
for i in {1..200}; do
  curl http://localhost:8080/api/v1/users/me/wallet &
done
wait
```

## Monitoring

### Prometheus Metrics (Future)
```
http://localhost:9090/metrics
```

### Health Endpoints

```bash
# All services
curl http://localhost:8080/health  # Gateway
curl http://localhost:8081/health  # Wallet
curl http://localhost:8082/health  # Engine
```

---

**Happy Testing!**

If you encounter any issues, check the logs or open an issue in the repository.
