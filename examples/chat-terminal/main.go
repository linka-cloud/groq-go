// Package main demonstrates how to use groq-go to create a chat application
// using the groq api accessable through the terminal.
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/conneroisu/groq-go"
	"github.com/conneroisu/groq-go/pkg/tools"
)

var (
	history = []groq.ChatCompletionMessage{}
)

func main() {
	if err := run(
		context.Background(),
		os.Stdin,
		os.Stdout,
	); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	r io.Reader,
	w io.Writer,
) error {
	key := os.Getenv("GROQ_KEY")
	client, err := groq.NewClient(key)
	if err != nil {
		return err
	}
	for {
		err = input(ctx, client, r, w)
		if err != nil {
			return err
		}
	}
}

func input(
	ctx context.Context,
	client *groq.Client,
	r io.Reader,
	w io.Writer,
) error {
	fmt.Println("")
	fmt.Print("->")
	reader := bufio.NewReader(r)
	writer := w
	var lines []string
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		if len(strings.TrimSpace(line)) == 0 {
			break
		}
		lines = append(lines, line)
		break
	}
	history = append(history, groq.ChatCompletionMessage{
		Role:    groq.RoleUser,
		Content: strings.Join(lines, "\n"),
	})
	reasoningEffort := groq.ReasoningEffortMedium
	output, err := client.ChatCompletionStream(
		ctx,
		groq.ChatCompletionRequest{
			Model:           groq.ModelOpenaiGptOss20B,
			Messages:        history,
			MaxTokens:       2000,
			ReasoningEffort: &reasoningEffort,
			Tools:           []tools.Tool{tools.BuiltIn.BrowserSearch},
		},
	)
	if err != nil {
		return err
	}
	var reasoning bool
	for {
		response, err := output.Recv()
		if err != nil {
			return err
		}
		for _, v := range response.Choices[0].Delta.ExecutedTools {
			if v.Output == nil {
				continue
			}
			fmt.Fprintf(writer, "\n%s:\n", v.Type)
			fmt.Fprintf(writer, "  input: %s\n", string(v.Arguments))
			fmt.Fprintf(writer, "  output: %s\n\n", *v.Output)
		}
		if response.Choices[0].FinishReason == groq.ReasonStop {
			fmt.Fprintln(writer)
			break
		}
		if !reasoning && response.Choices[0].Delta.Reasoning != "" {
			fmt.Fprintln(writer, "\nreasoning: ")
			reasoning = true
		}
		fmt.Fprint(writer, response.Choices[0].Delta.Reasoning)
		if reasoning && response.Choices[0].Delta.Content != "" {
			reasoning = false
			fmt.Fprintln(writer, "\n\nresponse: ")
		}
		fmt.Fprint(writer, response.Choices[0].Delta.Content)
	}
	return nil
}
