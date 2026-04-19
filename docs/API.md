# API Documentation

Base URL: `http://localhost:8080/api/v1`

All authenticated endpoints require a JWT token in the Authorization header:
```
Authorization: Bearer YOUR_JWT_TOKEN
```

---

## Authentication

### Register User
**POST** `/auth/register`

Creates a new user account with KYC validation.

**Request:**
```json
{
  "phone_number": "254712345678",
  "email": "user@example.com",
  "password": "SecurePass123!",
  "full_name": "John Doe",
  "date_of_birth": "1990-01-15",
  "national_id": "12345678",
  "kra_pin": "A001234567Z"
}
```

**Response (201):**
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "Registration successful. Please verify your account.",
  "verification_required": true
}
```

### Login
**POST** `/auth/login`

Authenticates user and returns JWT token.

**Request:**
```json
{
  "phone_number": "254712345678",
  "password": "SecurePass123!"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 86400,
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "phone_number": "254712345678",
    "full_name": "John Doe",
    "is_verified": true
  }
}
```

---

## User Management

### Get User Profile
**GET** `/users/me`

Returns current user's profile.

**Response (200):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "phone_number": "254712345678",
  "email": "user@example.com",
  "full_name": "John Doe",
  "country_code": "KE",
  "currency": "KES",
  "is_verified": true,
  "created_at": "2026-01-15T10:30:00Z"
}
```

### Update Profile
**PUT** `/users/me`

**Request:**
```json
{
  "email": "newemail@example.com",
  "daily_deposit_limit": 5000
}
```

### Set Self-Exclusion
**POST** `/users/me/self-exclude`

**Request:**
```json
{
  "duration_days": 30,
  "reason": "Taking a break"
}
```

---

## Wallet & Payments

### Get Wallet Balance
**GET** `/users/me/wallet`

**Response (200):**
```json
{
  "balance": 5250.50,
  "currency": "KES",
  "bonus_balance": 100.00,
  "today_deposit": 1000.00,
  "daily_limit": 10000.00
}
```

### Deposit (M-Pesa)
**POST** `/payments/deposit`

Initiates M-Pesa STK Push.

**Request:**
```json
{
  "phone_number": "254712345678",
  "amount": 500
}
```

**Response (202):**
```json
{
  "checkout_request_id": "ws_CO_15012026103045678",
  "message": "STK Push sent. Please enter M-Pesa PIN on your phone.",
  "status": "PENDING"
}
```

### Withdraw (M-Pesa)
**POST** `/payments/withdraw`

Initiates instant M-Pesa payout.

**Request:**
```json
{
  "phone_number": "254712345678",
  "amount": 1000
}
```

**Response (202):**
```json
{
  "transaction_id": "txn_550e8400-e29b-41d4-a716",
  "message": "Withdrawal initiated. M-Pesa payout in progress.",
  "estimated_time": "60 seconds"
}
```

### Transaction History
**GET** `/payments/transactions?limit=20&offset=0`

**Response (200):**
```json
{
  "transactions": [
    {
      "id": "txn_123",
      "type": "DEPOSIT",
      "amount": 500.00,
      "currency": "KES",
      "status": "COMPLETED",
      "provider": "MPESA",
      "created_at": "2026-04-15T10:30:00Z"
    },
    {
      "id": "txn_124",
      "type": "BET_PLACED",
      "amount": -100.00,
      "currency": "KES",
      "status": "COMPLETED",
      "reference_id": "bet_789",
      "created_at": "2026-04-15T10:35:00Z"
    }
  ],
  "total": 45,
  "limit": 20,
  "offset": 0
}
```

---

## Sports Betting

### Get Live Matches
**GET** `/odds/live`

Returns all in-play matches with live odds.

**Response (200):**
```json
{
  "matches": [
    {
      "event_id": "match_12345",
      "sport": "football",
      "league": "English Premier League",
      "home_team": "Arsenal",
      "away_team": "Chelsea",
      "kick_off": "2026-04-15T15:00:00Z",
      "status": "LIVE",
      "score": "2-1",
      "markets": {
        "1X2": {
          "home": 1.85,
          "draw": 4.20,
          "away": 4.50
        },
        "over_under_2.5": {
          "over": 1.90,
          "under": 1.95
        }
      }
    }
  ]
}
```

### Place Bet
**POST** `/bets`

**Request (Single Bet):**
```json
{
  "bet_type": "SINGLE",
  "stake": 100,
  "selections": [
    {
      "event_id": "match_12345",
      "market_id": "1X2",
      "outcome_name": "Arsenal",
      "odds": 2.50
    }
  ]
}
```

**Request (Multi Bet):**
```json
{
  "bet_type": "MULTI",
  "stake": 500,
  "selections": [
    {
      "event_id": "match_12345",
      "market_id": "1X2",
      "outcome_name": "Arsenal",
      "odds": 2.10
    },
    {
      "event_id": "match_67890",
      "market_id": "1X2",
      "outcome_name": "Barcelona",
      "odds": 1.75
    }
  ]
}
```

**Response (201):**
```json
{
  "bet_id": "bet_550e8400-e29b-41d4",
  "status": "PENDING",
  "stake": 100.00,
  "total_odds": 2.50,
  "potential_win": 250.00,
  "placed_at": "2026-04-15T10:45:00Z",
  "message": "Bet placed successfully"
}
```

### Get Bet Details
**GET** `/bets/{bet_id}`

**Response (200):**
```json
{
  "id": "bet_550e8400-e29b-41d4",
  "user_id": "user_123",
  "bet_type": "SINGLE",
  "stake": 100.00,
  "currency": "KES",
  "total_odds": 2.50,
  "potential_win": 250.00,
  "actual_win": 0.00,
  "status": "PENDING",
  "selections": [
    {
      "event_id": "match_12345",
      "event_name": "Arsenal vs Chelsea",
      "market_name": "Match Winner",
      "outcome_name": "Arsenal",
      "odds": 2.50,
      "status": "PENDING"
    }
  ],
  "placed_at": "2026-04-15T10:45:00Z"
}
```

### Bet History
**GET** `/bets/history?status=ALL&limit=20&offset=0`

Query Parameters:
- `status`: ALL, PENDING, WON, LOST, VOID
- `limit`: Number of results (max 100)
- `offset`: Pagination offset

**Response (200):**
```json
{
  "bets": [
    {
      "id": "bet_123",
      "stake": 100.00,
      "total_odds": 3.50,
      "potential_win": 350.00,
      "actual_win": 350.00,
      "status": "WON",
      "placed_at": "2026-04-14T18:30:00Z",
      "settled_at": "2026-04-14T21:45:00Z"
    }
  ],
  "total": 127,
  "limit": 20,
  "offset": 0
}
```

---

## Crash Game (Aviator)

### Get Current Game
**GET** `/games/crash/current`

**Response (200):**
```json
{
  "game_id": "game_550e8400",
  "round_number": 42,
  "status": "RUNNING",
  "current_multiplier": 2.45,
  "server_seed_hash": "7a3f...e92c",
  "started_at": "2026-04-15T11:00:00Z",
  "active_players": 1247
}
```

### Game History
**GET** `/games/crash/history?limit=10`

**Response (200):**
```json
{
  "games": [
    {
      "round_number": 41,
      "crash_point": 3.52,
      "server_seed": "revealed_after_crash",
      "started_at": "2026-04-15T10:59:45Z",
      "crashed_at": "2026-04-15T11:00:03Z"
    },
    {
      "round_number": 40,
      "crash_point": 1.23,
      "server_seed": "revealed_after_crash",
      "started_at": "2026-04-15T10:59:30Z",
      "crashed_at": "2026-04-15T10:59:43Z"
    }
  ]
}
```

### Place Game Bet
**POST** `/games/crash/bet`

**Request:**
```json
{
  "amount": 100,
  "auto_cashout": 2.00
}
```

**Response (201):**
```json
{
  "bet_id": "game_bet_123",
  "game_id": "game_550e8400",
  "amount": 100.00,
  "status": "ACTIVE",
  "message": "Bet placed. Connect to WebSocket for live updates."
}
```

### Verify Game (Provably Fair)
**POST** `/games/crash/verify`

**Request:**
```json
{
  "round_number": 40,
  "server_seed": "revealed_seed",
  "client_seed": "combined_client_seed",
  "claimed_crash": 1.23
}
```

**Response (200):**
```json
{
  "verified": true,
  "calculated_crash": 1.23,
  "message": "Game result is provably fair"
}
```

---

## WebSocket API

### Connect to Crash Game
```
ws://localhost:8080/ws/games/{game_id}
```

### Client → Server Messages

**Place Bet:**
```json
{
  "action": "place_bet",
  "amount": 100,
  "auto_cashout": null
}
```

**Cashout:**
```json
{
  "action": "cashout",
  "bet_id": "game_bet_123"
}
```

### Server → Client Messages

**Game State Update:**
```json
{
  "type": "state_update",
  "game_id": "game_123",
  "round_number": 42,
  "status": "RUNNING",
  "current_multiplier": 2.45,
  "time_remaining": null,
  "active_players": 1247
}
```

**Bet Confirmed:**
```json
{
  "type": "bet_confirmed",
  "bet_id": "game_bet_123",
  "amount": 100.00
}
```

**Game Crashed:**
```json
{
  "type": "crashed",
  "crash_point": 3.52,
  "server_seed": "revealed_seed",
  "your_result": {
    "bet_id": "game_bet_123",
    "cashed_out": false,
    "payout": 0.00,
    "status": "LOST"
  }
}
```

**Cashout Successful:**
```json
{
  "type": "cashout_success",
  "bet_id": "game_bet_123",
  "multiplier": 2.45,
  "payout": 245.00
}
```

---

## Error Responses

All errors follow this format:

**Response (4xx/5xx):**
```json
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "Your balance is too low to place this bet",
    "details": {
      "required": 100.00,
      "available": 50.00
    }
  }
}
```

### Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `UNAUTHORIZED` | 401 | Invalid or missing authentication token |
| `INSUFFICIENT_BALANCE` | 400 | Wallet balance too low |
| `USER_NOT_VERIFIED` | 403 | KYC verification required |
| `SELF_EXCLUDED` | 403 | User has self-excluded |
| `INVALID_BET` | 400 | Bet configuration is invalid |
| `ODDS_CHANGED` | 409 | Odds have changed since bet was created |
| `MARKET_CLOSED` | 400 | Market is no longer accepting bets |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `MPESA_TIMEOUT` | 504 | M-Pesa service timeout |
| `GAME_NOT_ACTIVE` | 400 | Cannot bet on inactive game |

---

## Rate Limits

| Endpoint | Limit |
|----------|-------|
| `/auth/login` | 5 per minute per IP |
| `/bets` | 10 per minute per user |
| `/payments/withdraw` | 3 per hour per user |
| `/games/crash/bet` | 20 per minute per user |
| All other endpoints | 100 per minute per IP |

---

## Webhooks (M-Pesa Callbacks)

### STK Push Callback
**POST** `/api/mpesa/callback`

Safaricom sends this when a user completes/cancels the STK Push.

**Request from M-Pesa:**
```json
{
  "Body": {
    "stkCallback": {
      "MerchantRequestID": "29115-34620561-1",
      "CheckoutRequestID": "ws_CO_191220191020363925",
      "ResultCode": 0,
      "ResultDesc": "The service request is processed successfully.",
      "CallbackMetadata": {
        "Item": [
          {"Name": "Amount", "Value": 500},
          {"Name": "MpesaReceiptNumber", "Value": "NLJ7RT61SV"},
          {"Name": "TransactionDate", "Value": 20191219102115},
          {"Name": "PhoneNumber", "Value": 254708374149}
        ]
      }
    }
  }
}
```

---

**For more details, see the source code or contact support.**
