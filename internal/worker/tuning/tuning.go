package tuning

type Tuning struct {
	Vetu Vetu `yaml:"vetu"`
}

type Vetu struct {
	MTU int `yaml:"mtu"`
}

func (t *Tuning) GetVetu() Vetu {
	if t == nil {
		return Vetu{}
	}

	return t.Vetu
}
