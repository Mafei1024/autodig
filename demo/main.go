package main

import (
	"fmt"
	"github.com/Mafei1024/autodig/demo/app"
	"github.com/Mafei1024/autodig/dep"
)

func main() {
	if err := dep.Invoke(func(object app.Object) {
		fmt.Println(object.GetName())
	}); err != nil {
		panic(err)
	}
}
