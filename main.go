package main

import (
	"bufio" // For the Scanner helper
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/i2y/langchaingo-mcp-adapter"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/memory"

	"github.com/mark3labs/mcp-go/client" // New import for the client
)

// NewScanner creates a new Scanner from standard input.
type Scanner struct {
	scanner *bufio.Scanner
}

// NewScanner returns a new Scanner reading from standard input.
func NewScanner() *Scanner {
	return &Scanner{bufio.NewScanner(os.Stdin)}
}

// Scan reads a line from the standard input.
func (s *Scanner) Scan() string {
	s.scanner.Scan()
	return s.scanner.Text()
}

func main() {
	// --- MCP Client and Gemini Configuration ---
	// IMPORTANT: Set this environment variable before running:
	// export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
	//
	// You also need an MCP server executable in your path.
	// Replace with the actual path to your MCP server executable.
	mcpServerExecutablePath := "releasecontroller-mcp-server" // Path to your MCP server executable
	geminiModelName := "gemini-2.5-flash"                     // Use "gemini-1.5-pro", "gemini-flash", etc., as needed.

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("ERROR: GEMINI_API_KEY environment variable not set. Please set it to your actual Gemini API key.")
	}

	// Initialize the MCP client directly
	fmt.Printf("Attempting to initialize MCP client with executable at %s...\n", mcpServerExecutablePath)
	mcpClient, err := client.NewStdioMCPClient(
		mcpServerExecutablePath, // Path to an MCP server executable
		nil,                     // Additional environment variables (e.g., {"DEBUG": "true"})
	)
	if err != nil {
		log.Fatalf("ERROR: Failed to create MCP client: %v. Ensure '%s' is a valid executable.", err, mcpServerExecutablePath)
	}
	defer mcpClient.Close() // Ensure the client is closed when main exits

	fmt.Println("Successfully initialized MCP client.")

	// Use the explicit MCP client to create the main LLM adapter
	// Create the adapter
	adapter, err := langchaingo_mcp_adapter.New(mcpClient)
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}

	// Get all tools from MCP server
	mcpTools, err := adapter.Tools()
	if err != nil {
		log.Fatalf("Failed to get tools: %v", err)
	}

	fmt.Println("Successfully created main agent LLM adapter.")

	ctx := context.Background()

	// Create a Google AI LLM client
	llm, err := googleai.New(
		ctx,
		googleai.WithDefaultModel(geminiModelName),
		googleai.WithAPIKey(apiKey),
	)
	if err != nil {
		log.Fatalf("Create Google AI client: %v", err)
	}

	convMem := memory.NewConversationWindowBuffer(6)

	// 3. Initialize the NewConversationalAgent
	// This agent is designed for conversational interactions and uses the provided memory.
	agent := agents.NewConversationalAgent(
		llm,      // The primary LLM for generating conversational responses
		mcpTools, // The tools the agent can use
	)

	executor := agents.NewExecutor(agent, agents.WithMemory(convMem), agents.WithMaxIterations(20))

	fmt.Println("Starting conversational agent. Type 'exit' to quit.")
	fmt.Println("This agent requires an MCP server executable at the specified path and a set GEMINI_API_KEY environment variable.")

	// Simulate a conversation
	scanner := NewScanner() // Helper for reading user input from console

	for {
		fmt.Print("\nYou: ")
		input := scanner.Scan()
		if input == "exit" {
			fmt.Println("Agent: Goodbye!")
			break
		}

		// Invoke the agent with the user's input
		result, err := chains.Run(
			ctx,
			executor,
			input,
		)
		if err != nil {
			if strings.HasPrefix(err.Error(), agents.ErrUnableToParseOutput.Error()) {
				result = strings.TrimPrefix(err.Error(), agents.ErrUnableToParseOutput.Error()+": ")
				fmt.Printf("Agent: %s\n", result)
			} else {
				fmt.Printf("Agent: %v\n", err)
			}
		} else {
			fmt.Printf("Agent: %s\n", result)
		}
	}
}
