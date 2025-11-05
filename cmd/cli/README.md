# CLI tool

This tool allows interacting with the application using command line interface.

You can run it from the root of the repository using:
```bash
$ go run ./cmd/cli
```

Available commands:
-  **ask** - Create a new conversation with assistant or continue an existing one
-  **list** - List existing conversations
-  **show** - Show conversation by ID

## Start a conversation

To start a conversation use `ask`:
```bash
$ go run ./cmd/cli ask
Press CMD+C to exit.

Starting a new conversation, type your message below.

USER:
<type your message here>
```

Wait for the assistant to respond, ask more questions, or exit the conversation by pressing `CMD+C` (or `CTRL+C` on
Windows/Linux).

## List conversations

To list existing conversations, use the `list` command:

```bash
$ go run ./cmd/cli list
ID                         TITLE
68a5aa7b14ba62ef8448c917   Today's date
68a5aa5714ba62ef8448c912   Weather in Barcelona
```

## View a conversation

To view a conversation by ID use the `show` command:
```bash
$ go run ./cmd/cli show 68a5aa7b14ba62ef8448c917
ID: 68a5aa7b14ba62ef8448c917
Title: Today's date
Timestamp: Wed, 20 Aug 2025 10:59:07 UTC

USER, 10:59:07:
What day is today?

ASSISTANT, 10:59:13:
Today is August 20, 2025.
```

You can also continue a conversation by ID using the `ask` command, with conversation ID as an argument.

```bash
$ go run ./cmd/cli ask 68a5aa7b14ba62ef8448c917 
Press CMD+C to exit.

ID: 68a5aa7b14ba62ef8448c917
Title: Today's date
Timestamp: Wed, 20 Aug 2025 10:59:07 UTC

USER, 10:59:07:
What day is today?

ASSISTANT, 10:59:13:
Today is August 20, 2025.

USER:
<type your message>
```
