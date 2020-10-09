package boolevator

type Option func(*Boolevator)

func WithFunctions(functions map[string]Function) Option {
	return func(boolevator *Boolevator) {
		boolevator.functions = functions
	}
}
