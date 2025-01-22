package environments

import "expo-open-ota/config"

func ValidateEnvironment(environment string) bool {
	environments := config.GetEnvironmentsList()
	for _, env := range environments {
		if env == environment {
			return true
		}
	}
	return false
}
