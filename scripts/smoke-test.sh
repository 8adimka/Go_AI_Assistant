#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting Smoke Test...${NC}\n"

# 1. Start docker-compose
echo -e "${YELLOW}1. Starting services...${NC}"
docker-compose up -d

# 2. Wait for services to be ready
echo -e "${YELLOW}2. Waiting for services to be ready...${NC}"

# Wait for MongoDB
echo -n "  Waiting for MongoDB..."
for i in {1..30}; do
    if docker-compose exec -T mongodb mongosh --eval "db.adminCommand('ping')" >/dev/null 2>&1; then
        echo -e " ${GREEN}✓${NC}"
        break
    fi
    sleep 1
    echo -n "."
done

# Wait for Redis
echo -n "  Waiting for Redis..."
for i in {1..30}; do
    if docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
        echo -e " ${GREEN}✓${NC}"
        break
    fi
    sleep 1
    echo -n "."
done

# 3. Start the server in background
echo -e "${YELLOW}3. Starting application server...${NC}"
go build -o /tmp/test-server ./cmd/server
/tmp/test-server > /tmp/server.log 2>&1 &
SERVER_PID=$!
echo "  Server PID: $SERVER_PID"

# Wait for server to be ready
echo -n "  Waiting for server to start..."
for i in {1..30}; do
    if curl -s http://localhost:8080/health >/dev/null 2>&1; then
        echo -e " ${GREEN}✓${NC}"
        break
    fi
    sleep 1
    echo -n "."
done

# 4. Run health check
echo -e "\n${YELLOW}4. Testing /health endpoint...${NC}"
HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
echo "  Response: $HEALTH_RESPONSE"

if echo "$HEALTH_RESPONSE" | grep -q '"status":"healthy"'; then
    echo -e "  ${GREEN}✓ Health check passed${NC}"
else
    echo -e "  ${RED}✗ Health check failed${NC}"
    kill $SERVER_PID
    exit 1
fi

if echo "$HEALTH_RESPONSE" | grep -q '"mongodb":"ok"'; then
    echo -e "  ${GREEN}✓ MongoDB connected${NC}"
else
    echo -e "  ${RED}✗ MongoDB not connected${NC}"
    kill $SERVER_PID
    exit 1
fi

if echo "$HEALTH_RESPONSE" | grep -q '"redis":"ok"'; then
    echo -e "  ${GREEN}✓ Redis connected${NC}"
else
    echo -e "  ${RED}✗ Redis not connected${NC}"
    kill $SERVER_PID
    exit 1
fi

# 5. Test metrics endpoint
echo -e "\n${YELLOW}5. Testing /metrics endpoint...${NC}"
METRICS_RESPONSE=$(curl -s http://localhost:8080/metrics)

if echo "$METRICS_RESPONSE" | grep -q 'go_info'; then
    echo -e "  ${GREEN}✓ Metrics endpoint working${NC}"
else
    echo -e "  ${RED}✗ Metrics endpoint not working${NC}"
    kill $SERVER_PID
    exit 1
fi

# 6. Test conversation flow
echo -e "\n${YELLOW}6. Testing conversation flow...${NC}"

# Start a conversation
echo "  Starting conversation..."
START_RESPONSE=$(curl -s -X POST http://localhost:8080/twirp/chat.ChatService/StartConversation \
  -H "Content-Type: application/json" \
  -d '{}')

CONV_ID=$(echo $START_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
echo "  Conversation ID: $CONV_ID"

if [ -z "$CONV_ID" ]; then
    echo -e "  ${RED}✗ Failed to start conversation${NC}"
    echo "  Response: $START_RESPONSE"
    kill $SERVER_PID
    exit 1
fi
echo -e "  ${GREEN}✓ Conversation started${NC}"

# Send a message (without expecting AI response since we may not have real API key)
echo "  Sending message..."
SEND_RESPONSE=$(curl -s -X POST http://localhost:8080/twirp/chat.ChatService/SendMessage \
  -H "Content-Type: application/json" \
  -d "{\"conversation_id\":\"$CONV_ID\",\"content\":\"Hello, this is a test message\"}")

if echo "$SEND_RESPONSE" | grep -q '"conversation_id"'; then
    echo -e "  ${GREEN}✓ Message sent successfully${NC}"
else
    echo "  Response: $SEND_RESPONSE"
    # Don't fail if OpenAI key is missing - that's expected in tests
    if echo "$SEND_RESPONSE" | grep -q "OPENAI_API_KEY"; then
        echo -e "  ${YELLOW}⚠ OpenAI API key not configured (expected in test)${NC}"
    else
        echo -e "  ${RED}✗ Failed to send message${NC}"
        kill $SERVER_PID
        exit 1
    fi
fi

# 7. Cleanup
echo -e "\n${YELLOW}7. Cleaning up...${NC}"
kill $SERVER_PID
rm -f /tmp/test-server
echo -e "  ${GREEN}✓ Server stopped${NC}"

echo -e "\n${GREEN}╔════════════════════════════════╗${NC}"
echo -e "${GREEN}║   Smoke Test Passed! 🎉       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════╝${NC}\n"
