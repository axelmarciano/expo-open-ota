package environments

func ValidateEnvironment(environment string) bool {
	return environment == "staging" || environment == "production" || environment == "demo"
}
