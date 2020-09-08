package nameable

type Nameable interface {
	Matches(s string) bool
	String() string
}
