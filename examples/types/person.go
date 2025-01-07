package types

// Address represents a physical address
type Address struct {
	Street string
	City   string
}

// Person represents a person with their details
type Person struct {
	Name    string
	Age     int
	Address Address
	job     string
}

// HasJob returns true if the person has a job
func (p *Person) HasJob() bool {
	return p.job != ""
}

// GetJob returns the person's job
func (p *Person) GetJob() string {
	return p.job
}
