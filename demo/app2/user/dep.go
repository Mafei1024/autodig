package group

import (
	"demo/app2/group"
	"fmt"
)

// @autodig
type My struct {
	DigReturn group.Me
}

func (m *My) Speak() {

}

// @autodig
type S1 struct {
	DigReturn group.Self
	Name      string `autodig:"-"` // 不会填充
}

// @autodig
type S2 struct {
	DigReturn group.Self
	Name      string `autodig:"-"`
}

// @autodig
func NewS1() group.Self {
	return &S1{
		Name: "S1我叫海伦",
	}
}

// @autodig
func NewS2() group.Self {
	return &S2{
		Name: "S2我叫凯乐",
	}
}

func (s *S1) Run() {
	if s.Name == "" {
		fmt.Println("name is nil S1")
	} else {
		fmt.Println("name is ", s.Name, ",S1")
	}

}

func (s *S2) Run() {
	if s.Name == "" {
		fmt.Println("name is nil S2")
	} else {
		fmt.Println("name is ", s.Name, ",S2")
	}
}

// @autodig
type B struct {
	S  group.Self `autodig:"name:S1"`
	SS []group.Self
}

type ControllerI interface {
}

// @autodig
type ControllerDemo1 struct {
	DigReturn ControllerI
}

// @autodig
type ControllerDemo2 struct {
	DigReturn ControllerI
}

// @autodig name:cd3
type ControllerDemo3 struct {
	DigReturn ControllerI
}

// @autodig
func NewControllerDemo4() ControllerI {
	return &ControllerDemo3{}
}

// @autodig name:ncd5
func NewControllerDemo5() ControllerI {
	return &ControllerDemo3{}
}

type GrpcClient struct {
}

// @autodig name:abGrpcClient
func NewAbGrpcClient() *GrpcClient {
	return &GrpcClient{}
}

// @autodig
func NewGrpcClient() *GrpcClient {
	return &GrpcClient{}
}

// @autodig
type Service struct {
	GrpcClient   *GrpcClient
	AbGrpcClient *GrpcClient `autodig:"name:abGrpcClient"`
}
