import logging
import os
import time
from typing import Optional

import requests
from dotenv import load_dotenv
from telegram import Update
from telegram.ext import (
    Application,
    CommandHandler,
    ContextTypes,
    MessageHandler,
    filters,
)

# Load environment variables from .env file
load_dotenv()

# Configure logging
logging.basicConfig(
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s", level=logging.INFO
)
logger = logging.getLogger(__name__)

# Base URL for our Go service API
API_BASE_URL = "http://localhost:8080"

# Get bot token from environment variables
TELEGRAM_BOT_TOKEN = os.getenv("TELEGRAM_BOT_TOKEN")

if not TELEGRAM_BOT_TOKEN:
    raise ValueError("TELEGRAM_BOT_TOKEN is not set in environment variables")

# Configuration
MAX_RETRIES = 3
RETRY_DELAY = 2
REQUEST_TIMEOUT = 30


class TelegramBotEnhanced:
    def __init__(self):
        self.application = None

    def _make_api_request(self, method: str, endpoint: str, **kwargs) -> Optional[dict]:
        """Makes API request with retry logic"""
        for attempt in range(MAX_RETRIES):
            try:
                url = f"{API_BASE_URL}{endpoint}"
                response = requests.request(
                    method, url, timeout=REQUEST_TIMEOUT, **kwargs
                )

                if response.status_code == 200:
                    return response.json()
                elif response.status_code >= 500:
                    logger.warning(
                        f"Server error {response.status_code}, attempt {attempt + 1}/{MAX_RETRIES}"
                    )
                else:
                    logger.error(f"API error {response.status_code}: {response.text}")
                    return None

            except requests.exceptions.ConnectionError:
                logger.warning(f"Connection error, attempt {attempt + 1}/{MAX_RETRIES}")
            except requests.exceptions.Timeout:
                logger.warning(f"Request timeout, attempt {attempt + 1}/{MAX_RETRIES}")
            except Exception as e:
                logger.error(f"Unexpected error during request: {e}")
                return None

            if attempt < MAX_RETRIES - 1:
                time.sleep(RETRY_DELAY)

        logger.error(f"Failed to complete request after {MAX_RETRIES} attempts")
        return None

    async def _send_typing_action(self, update: Update):
        """Sends typing indicator"""
        if not update.message:
            return False

        try:
            await update.message.chat.send_action(action="typing")
        except Exception as e:
            logger.warning(f"Failed to send typing indicator: {e}")

    async def start(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """Start interaction with the bot"""
        if not update.message:
            return

        await update.message.reply_text(
            "👋 Welcome to Go AI Assistant Bot!\n\n"
            "🤖 This bot is connected to our Go AI Assistant system with support for:\n"
            "• 💬 Smart responses from OpenAI GPT-4\n"
            "• 🌤️ Real-time weather via WeatherAPI\n"
            "• 📅 Holiday information\n"
            "• ⏰ Current date and time\n\n"
            "Just send any message and I'll help you!"
        )

    async def handle_message(
        self, update: Update, context: ContextTypes.DEFAULT_TYPE
    ) -> None:
        """Handles regular messages"""
        if not update.message:
            return

        message_text = update.message.text

        # Send typing indicator
        await self._send_typing_action(update)

        # Send request to our Go system via Twirp API
        payload = {"message": message_text}

        result = self._make_api_request(
            "POST",
            "/twirp/acai.chat.ChatService/StartConversation",
            json=payload,
            headers={"Content-Type": "application/json"},
        )

        if not result:
            await update.message.reply_text(
                "❌ Error connecting to AI service.\n"
                "Make sure the Go server is running on localhost:8080"
            )
            return

        if "reply" in result:
            reply = result["reply"]
            conversation_id = result.get("conversation_id", "N/A")
            title = result.get("title", "No title")

            # Format response
            response_text = f"🤖 {reply}"

            # Split long responses into parts
            if len(response_text) > 4000:
                for i in range(0, len(response_text), 4000):
                    await update.message.reply_text(response_text[i : i + 4000])
            else:
                await update.message.reply_text(response_text)

            # Log conversation information
            logger.info(f"Conversation ID: {conversation_id}, Title: {title}")

        else:
            error_msg = result.get("error", "Unknown error")
            await update.message.reply_text(f"❌ AI Error: {error_msg}")

    async def status(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """Shows current system status"""
        if not update.message:
            return

        # Check main API availability
        health_result = self._make_api_request("GET", "/")

        if health_result:
            status_message = (
                "📊 **Go AI Assistant System Status:**\n\n"
                "🟢 **Service:** Running\n"
                "🌐 **API:** Available on localhost:8080\n"
                "🤖 **AI:** OpenAI GPT-4\n"
                "🌤️ **Weather:** WeatherAPI.com\n"
                "💾 **Cache:** Redis\n"
                "🗄️ **Database:** MongoDB\n\n"
                "✅ System is ready to work!"
            )
        else:
            status_message = (
                "📊 **Go AI Assistant System Status:**\n\n"
                "🔴 **Service:** Unavailable\n"
                "❌ **API:** Connection error\n\n"
                "Make sure the Go server is running on localhost:8080"
            )

        await update.message.reply_text(status_message)

    async def weather(self, update: Update, context: ContextTypes.DEFAULT_TYPE) -> None:
        """Requests weather for specified location"""
        if not update.message:
            return

        # Extract location from command (e.g.: /weather Barcelona)
        if context.args:
            location = " ".join(context.args)
        else:
            await update.message.reply_text(
                "❌ Please specify a location.\nExample: /weather Barcelona"
            )
            return

        await self._send_typing_action(update)

        # Use our system to get weather
        payload = {"message": f"What is the weather like in {location}?"}

        result = self._make_api_request(
            "POST",
            "/twirp/acai.chat.ChatService/StartConversation",
            json=payload,
            headers={"Content-Type": "application/json"},
        )

        if result and "reply" in result:
            await update.message.reply_text(f"🌤️ {result['reply']}")
        else:
            await update.message.reply_text(f"❌ Failed to get weather for {location}")

    def setup_handlers(self):
        """Sets up command handlers"""
        if not self.application:
            logger.error("Application not initialized")
            return False

        # Command handlers
        self.application.add_handler(CommandHandler("start", self.start))
        self.application.add_handler(CommandHandler("status", self.status))
        self.application.add_handler(CommandHandler("weather", self.weather))

        # Handler for regular messages
        self.application.add_handler(
            MessageHandler(filters.TEXT & ~filters.COMMAND, self.handle_message)
        )

    def run(self):
        """Starts the bot"""
        if not TELEGRAM_BOT_TOKEN:
            logger.error("TELEGRAM_BOT_TOKEN is not set")
            return

        self.application = Application.builder().token(TELEGRAM_BOT_TOKEN).build()

        # Set up handlers
        self.setup_handlers()

        # Start the bot
        logger.info("Starting Telegram bot for Go AI Assistant...")
        self.application.run_polling()


def main():
    """Main function to start the bot"""
    bot = TelegramBotEnhanced()

    # Start the bot
    try:
        bot.run()
    except KeyboardInterrupt:
        print("\nBot stopped by user")
    except Exception as e:
        print(f"Error starting bot: {e}")


if __name__ == "__main__":
    main()
