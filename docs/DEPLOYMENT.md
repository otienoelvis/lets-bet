# Deployment Guide - BCLB Compliant Betting Platform

This guide covers the complete deployment process from local development to production, with specific focus on **BCLB (Betting Control and Licensing Board)** requirements for Kenya.

---

## Pre-Deployment Checklist

### **1. Legal & Regulatory**
- [ ] Company registered in Kenya (30% local shareholding recommended)
- [ ] BCLB application submitted (KES 1,000,000 fee)
- [ ] Security bond provided (Bank guarantee)
- [ ] Directors vetted by BCLB
- [ ] Tax compliance certificate (KRA)
- [ ] Insurance policy obtained
- [ ] Terms & Conditions drafted (reviewed by lawyer)
- [ ] Privacy Policy (GDPR/PDPA compliant)
- [ ] Responsible Gaming policy documented

### **2. Technical Infrastructure**
- [ ] AWS account created (recommend af-south-1 Cape Town region)
- [ ] Domain purchased and DNS configured
- [ ] SSL certificate obtained (Let's Encrypt or AWS ACM)
- [ ] Cloudflare account for DDoS protection
- [ ] Safaricom Daraja API credentials (production)
- [ ] Airtel Money API credentials
- [ ] Sportradar/Betgenius odds feed contract
- [ ] Smile ID or Metamap KYC service

### **3. Security**
- [ ] Penetration testing completed
- [ ] Security audit report available
- [ ] PCI-DSS compliance (if storing payment data)
- [ ] Regular backup strategy implemented
- [ ] Disaster recovery plan documented
- [ ] Incident response plan ready

---

## Infrastructure Setup

### **Option A: AWS (Recommended for Production)**

#### **1. Network Architecture**
```
┌────────────────────────────────────────────────────────┐
│                   CLOUDFLARE CDN/WAF                    │
│  DDoS Protection | Rate Limiting | SSL Termination     │
└────────────────────┬───────────────────────────────────┘
                     │
┌────────────────────▼───────────────────────────────────┐
│              AWS Application Load Balancer             │
│                 (af-south-1 - Cape Town)               │
└────┬────────────────────┬─────────────────────┬────────┘
     │                    │                     │
┌────▼─────┐      ┌──────▼──────┐      ┌──────▼──────┐
│ Gateway  │      │   Wallet    │      │   Games     │
│ EC2/ECS  │      │   EC2/ECS   │      │   EC2/ECS   │
└────┬─────┘      └──────┬──────┘      └──────┬──────┘
     │                    │                     │
     └────────────────────┼─────────────────────┘
                          │
             ┌────────────▼────────────┐
             │  RDS PostgreSQL (Multi-AZ)│
             │  + Read Replica for BCLB │
             └─────────────────────────┘
```

#### **2. RDS PostgreSQL Setup**
```bash
# Create production database
aws rds create-db-instance \
  --db-instance-identifier betting-platform-prod \
  --db-instance-class db.r6g.xlarge \
  --engine postgres \
  --engine-version 15.4 \
  --master-username admin \
  --master-user-password <STRONG_PASSWORD> \
  --allocated-storage 100 \
  --storage-type gp3 \
  --multi-az \
  --backup-retention-period 30 \
  --region af-south-1

# Create read replica for BCLB access
aws rds create-db-instance-read-replica \
  --db-instance-identifier betting-platform-bclb-mirror \
  --source-db-instance-identifier betting-platform-prod \
  --region af-south-1
```

#### **3. ElastiCache Redis**
```bash
aws elasticache create-cache-cluster \
  --cache-cluster-id betting-redis-prod \
  --cache-node-type cache.r6g.large \
  --engine redis \
  --num-cache-nodes 1 \
  --region af-south-1
```

#### **4. Application Deployment (ECS Fargate)**
```yaml
# ecs-task-definition.json
{
  "family": "betting-gateway",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "containerDefinitions": [
    {
      "name": "gateway",
      "image": "your-ecr-repo/betting-gateway:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {"name": "COUNTRY_CODE", "value": "KE"},
        {"name": "CURRENCY", "value": "KES"}
      ],
      "secrets": [
        {"name": "DATABASE_PASSWORD", "valueFrom": "arn:aws:secretsmanager:..."},
        {"name": "MPESA_CONSUMER_KEY", "valueFrom": "arn:aws:secretsmanager:..."}
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/betting-gateway",
          "awslogs-region": "af-south-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

---

## Kenya-Specific Configuration

### **1. M-Pesa Production Setup**

#### **Step 1: Get Production Credentials**
1. Go to https://developer.safaricom.co.ke
2. Create an app (select "Lipa Na M-Pesa Online" and "B2C")
3. Apply for production access
4. Wait for approval (usually 5-7 business days)

#### **Step 2: Configure Webhooks**
```bash
# Your public endpoints for M-Pesa callbacks
https://api.yourdomain.com/api/mpesa/callback      # STK Push results
https://api.yourdomain.com/api/mpesa/b2c-result    # B2C payment results
https://api.yourdomain.com/api/mpesa/timeout       # Timeout notifications
```

#### **Step 3: Test Production M-Pesa**
```bash
# Test deposit (STK Push)
curl -X POST https://api.yourdomain.com/api/v1/payments/deposit \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "phone_number": "254712345678",
    "amount": 100
  }'

# User receives M-Pesa prompt on their phone
# After entering PIN, callback is triggered
```

### **2. BCLB Database Access**

The BCLB requires **direct access** to your database for auditing. Set up a **read-only user**:

```sql
-- Create read-only user for BCLB inspectors
CREATE USER bclb_inspector WITH PASSWORD 'secure_password_here';

-- Grant SELECT on all tables
GRANT CONNECT ON DATABASE betting_db TO bclb_inspector;
GRANT USAGE ON SCHEMA public TO bclb_inspector;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO bclb_inspector;

-- Ensure future tables are also readable
ALTER DEFAULT PRIVILEGES IN SCHEMA public 
  GRANT SELECT ON TABLES TO bclb_inspector;
```

**Whitelist BCLB IP addresses:**
```
# PostgreSQL pg_hba.conf
host    betting_db    bclb_inspector    196.201.XXX.XXX/32    md5
```

### **3. Tax Automation**

```go
// Automatically deduct 20% WHT on every winning payout
func ProcessWinningPayout(bet *domain.Bet) error {
    profit := bet.ActualWin.Sub(bet.Stake)
    
    if profit.GreaterThan(decimal.Zero) {
        // Kenya: 20% tax on profit
        tax := profit.Mul(decimal.NewFromFloat(0.20))
        bet.TaxAmount = tax
        
        netPayout := bet.ActualWin.Sub(tax)
        
        // Credit wallet with net amount
        // Store tax in separate ledger for KRA reporting
    }
}
```

---

## Security Hardening

### **1. Firewall Rules (AWS Security Groups)**
```
Inbound Rules:
- Port 443 (HTTPS) from 0.0.0.0/0 (via ALB only)
- Port 22 (SSH) from YOUR_OFFICE_IP/32 only
- Port 5432 (PostgreSQL) from app subnets only

Outbound Rules:
- Port 443 (HTTPS) to 0.0.0.0/0 (for M-Pesa API calls)
- Port 5432 (PostgreSQL) within VPC
```

### **2. Secrets Management**
```bash
# Store sensitive credentials in AWS Secrets Manager
aws secretsmanager create-secret \
  --name betting-platform/mpesa \
  --secret-string '{
    "consumer_key": "your_key",
    "consumer_secret": "your_secret",
    "passkey": "your_passkey"
  }'
```

### **3. Rate Limiting (Application Level)**
```go
// Prevent abuse
- Login attempts: 5 per minute per IP
- Bet placement: 10 per minute per user
- Withdrawal requests: 3 per hour per user
- API calls: 100 per minute per IP
```

---

## Monitoring & Alerts

### **1. CloudWatch Alarms**
```bash
# High error rate
aws cloudwatch put-metric-alarm \
  --alarm-name high-error-rate \
  --metric-name Errors \
  --namespace AWS/ApplicationELB \
  --statistic Sum \
  --period 300 \
  --threshold 100 \
  --comparison-operator GreaterThanThreshold

# Database CPU
aws cloudwatch put-metric-alarm \
  --alarm-name db-high-cpu \
  --metric-name CPUUtilization \
  --namespace AWS/RDS \
  --statistic Average \
  --period 300 \
  --threshold 80 \
  --comparison-operator GreaterThanThreshold
```

### **2. Application Metrics**
```
Key Metrics to Track:
- Bet placement latency (p50, p95, p99)
- Wallet transaction success rate
- M-Pesa callback success rate
- WebSocket connection count
- Active game sessions
- Daily Active Users (DAU)
- Gross Gaming Revenue (GGR)
- Total bets placed (by country)
```

---

## Load Testing

Before going live, test your system at **2x expected peak load**:

```bash
# Install k6
brew install k6

# Run load test
k6 run --vus 10000 --duration 5m loadtest.js
```

**loadtest.js:**
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export default function() {
  // Place a bet
  let res = http.post('https://api.yourdomain.com/api/v1/bets', {
    stake: 100,
    selections: [...]
  });
  
  check(res, {
    'status is 201': (r) => r.status === 201,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  sleep(1);
}
```

**Target Benchmarks:**
- 100,000 concurrent WebSocket connections
- 10,000 bets/second
- P95 latency < 200ms
- 0% error rate

---

## Go-Live Checklist

### **Day Before Launch**
- [ ] Final security scan
- [ ] Database backup
- [ ] M-Pesa production credentials verified
- [ ] BCLB final inspection passed
- [ ] Load testing completed
- [ ] Monitoring dashboards set up
- [ ] On-call team briefed

### **Launch Day**
- [ ] Deploy to production
- [ ] Smoke tests passed
- [ ] M-Pesa deposits working
- [ ] M-Pesa withdrawals working
- [ ] Crash game running smoothly
- [ ] Mobile app (APK) uploaded to website

### **Post-Launch (First Week)**
- [ ] Monitor error rates 24/7
- [ ] Track M-Pesa callback failures
- [ ] Review user feedback
- [ ] Fix critical bugs immediately
- [ ] Daily GGR reporting to finance team

---

## Emergency Contacts

**Infrastructure Issues:**
- AWS Support: +27 11 243 3000
- Cloudflare Support: support@cloudflare.com

**Payment Providers:**
- Safaricom Daraja Support: apisupport@safaricom.co.ke
- Airtel Money Support: airtelmoneyke@airtel.com

**Regulatory:**
- BCLB Hotline: +254 20 329 5000
- KRA Tax Compliance: +254 20 310 9000

---

## Success Metrics (First Month)

| Metric | Target |
|--------|--------|
| Registered Users | 10,000+ |
| Daily Active Users | 2,000+ |
| Gross Gaming Revenue | KES 5,000,000+ |
| M-Pesa Success Rate | 99%+ |
| System Uptime | 99.9%+ |
| Average Payout Time | < 2 minutes |

---

**You are now ready to launch a Tier-1 betting platform in Kenya!**
