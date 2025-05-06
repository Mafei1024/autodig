package dep

import "go.uber.org/dig"

var container = dig.New()

// Provide help for provider
func Provide(cstors []interface{}, opts ...dig.ProvideOption) error {
	for _, cstor := range cstors {
		if err := container.Provide(cstor, opts...); err != nil {
			return err
		}
	}

	return nil
}

func MustProvide(cstors []interface{}, opts ...dig.ProvideOption) {
	if err := Provide(cstors, opts...); err != nil {
		panic(err)
	}
}

func Invoke(function interface{}, opts ...dig.InvokeOption) error {
	return container.Invoke(function, opts...)
}
