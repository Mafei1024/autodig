# autodig - 自动生成依赖注入代码工具

autodig是基于go-ast的自动生成[dig](https://github.com/uber-go/dig)的依赖注入代码的工具。

## 基础用法

```go get github.com/Mafei1024/autodig```

```go install github.com/Mafei1024/autodig```

``` $(GOPATH)/bin/autodig -scans ./app -output ./app```
``` -output ./app  可以自定义去生成文件名称 autodig_gen.go,-output ./app/autodig_gen.go ```
会在./app下生成autodig.go文件，里面包含所有./app下标记了 ```//@autodig```的方法/类的相关依赖注入代码

Source Code:
``` golang
package demo
//@autodig
type Service struct {
	Logger string //public字段自动注入
	config string //private字段会被忽略
}

//@autodig
func NewLogger() Logger {
	return Logger{}
}
```

Output:
``` golang
func NewdemoService(Logger string) (*demo.Service, error) {
	var autoDigErr error
	service := demo.Service{Logger: Logger}
	return &service, autoDigErr
}
func demo_NewLogger() demo.Logger {
	return demo.NewLogger()
}
func init() {
	dep.MustProvide([]interface {
	}{NewdemoService, demo_NewLogger})
}
```

## 命令行参数
```
Usage of autodig:
  -output string
        output file path (default "./app/entrypoint/autodig.go")
  -scans string
        source code scan dirs, split with ',' (default "./app")
  -tag string
        tag, only support one, e.g.mock will only generate `//@autodig` or `//@autodig tag:mock` funcs/structs
```
不传参数默认扫描./app，生成文件为./app/entrypoint/autodig.go

## 接入方式
makefile中增加dig指令，make dig可以执行，可选择在build指令下增加```@make dig```，让每次部署前自动执行
```
GOBINVAR=${GOBIN}
dig:
	autodig
	$(GOBINVAR)/autodig
```
有需要条件注入同学，可参考最下方条件注入，通过参数控制

## 其他功能
#### struct: 初始化
支持在生成struct后自动执行它的init()error方法，用于进行一些初始化工作。e.g.
Source Code:
```golang

//@autodig
type Service struct {
	Logger string
	config string
}

func (s *Service) Init() error {
	s.config = "{\"Debug\":true}"
	return nil
}
```
Output:
```golang
func NewdemoService(Logger string) (*demo.Service, error) {
	var autoDigErr error
	service := demo.Service{Logger: Logger}
	autoDigErr = service.Init() //call init function
	return &service, autoDigErr
}
```
#### Struct:注入其他类型
struct默认是注入*Struct，可以通过DigReturn字段指定其他类型。e.g.
Source Code:
```golang
type ControllerI interface {
}

//@autodig
type ControllerDemo struct {
	DigReturn  ControllerI
	Service *Service
}
```
Output:
```golang
func NewdemoControllerDemo(Service *demo.Service) (demo.ControllerI, error) {
	var autoDigErr error
	controllerdemo := demo.ControllerDemo{Service: Service, Return: nil}
	return &controllerdemo, autoDigErr
}
```
#### group
通过在注释上增加 outgroup:组名 即可指定注入到某个group。 e.g.
Source Code:
```golang
//@autodig outgroup:restControllers
type ControllerDemo struct {
	DigReturn  ControllerI
	Service *Service
}

//@autodig outgroup:loggers
func NewLogger() Logger {
	return Logger{}
}
```
当需要依赖注入中某个group的所有对象时，给对应field(必须是array)加上tag```autodig:"ingroup:组名"```即可。e.g.
Source Code:
```golang
//@autodig
type Service struct {
	Loggers []Logger `autodig:"ingroup:loggers"`
	config string
}
```
Output:
```golang
func NewdemoControllerDemo(Service *demo.Service) (demo.ControllerI, error) {
	var autoDigErr error
	controllerdemo := demo.ControllerDemo{Service: Service, Return: nil}
	return &controllerdemo, autoDigErr
}
func NewdemoService(demoServiceParam struct {
	dig.In
	Loggers []demo.Logger `group:"loggers"`
}) (*demo.Service, error) {
	var autoDigErr error
	service := demo.Service{Loggers: demoServiceParam.Loggers}
	return &service, autoDigErr
}
func demo_NewLogger() demo.Logger {
	return demo.NewLogger()
}
func init() {
	dep.MustProvide([]interface {
	}{NewdemoControllerDemo}, dig.Group("restControllers"))
	dep.MustProvide([]interface {
	}{demo_NewLogger}, dig.Group("loggers"))
	dep.MustProvide([]interface {
	}{NewdemoService})
}
```
#### name
当需要注入多个一样的类时，可以通过指定name来区分。通过在注释/tag上增加 name:名字 即可指定。e.g.
Source Code:
```
//@autodig name:abGrpcClient
func NewAbGrpcClient() *GrpcClient {
	return &GrpcClient{}
}
//@autodig
func NewGrpcClient() *GrpcClient {
	return &GrpcClient{}
}
//@autodig
type Service struct {
	GrpcClient   *GrpcClient
	AbGrpcClient *GrpcClient `autodig:"name:abGrpcClient"`
}
```
Output:
```
func NewdemoService(GrpcClient *GrpcClient, demoServiceParam struct {
	dig.In
	AbGrpcClient *GrpcClient `name:"abGrpcClient"`
}) (*Service, error) {
	var autoDigErr error
	service := Service{GrpcClient: GrpcClient, AbGrpcClient: demoServiceParam.AbGrpcClient}
	return &service, autoDigErr
}

func init() {
	dep.MustProvide([]interface {
	}{demo_NewGrpcClient, NewdemoService})
	dep.MustProvide([]interface {
	}{demo_NewAbGrpcClient}, dig.Name("abGrpcClient"))
}

```
#### 条件扫描
给@autodig注释增加tag，可以通过命令行的tag指定条件扫描，目前命令行仅支持一个tag,仅支持非关系。```// @autodig tag:mock```

映射关系：

|cmd tag| valid source code|
|---|---|
|"mock"|"" "mock"|
|"!mock"|"" "!mock" "other"|
|""|"" "!mock"|

SourceCode:
```golang
//@autodig outgroup:restControllers tag:mock
type ControllerDemo struct {
	DigReturn  ControllerI
	Service *Service
}
//@autodig tag:!mock
type Service struct {
	Loggers []Logger `autodig:"ingroup:loggers"`
	config  string
}
```
OutPut: cmd empty tag, no Controller
```
import (
	demo "gitlab.p1staff.com/backend/tantan-backend-pcs/app/entrypoint/demo"
	dep "autodig/dep"
	dig "go.uber.org/dig"
)

func NewdemoService(demoServiceParam struct {
	dig.In
	Loggers []demo.Logger `group:"loggers"`
}) (*demo.Service, error) {
	var autoDigErr error
	service := demo.Service{Loggers: demoServiceParam.Loggers}
	return &service, autoDigErr
}
func init() {
	dep.MustProvide([]interface {
	}{NewdemoService})
}
```
#### Struct:忽略字段
想忽略某些Public Field时，在后面加上tag```autodig:"-"``` e.g.
Source Code:
```
//@autodig tag:!mock
type Service struct {
	Loggers []Logger `autodig:"-"`
	config  string
}
```
OutPut: Loggers be ignored
```
func NewdemoService() (*demo.Service, error) {
	var autoDigErr error
	service := demo.Service{}
	return &service, autoDigErr
}
```