package mocks

type Mocks []Mock

func (m Mocks) Output(name string) {
}

func Generate(finder TypeFinder) Mocks {
	return nil
}
