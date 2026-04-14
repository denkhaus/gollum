# Task 7: End-to-End CLI Testing Checklist

This checklist is for manual testing of the ACP transport CLI flags.

## Pre-requisites
- Build the binary: `go build -o gollum ./cmd/gollum`
- Binary available at: `./gollum`

## Test Cases

### Test 1: Stdio Transport (Existing Behavior)
**Purpose:** Verify backward compatibility with stdio mode

```bash
# Should work as before
./gollum acp

# Verify with explicit flag
./gollum acp --transport stdio
```

**Expected:** ACP starts in stdio mode
**Result:** [ ] PASS / [ ] FAIL

---

### Test 2: HTTP Transport with Defaults
**Purpose:** Verify HTTP server starts with default configuration

```bash
./gollum acp --transport http &
HTTP_PID=$!

# Wait for server to start
sleep 2

# Test connection (will fail ACP handshake but should reach server)
curl -X POST http://localhost:8080 -d '{"jsonrpc":"2.0","id":1,"method":"initialize"}' || true

# Cleanup
kill $HTTP_PID
```

**Expected:** HTTP server starts on port 8080
**Result:** [ ] PASS / [ ] FAIL

---

### Test 3: HTTP Transport with Custom Host/Port
**Purpose:** Verify custom host/port configuration works

```bash
./gollum acp --transport http --host 127.0.0.1 --port 9000 &
HTTP_PID=$!

sleep 2

curl -X POST http://127.0.0.1:9000 -d '{"jsonrpc":"2.0","id":1,"method":"initialize"}' || true

kill $HTTP_PID
```

**Expected:** HTTP server starts on custom port 9000
**Result:** [ ] PASS / [ ] FAIL

---

### Test 4: Invalid Transport Flag
**Purpose:** Verify error handling for invalid transport type

```bash
./gollum acp --transport invalid
```

**Expected:** Error message about unsupported transport type
**Result:** [ ] PASS / [ ] FAIL

---

### Test 5: Port Validation
**Purpose:** Verify port range validation (1-65535)

```bash
./gollum acp --transport http --port 99999
```

**Expected:** Error about port range
**Result:** [ ] PASS / [ ] FAIL

---

### Test 6: Environment Variable Configuration
**Purpose:** Verify environment variable support

```bash
GOLLUM_ACP_TRANSPORT=http GOLLUM_ACP_PORT=9000 ./gollum acp &
HTTP_PID=$!

sleep 2

curl http://localhost:9000 || true

kill $HTTP_PID
```

**Expected:** HTTP server reads config from environment variables
**Result:** [ ] PASS / [ ] FAIL

---

## Summary

**Tests Passed:** ___ / 6

**Notes:**
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________

**Issues Found:**
_______________________________________________________________
_______________________________________________________________
_______________________________________________________________
