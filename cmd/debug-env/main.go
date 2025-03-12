package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Debugging environment variables:")
	fmt.Println("--------------------------------")

	// Check for R2 configuration variables
	checkEnv("BACKUP_DIR")
	checkEnv("BUCKET_NAME")
	checkEnv("BUCKET_ENDPOINT")
	checkEnv("ACCESS_KEY_ID")
	checkEnv("SECRET_ACCESS_KEY")
	checkEnv("USE_SSL")
	checkEnv("RETENTION_DAYS")
	checkEnv("UPLOAD_SCHEDULE")
	checkEnv("ENABLE_MONITORING")
	checkEnv("DISK_THRESHOLD")
	checkEnv("WEBHOOK_URLS")

	fmt.Println("--------------------------------")
	fmt.Println("All environment variables:")
	fmt.Println("--------------------------------")

	// Print all environment variables
	for _, env := range os.Environ() {
		fmt.Println(env)
	}
}

func checkEnv(name string) {
	value, exists := os.LookupEnv(name)
	if exists {
		// Mask sensitive values
		if name == "ACCESS_KEY_ID" {
			if len(value) > 6 {
				value = value[:3] + "..." + value[len(value)-3:]
			} else {
				value = "***"
			}
		} else if name == "SECRET_ACCESS_KEY" {
			value = fmt.Sprintf("[MASKED] (length: %d)", len(value))
		}

		fmt.Printf("%s: %s\n", name, value)
	} else {
		fmt.Printf("%s: [NOT SET]\n", name)
	}
}
