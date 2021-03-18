package nameable

type SimpleNameable struct {
	name string
}

func NewSimpleNameable(name string) *SimpleNameable {
	return &SimpleNameable{name: name}
}

func (sn *SimpleNameable) Matches(s string) bool {
	return sn.name == s
}

func (sn *SimpleNameable) Name() string {
	return sn.name
}

func (sn *SimpleNameable) String() string {
	return sn.name
}

func (sn *SimpleNameable) MapKey() string {
	return "SimpleNameable(" + sn.name + ")"
}
