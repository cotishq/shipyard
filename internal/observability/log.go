package observability

import (
	"encoding/json"
	"log"
	"time"
)

func Info(message string, fields map[string]any) {
	write("info", message, fields)
}

func Error(message string, fields map[string]any) {
	write("error", message, fields)
}

func write(level, message string, fields map[string]any) {
	payload := map[string]any{
		"time":  time.Now().UTC().Format(time.RFC3339),
		"level": level,
		"msg":   message,
	}

	for key, value := range fields {
		payload[key] = value
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		log.Printf(`{"time":"%s","level":"error","msg":"failed to marshal log payload","error":%q}`, time.Now().UTC().Format(time.RFC3339), err.Error())
		return
	}

	log.Println(string(encoded))
}
