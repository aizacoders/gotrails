# gotrails — Global Audit Trail System for Go

> **gotrails** is a global audit trail library for Go services.  
> **ONE request = ONE trail log**, containing ALL internal steps and ALL integrations (HTTP, external API, Kafka, database, cache, gRPC, etc).

---

## 🎯 Core Principle

> ❗ **One Request → One Global Trail**
> - ❌ NOT event-per-log  
> - ✅ ONE GLOBAL LOG  
> - All processes are collected, then flushed once at the end

```
Incoming Request
       ↓
   Create Trail
       ↓
  Internal Steps
       ↓
Integrations (HTTP / Kafka / DB / Cache / gRPC)
       ↓
    Response
       ↓
  FLUSH → 1 JSON LOG
```

---

## 🧠 Trail Definition

**Trail** = complete representation of a single request lifecycle.

**Contents:**
- Incoming request
- Outgoing response
- Internal steps
- ALL integrations
- Errors
- Metadata
- Total latency

---

## 📜 Global Trail Log Structure

```json
{
  "timestamp": "2026-01-23T01:12:45.123Z",
  "trace_id": "req-abc-123",
  "request_id": "req-abc-123",
  "service": "payment-service",
  "environment": "production",
  "request": {},
  "response": {},
  "latency_ms": 512,
  "internal_steps": [],
  "integrations": [],
  "errors": [],
  "metadata": {}
}
```

> All fields above are part of the **gotrails contract**.

---

## 🔑 Trace & Context Rules

- **trace_id**: Extract from `X-Trace-ID` header if present, generate if not
- TraceID must bind all end-to-end flows in logs
- Trail is stored in context
- All internal & integration operations append to trail

---

## 🧩 Trail Components

### 1️⃣ Incoming Request

```json
{
  "request": {
    "method": "POST",
    "path": "/v1/payments",
    "headers": {
      "content-type": ["application/json"]
    },
    "body": {
      "amount": 150000,
      "payment_method": "bank_transfer"
    }
  }
}
```

**Rules:**
- Body masked
- Size limited
- Header filtered

---

### 2️⃣ Outgoing Response

```json
{
  "response": {
    "status": 200,
    "body": {
      "payment_id": "pay-123",
      "status": "PENDING"
    }
  }
}
```

---

### 3️⃣ Internal Steps (`internal_steps[]`)

Used for:
- Usecase
- Service
- Repository
- Validation

```json
{
  "internal_steps": [
    {
      "name": "ValidateRequest",
      "latency_ms": 12
    },
    {
      "name": "CreatePaymentUsecase",
      "latency_ms": 45
    }
  ]
}
```

**Rules:**
- Ordered by execution
- Request/response body is optional

---

### 4️⃣ Integrations (`integrations[]`) — CORE FEATURE 🔥

> `integrations[]` is the **CORE** of gotrails.  
> All IO outside service boundary **MUST** go here.

#### Integration Base Shape

```json
{
  "type": "http | kafka | database | cache | grpc | custom",
  "name": "string",
  "latency_ms": 123,
  "request": {},
  "response": {},
  "error": null,
  "metadata": {}
}
```

---

#### 4.1 External/Internal HTTP API

```json
{
  "type": "http",
  "name": "midtrans.charge",
  "latency_ms": 342,
  "request": {
    "method": "POST",
    "url": "https://api.midtrans.com/v2/charge",
    "headers": {
      "authorization": ["***"]
    },
    "body": {
      "order_id": "ord-789",
      "gross_amount": 150000
    }
  },
  "response": {
    "status": 201,
    "body": {
      "transaction_id": "trx-456",
      "transaction_status": "pending"
    }
  }
}
```

---

#### 4.2 Kafka Integration

```json
{
  "type": "kafka",
  "name": "payment-events",
  "latency_ms": 18,
  "request": {
    "action": "produce",
    "topic": "payment.created",
    "key": "pay-123",
    "value": {
      "payment_id": "pay-123",
      "status": "PENDING"
    }
  },
  "response": {
    "partition": 3,
    "offset": 98123
  }
}
```

---

#### 4.3 Database Integration

```json
{
  "type": "database",
  "name": "postgres.payments.insert",
  "latency_ms": 22,
  "request": {
    "query": "INSERT INTO payments (...)",
    "args": ["pay-123", 150000]
  },
  "response": {
    "rows_affected": 1
  }
}
```

---

#### 4.4 Cache Integration (Redis)

```json
{
  "type": "cache",
  "name": "redis.set",
  "latency_ms": 3,
  "request": {
    "command": "SET",
    "key": "payment:pay-123"
  },
  "response": {
    "status": "OK"
  }
}
```

---

#### 4.5 gRPC Integration

```json
{
  "type": "grpc",
  "name": "UserService.GetUser",
  "latency_ms": 14,
  "request": {
    "user_id": "u-123"
  },
  "response": {
    "email": "user@email.com"
  }
}
```

---

### 5️⃣ Errors (`errors[]`)

```json
{
  "errors": [
    {
      "source": "midtrans.charge",
      "message": "timeout while calling external API"
    }
  ]
}
```

---

### 6️⃣ Metadata (Free-form)

```json
{
  "metadata": {
    "user_id": "u-123",
    "merchant_id": "m-789",
    "order_id": "ord-789"
  }
}
```

---

## 🛣 Roadmap

### Phase 1 — Core Global Trail ✅

**Target:** ONE trail per request

- [x] Trail struct & lifecycle
- [x] Context-based trail storage
- [x] net/http middleware (Gin, httprouter)
- [x] Global flush at response end
- [x] Masking & size limit
- [x] Stdout + async sink

**Output:** 1 JSON log per request

---

### Phase 2 — Integration Collectors ✅

- [x] HTTP RoundTripper → append to `integrations[]`
- [x] Kafka producer wrapper
- [x] SQL wrapper (DB sink)
- [x] Redis wrapper (Cache sink)
- [x] gRPC interceptor
- [x] Error capture

---

### Phase 3 — Internal Steps API 🚧

- [x] `StartStep` / `EndStep`
- [x] `TraceStep(ctx, name, fn)`
- [x] Auto latency capture

---

### Phase 4 — Advanced & Compliance

- [x] Sampling
- [x] Immutable trail
- [x] Hash chaining
- [x] OpenTelemetry bridge

---

### Phase 5 — OSS Polish

- [x] README with examples
- [x] CI
- [x] v1.0.0 release

---

## 🧾 Final Statement

- **gotrails** = Global Audit Trail
- `integrations[]` = core field
- All IO goes into integrations
- Flush once, at the end
- This is **audit-grade**, not a logger

---

## 📋 Next Steps

1. Define final Go structs:
   - `Trail`
   - `Integration`
   - `InternalStep`
   - `Context lifecycle`

2. Implement Internal Steps API (Phase 3)

3. Add comprehensive tests and examples
