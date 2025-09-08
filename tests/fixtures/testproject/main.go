package main

import "fmt"

// This is our target function that will be called by multiple places
func TargetFunction(x int) int {
	return x * 2
}

func main() {
	// Direct call from main
	result := TargetFunction(5)
	fmt.Println("Result from main:", result)
	
	// Call through another function
	processData()
	
	// Call through a struct method
	s := &Service{}
	s.Execute()
}

func processData() {
	// Direct call from processData
	val := TargetFunction(10)
	fmt.Println("Process data result:", val)
	
	// Nested call
	helperFunction()
}

func helperFunction() {
	// Direct call from helperFunction
	TargetFunction(15)
}

type Service struct{}

func (s *Service) Execute() {
	// Method calls TargetFunction
	TargetFunction(20)
	
	// Call another method that calls target
	s.internalProcess()
}

func (s *Service) internalProcess() {
	// Another method call
	TargetFunction(25)
}

// Anonymous function caller
func init() {
	go func() {
		TargetFunction(30)
	}()
}