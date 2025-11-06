package unit

import (
	"fmt"

	"github.com/8adimka/Go_AI_Assistant/internal/config"
)

func main() {
	cfg := config.Load()
	fmt.Printf("OpenAI Key: %s\n", cfg.OpenAIApiKey)
	fmt.Printf("Weather Key: %s\n", cfg.WeatherApiKey)
}
