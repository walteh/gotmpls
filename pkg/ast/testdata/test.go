package test

type Address struct {
	Street string
	City   string
}

type Person struct {
	Name    string
	Address Address
}

func (p *Person) GetJob() string {
	return "Developer"
}

func (p *Person) GetAddress() *Address {
	return &p.Address
}
