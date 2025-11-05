package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/acai-travel/tech-challenge/internal/pb"
)

func main() {
	flag.Usage = func() {
		fmt.Printf("Usage: acai-cli [command] [options]\n")
		fmt.Println("Commands:")
		fmt.Println("  ask        Create a new conversation with assistant or continue an existing one")
		fmt.Println("  list       List existing conversations")
		fmt.Println("  show       Show conversation by ID")
	}

	if len(os.Args) < 2 {
		fmt.Println("Error: No command provided")
		fmt.Println("")
		flag.Usage()
		os.Exit(-1)
	}

	url := "http://localhost:8080"
	if v := os.Getenv("API_URL"); v != "" {
		url = v
	}

	cli := pb.NewChatServiceJSONClient(url, http.DefaultClient)
	ctx := context.Background()

	switch os.Args[1] {
	case "ask":
		fmt.Println("Press CMD+C to exit.")
		fmt.Println()

		cid := ""
		if len(os.Args) >= 3 {
			cid = os.Args[2]
			resp, err := cli.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: cid})

			if err != nil {
				fmt.Printf("Error describing conversation: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("ID:", resp.GetConversation().GetId())
			fmt.Println("Title:", resp.GetConversation().GetTitle())
			fmt.Println("Timestamp:", resp.GetConversation().GetTimestamp().AsTime().Format(time.RFC1123))
			fmt.Println("")
			for _, msg := range resp.GetConversation().GetMessages() {
				fmt.Printf("%s, %s:\n%s\n\n", msg.GetRole(), msg.GetTimestamp().AsTime().Format(time.TimeOnly), msg.GetContent())
			}
		} else {
			fmt.Println("Starting a new conversation, type your message below.")
			fmt.Println()
		}

		reader := bufio.NewReader(os.Stdin)

		for {
			fmt.Printf("USER:\n")
			line, _, err := reader.ReadLine()
			if err != nil {
				fmt.Printf("Error reading input: %v\n", err)
				os.Exit(1)
			}

			fmt.Println()

			if cid == "" {
				out, err := cli.StartConversation(ctx, &pb.StartConversationRequest{
					Message: string(line),
				})

				if err != nil {
					fmt.Printf("Error starting conversation: %v\n", err)
					os.Exit(1)
				}

				fmt.Println("New conversation started:")
				fmt.Println("ID:", out.GetConversationId())
				fmt.Println("Title:", out.GetTitle())
				fmt.Println()

				cid = out.GetConversationId()
				fmt.Printf("ASSISTANT:\n%s\n\n", out.GetReply())
				continue
			}

			out, err := cli.ContinueConversation(ctx, &pb.ContinueConversationRequest{
				ConversationId: cid,
				Message:        string(line),
			})

			if err != nil {
				fmt.Printf("Error continuing conversation: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("ASSISTANT:\n%s\n\n", out.GetReply())
		}

	case "list":
		resp, err := cli.ListConversations(ctx, &pb.ListConversationsRequest{})
		if err != nil {
			fmt.Printf("Error listing conversations: %v\n", err)
			os.Exit(1)
		}

		if len(resp.Conversations) == 0 {
			fmt.Println("No conversations found.")
			return
		}

		fmt.Println("ID                         TITLE")
		for _, conv := range resp.Conversations {
			fmt.Printf("%s   %s\n", conv.GetId(), conv.GetTitle())
		}
	case "show":
		if len(os.Args) < 3 {
			fmt.Println("Error: Conversation ID is required")
			os.Exit(1)
		}

		resp, err := cli.DescribeConversation(ctx, &pb.DescribeConversationRequest{
			ConversationId: os.Args[2],
		})

		if err != nil {
			fmt.Printf("Error describing conversation: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("ID:", resp.GetConversation().GetId())
		fmt.Println("Title:", resp.GetConversation().GetTitle())
		fmt.Println("Timestamp:", resp.GetConversation().GetTimestamp().AsTime().Format(time.RFC1123))
		fmt.Println("")
		for _, msg := range resp.GetConversation().GetMessages() {
			fmt.Printf("%s, %s:\n%s\n\n", msg.GetRole(), msg.GetTimestamp().AsTime().Format(time.TimeOnly), msg.GetContent())
		}
	}
}
