package main

// Complex scenarios for testing

type ComplexService struct {
	data []int
}

// Method with pointer receiver
func (c *ComplexService) Process() {
	TargetFunction(300)
	c.chainCall()
}

// Method with value receiver
func (c ComplexService) ProcessValue() {
	TargetFunction(400)
}

// Chain of method calls
func (c *ComplexService) chainCall() {
	c.deeperCall()
}

func (c *ComplexService) deeperCall() {
	TargetFunction(500)
}

// Interface implementation
type Processor interface {
	DoWork()
}

type ConcreteProcessor struct{}

func (p *ConcreteProcessor) DoWork() {
	TargetFunction(600)
}

// Function that returns a function
func GetProcessor() func() {
	return func() {
		TargetFunction(700)
	}
}

// Variadic function caller
func VariadicCaller(nums ...int) {
	for _, n := range nums {
		TargetFunction(n)
	}
}

// Recursive caller (indirect)
func RecursiveCaller(n int) {
	if n > 0 {
		TargetFunction(n)
		RecursiveCaller(n - 1)
	}
}