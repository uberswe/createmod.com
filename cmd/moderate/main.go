package main

import (
	"createmod/internal/openai"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// SimpleLogger implements the openai.Logger interface for command-line use
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, args ...any) {
	fmt.Printf("[INFO] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Debug(msg string, args ...any) {
	fmt.Printf("[DEBUG] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Error(msg string, args ...any) {
	fmt.Printf("[ERROR] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func (l *SimpleLogger) Warn(msg string, args ...any) {
	fmt.Printf("[WARN] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func main() {
	// Load environment variables from .env file
	envFile, err := godotenv.Read(".env")
	if err != nil {
		log.Println("Warning: Could not load .env file:", err)
	}

	// Get OpenAI API key from environment
	apiKey := envFile["OPENAI_API_KEY"]
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if apiKey == "" {
		log.Fatal("Error: OPENAI_API_KEY is required. Set it in .env file or as an environment variable.")
	}

	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/moderate/main.go <image_path> [text]")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("  image_path  Path to local image file or URL to image")
		fmt.Println("  text        Optional text to moderate alongside the image")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run ./cmd/moderate/main.go ./my-image.jpg")
		fmt.Println("  go run ./cmd/moderate/main.go ./my-image.jpg \"Some text to check\"")
		fmt.Println("  go run ./cmd/moderate/main.go https://example.com/image.jpg \"Title: Test\"")
		os.Exit(1)
	}

	imagePath := os.Args[1]
	text := "Image moderation check"
	if len(os.Args) > 2 {
		text = os.Args[2]
	}

	// Create OpenAI client
	logger := &SimpleLogger{}
	client := openai.NewClient(apiKey, logger)

	fmt.Println("=== OpenAI Image Moderation ===")
	fmt.Println()
	fmt.Printf("Image: %s\n", imagePath)
	fmt.Printf("Text: %s\n", text)
	fmt.Println()

	// Determine if imagePath is a URL or local file
	var imageURL string
	if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
		// It's already a URL
		imageURL = imagePath
		fmt.Println("Using image URL directly")
	} else {
		// It's a local file, encode to base64
		fmt.Println("Encoding local image to base64...")
		base64Image, err := client.EncodeImageToBase64(imagePath)
		if err != nil {
			log.Fatalf("Error encoding image: %v", err)
		}

		// Detect image format from file extension
		format := "jpeg"
		lower := strings.ToLower(imagePath)
		if strings.HasSuffix(lower, ".png") {
			format = "png"
		} else if strings.HasSuffix(lower, ".gif") {
			format = "gif"
		} else if strings.HasSuffix(lower, ".webp") {
			format = "webp"
		}

		// Create data URL
		imageURL = fmt.Sprintf("data:image/%s;base64,%s", format, base64Image)
		fmt.Printf("Image encoded as data URL (%s format)\n", format)
	}

	fmt.Println()
	fmt.Println("Sending request to OpenAI moderation API...")
	fmt.Println()

	// Send moderation request
	response, err := client.ModerateTextAndImage(text, imageURL)
	if err != nil {
		log.Fatalf("Error moderating content: %v", err)
	}

	// Display results
	fmt.Println("=== Moderation Results ===")
	fmt.Println()
	fmt.Printf("Response ID: %s\n", response.ID)
	fmt.Printf("Model: %s\n", response.Model)
	fmt.Println()

	if len(response.Results) == 0 {
		fmt.Println("No results returned")
		return
	}

	for i, result := range response.Results {
		if len(response.Results) > 1 {
			fmt.Printf("--- Result %d ---\n", i+1)
		}

		// Overall status
		if result.Flagged {
			fmt.Println("Status: ⚠️  FLAGGED")
		} else {
			fmt.Println("Status: ✅ APPROVED")
		}
		fmt.Println()

		// Show flagged categories
		flaggedCategories := getFlaggedCategories(result)
		if len(flaggedCategories) > 0 {
			fmt.Println("Flagged Categories:")
			for _, category := range flaggedCategories {
				fmt.Printf("  - %s\n", category)
			}
			fmt.Println()
		}

		// Show category scores
		fmt.Println("Category Scores:")
		fmt.Printf("  Sexual:                 %.6f %s\n", result.CategoryScores.Sexual, flagEmoji(result.Categories.Sexual))
		fmt.Printf("  Sexual/Minors:          %.6f %s\n", result.CategoryScores.SexualMinors, flagEmoji(result.Categories.SexualMinors))
		fmt.Printf("  Hate:                   %.6f %s\n", result.CategoryScores.Hate, flagEmoji(result.Categories.Hate))
		fmt.Printf("  Hate/Threatening:       %.6f %s\n", result.CategoryScores.HateThreatening, flagEmoji(result.Categories.HateThreatening))
		fmt.Printf("  Harassment:             %.6f %s\n", result.CategoryScores.Harassment, flagEmoji(result.Categories.Harassment))
		fmt.Printf("  Harassment/Threatening: %.6f %s\n", result.CategoryScores.HarassmentThreatening, flagEmoji(result.Categories.HarassmentThreatening))
		fmt.Printf("  Self-Harm:              %.6f %s\n", result.CategoryScores.SelfHarm, flagEmoji(result.Categories.SelfHarm))
		fmt.Printf("  Self-Harm/Intent:       %.6f %s\n", result.CategoryScores.SelfHarmIntent, flagEmoji(result.Categories.SelfHarmIntent))
		fmt.Printf("  Self-Harm/Instructions: %.6f %s\n", result.CategoryScores.SelfHarmInstructions, flagEmoji(result.Categories.SelfHarmInstructions))
		fmt.Printf("  Violence:               %.6f %s\n", result.CategoryScores.Violence, flagEmoji(result.Categories.Violence))
		fmt.Printf("  Violence/Graphic:       %.6f %s\n", result.CategoryScores.ViolenceGraphic, flagEmoji(result.Categories.ViolenceGraphic))
		fmt.Println()
	}

	// Now check if the image shows a valid Minecraft build
	fmt.Println()
	fmt.Println("=== Minecraft Build Check ===")
	fmt.Println()
	fmt.Println("Checking if image shows a valid Minecraft build...")
	fmt.Println()

	isValidBuild, buildReason, err := client.CheckMinecraftBuildImage(imageURL)
	if err != nil {
		log.Printf("Warning: Failed to check Minecraft build: %v", err)
	} else {
		if isValidBuild {
			fmt.Println("Status: ✅ VALID MINECRAFT BUILD")
		} else {
			fmt.Println("Status: ⚠️  NOT A VALID BUILD")
			fmt.Printf("Reason: %s\n", buildReason)
		}
	}

	// Exit with appropriate code
	if response.IsFlagged() || !isValidBuild {
		os.Exit(1)
	}
}

// getFlaggedCategories returns a list of human-readable flagged categories
func getFlaggedCategories(result openai.ModerationResult) []string {
	var flagged []string
	if result.Categories.Sexual {
		flagged = append(flagged, "Sexual content")
	}
	if result.Categories.SexualMinors {
		flagged = append(flagged, "Sexual content involving minors")
	}
	if result.Categories.Hate {
		flagged = append(flagged, "Hate speech")
	}
	if result.Categories.HateThreatening {
		flagged = append(flagged, "Threatening hate speech")
	}
	if result.Categories.Harassment {
		flagged = append(flagged, "Harassment")
	}
	if result.Categories.HarassmentThreatening {
		flagged = append(flagged, "Threatening harassment")
	}
	if result.Categories.SelfHarm {
		flagged = append(flagged, "Self-harm")
	}
	if result.Categories.SelfHarmIntent {
		flagged = append(flagged, "Self-harm intent")
	}
	if result.Categories.SelfHarmInstructions {
		flagged = append(flagged, "Self-harm instructions")
	}
	if result.Categories.Violence {
		flagged = append(flagged, "Violence")
	}
	if result.Categories.ViolenceGraphic {
		flagged = append(flagged, "Graphic violence")
	}
	return flagged
}

// flagEmoji returns a flag emoji if the category is flagged
func flagEmoji(flagged bool) string {
	if flagged {
		return "⚠️"
	}
	return ""
}
