# Acai Technical Challenge

This technical challenge is part of the interview process for a Software Engineer position at [Acai Travel](https://acaitravel.com).
If you weren't sent here by one of our engineers, you can [get started here](https://www.acaitravel.com/about/careers).

We know you're eager to get to the code, but please read the instructions carefully before you begin.

The challenge might seem tricky at first, but once you get into it, we hope you'll enjoy the process and have fun
working with AI and Go.

## Introduction

In this challenge, you'll work on an existing application from this repository, written in [Go](https://go.dev). You can
make changes, add features, refactor existing code, etc. Think of it as if you've just joined a team and received a task
to improve an existing codebase.

You will be given a few specific [tasks to complete](#tasks), but feel free to do some housekeeping if you see something that
could be improved.

The application is a personal assistant service, which provides an API for conversations with an AI assistant. You could
say it's an API for an interface similar to ChatGPT: you have an endpoint to start a new conversation, an endpoint to
send a message to an existing conversation, a way to list conversations, and an endpoint to fetch a conversation by ID.

The assistant is built on top of [OpenAI's model](https://openai.com/), but it leverages
[additional tools](https://platform.openai.com/docs/guides/function-calling) and potentially some clever prompting to
provide a more useful experience.

Currently, the assistant can:

- Answer questions about the current date and time.
- Provide weather information (though it seems broken).
- Provide information about holidays in Barcelona.
- Provide general AI assistance.

## About the codebase

We expect you to be able to navigate and figure out the codebase on your own, but here are some key takeaways to give
you a boost:

- There is a `Makefile` with a few handy commands like `make up` and `make run`.
- The entry point to the application is in `cmd/server/main.go`, but the main logic lives in `internal/chat/server.go`.
- The application stores conversations in a [MongoDB](https://www.mongodb.com/) database. There's a docker compose file
  to start a local MongoDB instance.
- The application uses [Twirp](https://twitchtv.github.io/twirp/docs/intro.html) and [protobuf](https://protobuf.dev/)
  as a framework for the API. **You do NOT need to dig deep into Twirp and protobuf**. It's easy to use, provides JSON
  via HTTP endpoints, and "automagically" wires HTTP handlers and server implementation.
- The project uses code generation, but you should be able to complete the challenge without needing to run or
  understand it. In any case, do **not** make manual changes to the `internal/pb` package, maybe consider it a blackbox.

## General guidelines

1. **Do not fork this repository.** Instead, create a new repository in your own GitHub account and copy the contents of
   this repository into it. Forks are linked to the original repository, and we'd like to avoid candidates discovering
   each other's solutions. Keep your repository **public** so we can see your solution.
2. **Make use of git history.** It's easier for us to review your code if you commit your changes in meaningful chunks
   with clear descriptions.
3. **Use standard Go tools.** Use the tools shipped with the Go compiler, such as `go fmt`, `go test`, etc. Avoid
   unnecessary dependencies or tools. Keep it simple.
4. **Use Go conventions.** Follow Go conventions for naming, formatting, and structuring your code. Check the
   [Effective Go](https://go.dev/doc/effective_go) and [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
5. **Leave comments** where it makes sense. It helps whoever reads the code after you.
6. **You may use AI assistance/co-pilots**, but remember we are looking for a meaningful and maintainable codebase, not
   something slapped together quickly.

## Setting things up

You'll need:

- [Go](https://go.dev/doc/install) (use whatever version you have, or install the latest).
- [Docker](https://docs.docker.com/get-docker/) (to run the MongoDB container).
- The usual developer tools: git, make, etc.

Set up a repository:

1. Create a new repository in your GitHub account. Clone this repository, then copy everything except the `.git` folder
   into your own repo.
2. Commit the changes as **"Initial commit"** to set your starting point.

Start the application:

1. Set your OpenAI API key in the environment variable `OPENAI_API_KEY`.

   ```bash
   export OPENAI_API_KEY=your_openai_api_key
   ```

2. Use make to start MongoDB and the application. Make sure docker daemon is running.

   ```bash
   make up run
   ```

3. You should see `Starting the server...`, indicating the HTTP server is running at [localhost:8080](http://localhost:8080).
4. Use `command+C` to stop the server when you're done.
5. Use `make down` to stop the MongoDB container.

## Usage

> Before you interact with the application, make sure it's running, follow steps in the **Setting things up** section.

The application provides a simple HTTP-based API, you can interact with it using any HTTP client (like Postman, curl,
etc.) or use the [CLI tool](cmd/cli/README.md) provided in this repository.

### CLI tool

You can find [CLI tool](cmd/cli/README.md) in `cmd/cli` to interact with the application.

### HTTP API

We have created a [postman collection](https://documenter.getpostman.com/view/40257649/2sB3BKFo8S) for you to explore
the API. You can use [postman](https://www.postman.com/) or any other HTTP client.

## Testing

The codebase includes tests for the server and the assistant. The tests require mongoDB to be running, so make sure
to start it with `make up` before running the tests.

Run the tests using:

```bash
go test ./...
```

## Tasks

**You need to complete all the tasks** We would like you to spend at least 1 hour on the challenge.

### Task 1: Fix conversation title

> We recommend starting with this one. This task is relatively easy and requires you to debug the application, allowing you to get familiar with the codebase, and understand how the application works.

If you start a conversation, you'll notice the title does not really reflect the topic. Instead of summarizing your
question, it tries to answer it.

Your task is to fix the title generation logic so it summarizes the question instead of answering it. The system should
generate a concise title that reflects the main topic of the conversation.

For example, if you ask *"What is the weather like in Barcelona?"*, the title should be something like *"Weather in
Barcelona"*.

**Bonus:** Optimize performance for the `StartConversation` API to make it faster.

---

### Task 2: Fix the weather

The assistant is supposed to provide weather information, but currently it just says *"the weather is fine."* You need to connect it to a real weather API and return actual weather information (temperature, wind speed, conditions, etc.).

You can use any public weather API, e.g. [WeatherAPI](https://www.weatherapi.com/). This particular API is free to use,
but you need to sign up and get an API key.

**Bonus:** Enable the assistant to provide forecast information as well as current weather.

---

### Task 3: Refactor tools

The team is concerned that the way tools are currently defined in the codebase makes them difficult to maintain and extend. We're planning to add many more tools to give the assistant more capabilities, so we need a robust way to define and implement tools.

Refactor `internal/assistant/assistant.go` to make working with tools easier. Feel free to split things into files, introduce new package(s), or reorganize code as you see fit.

**Bonus:** Create a new tool of your choice.

---

### Task 4: Create a test for StartConversation API

The team wants a test for the `StartConversation` API to ensure it works as expected. Create an automated test in `internal/chat/server_test.go` to ensure the API:

- Creates new conversations.
- Populates the title.
- Triggers the assistant's response.
  
**Bonus:** Add tests for assistant's `Title` method in `internal/assistant/assistant.go`.

---

### Task 5: Instrument web server

The team wants better visibility into the performance of the web server. Add some basic metrics to track the number of requests, response times, and error rates.

Use [OpenTelemetry](https://opentelemetry.io/docs/languages/go/instrumentation/#metrics) to capture metrics for the number of requests and response times.

Keep the exporter and provider configuration simple—the key part is how you capture and configure specific metrics.

**Bonus:** Add tracing to the web server to track request flow through the application.
