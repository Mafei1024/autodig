package dep

import (
	"go.uber.org/dig"
)

var container = dig.New()

// provide help for provider
func provide(cstors []interface{}, opts ...dig.ProvideOption) error {
	for _, cstor := range cstors {
		if err := container.Provide(cstor, opts...); err != nil {
			return err
		}
	}
	return nil
}

func MustProvide(cstors []interface{}, opts ...dig.ProvideOption) {
	if err := provide(cstors, opts...); err != nil {
		panic(err)
	}
}

func Invoke(function interface{}, opts ...dig.InvokeOption) error {
	return container.Invoke(function, opts...)
}

func Instance[T any](opts ...dig.InvokeOption) (T, error) {
	var t T
	f := func(value T) error {
		t = value
		return nil
	}
	err := Invoke(f, opts...)
	return t, err
}
