# Go AI Assistant Telegram Bot

A Python Telegram bot that provides an interface to the Go AI Assistant system. This bot allows users to interact with the AI assistant through Telegram, with features including weather information, holiday data, and intelligent responses powered by OpenAI GPT-4.

## Features

- 🤖 **AI Assistant**: Get intelligent responses from OpenAI GPT-4
- 🌤️ **Weather Information**: Real-time weather data via WeatherAPI.com
- 📅 **Holiday Information**: Local bank and public holidays
- ⏰ **Current Date/Time**: Get current date and time information
- 💾 **Caching**: Redis caching for improved performance
- 🔄 **Fallback**: Mock data fallback when external APIs are unavailable

## Prerequisites

- Python 3.8+
- Go AI Assistant server running on `localhost:8080`
- Redis server (for caching)
- MongoDB (for conversation storage)
- Telegram Bot Token

## Quick Start

### 1. Setup Python Environment

```bash
# Navigate to the bot directory
cd python_telegram_bot

# Create virtual environment
python -m venv venv

# Activate virtual environment
source venv/bin/activate  # On Linux/Mac
# venv\Scripts\activate   # On Windows

# Install dependencies
pip install -r requirements.txt
```

### 2. Create a Telegram Bot (if needed)

#### Step 1: Find BotFather

1. Open Telegram
2. Search for `@BotFather` (the official Telegram bot for creating bots)
3. Start a chat with BotFather

#### Step 2: Create New Bot

Send the following command to BotFather:

```
/newbot
```

#### Step 3: Configure Bot

1. **Choose a name** for your bot (e.g., "My AI Assistant")
2. **Choose a username** for your bot (must end with 'bot', e.g., "my_ai_assistant_bot")
3. **Copy the token** that BotFather provides - this is your `TELEGRAM_BOT_TOKEN`

Example conversation with BotFather:

```
You: /newbot
BotFather: Alright, a new bot. How are we going to call it? Please choose a name for your bot.
You: My AI Assistant
BotFather: Good. Now let's choose a username for your bot. It must end in `bot`.
You: my_ai_assistant_bot
BotFather: Done! Congratulations on your new bot. Use this token to access the HTTP API:
          [YOUR_BOT_TOKEN_HERE]
```

### 3. Get Your Telegram Chat ID

#### Method 1: Using the Bot

1. Start a chat with your new bot
2. Send any message to the bot
3. Check the bot logs - it will display your chat ID

#### Method 2: Using UserInfoBot

1. Search for `@userinfobot` in Telegram
2. Start the bot
3. It will automatically send you your chat ID

#### Method 3: Programmatically

Send a message to your bot and check the update in logs:

```python
# The chat ID will appear in logs like:
# "chat": {"id": 123456789, ...}
```

### 4. Configure Environment

Copy the example environment file and update with your values:

```bash
cp .env.example .env
```

Edit `.env` file:

```env
# Required: Your Telegram Bot Token from BotFather
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

# Optional: Your Telegram Chat ID for notifications
TELEGRAM_CHAT_ID=your_telegram_chat_id_here

# API Configuration (default should work for local development)
API_BASE_URL=http://localhost:8080
```

### 5. Start Required Services

Make sure the Go AI Assistant infrastructure is running:

```bash
# From the project root directory
docker-compose up -d  # Starts Redis and MongoDB

# Start the Go server (from project root)
go run ./cmd/server
```

### 6. Run the Bot

```bash
# Make sure you're in the python_telegram_bot directory
cd python_telegram_bot

# Activate virtual environment
source venv/bin/activate

# Run the bot
python telegram_bot_enhanced.py
```

You should see output like:

```
2025-11-05 15:20:12,964 - __main__ - INFO - Запуск Telegram бота для Go AI Assistant...
2025-11-05 15:20:13,189 - httpx - INFO - HTTP Request: POST https://api.telegram.org/bot.../getMe "HTTP/1.1 200 OK"
```

## Bot Commands

- `/start` - Welcome message and bot description
- `/status` - Check system status and connectivity
- `/weather <city>` - Get weather for specified city
- **Any message** - Send to AI assistant for response

## Example Usage

1. **Start the bot**: Send `/start` to see welcome message
2. **Check status**: Send `/status` to verify system connectivity
3. **Get weather**: Send `/weather Barcelona` for weather information
4. **Ask questions**: Send any message like "What's the time?" or "Tell me about machine learning"

## Project Structure

```
python_telegram_bot/
├── telegram_bot_enhanced.py  # Main bot code
├── requirements.txt          # Python dependencies
├── .env                     # Environment variables (not in git)
├── .env.example             # Example environment configuration
├── venv/                    # Python virtual environment
└── README.md               # This file
```

## Dependencies

- `python-telegram-bot==22.5` - Telegram Bot API library
- `requests==2.31.0` - HTTP requests for API calls
- `python-dotenv==1.0.0` - Environment variable management

## Troubleshooting

### Common Issues

1. **Bot not starting**: Check `TELEGRAM_BOT_TOKEN` is correct
2. **API connection errors**: Ensure Go server is running on `localhost:8080`
3. **Module not found**: Reactivate virtual environment and reinstall dependencies
4. **Bot not responding**: Check that Redis and MongoDB are running via `docker-compose up -d`

### Logs and Debugging

The bot provides detailed logs:

- Connection status with Telegram API
- API requests to Go server
- Conversation IDs and titles
- Error messages for troubleshooting

### Testing Connectivity

You can test the Go API directly:

```bash
curl -X POST http://localhost:8080/twirp/acai.chat.ChatService/StartConversation \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, are you working?"}' | jq
```

## Development

### Adding New Features

1. Add new command handlers in `telegram_bot_enhanced.py`
2. Update the Go AI Assistant system if new API endpoints are needed
3. Test locally before deployment

### Environment Variables

- `TELEGRAM_BOT_TOKEN`: **Required** - Your bot token from BotFather
- `TELEGRAM_CHAT_ID`: Optional - For specific user notifications
- `API_BASE_URL`: Optional - Defaults to `http://localhost:8080`

## Security Notes

- Keep your `TELEGRAM_BOT_TOKEN` secure and never commit it to version control
- The bot uses polling mode - for production consider using webhooks
- All API communication happens over HTTP (use HTTPS in production)

## Support

If you encounter issues:

1. Check the logs for error messages
2. Verify all services are running (Redis, MongoDB, Go server)
3. Ensure your Telegram bot token is valid
4. Check that your chat ID is correctly configured

## License

This project is part of the Go AI Assistant system.
