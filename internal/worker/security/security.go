package security

type Security struct {
	AllowedIsolations *AllowedIsolations `yaml:"allowed-isolations"`
}

type AllowedIsolations struct {
	None      *IsolationPolicyNone      `yaml:"none"`
	Container *IsolationPolicyContainer `yaml:"container"`
	Parallels *IsolationPolicyParallels `yaml:"parallels"`
	Tart      *IsolationPolicyTart      `yaml:"tart"`
}

func NoSecurity() *Security {
	return &Security{}
}

func (security *Security) NonePolicy() *IsolationPolicyNone {
	if isolation := security.AllowedIsolations; isolation != nil {
		return isolation.None
	}

	return &IsolationPolicyNone{}
}

func (security *Security) ContainerPolicy() *IsolationPolicyContainer {
	if isolation := security.AllowedIsolations; isolation != nil {
		return isolation.Container
	}

	return &IsolationPolicyContainer{}
}

func (security *Security) ParallelsPolicy() *IsolationPolicyParallels {
	if isolation := security.AllowedIsolations; isolation != nil {
		return isolation.Parallels
	}

	return &IsolationPolicyParallels{}
}

func (security *Security) TartPolicy() *IsolationPolicyTart {
	if isolation := security.AllowedIsolations; isolation != nil {
		return isolation.Tart
	}

	return &IsolationPolicyTart{}
}
