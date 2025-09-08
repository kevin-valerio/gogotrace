package main

// Another file with callers
func UtilityFunction() {
	// Call from utility
	TargetFunction(100)
	
	// Multiple calls in same function
	for i := 0; i < 3; i++ {
		TargetFunction(i)
	}
}

func AnotherHelper() {
	// Indirect call through utility
	UtilityFunction()
	
	// Direct call
	TargetFunction(200)
}