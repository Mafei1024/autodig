# autodig - 自动生成依赖注入代码工具

autodig是基于go-ast的自动生成[dig](https://github.com/uber-go/dig)的依赖注入代码的工具。

## 与上一个版本的差异
#### 1、删除了tag标签的使用与扫描
#### 2、删了ingroup与outgroup
#### 3、保留了多个结构体使用name的区分方式
#### 4、在单个接口多个实现时，会默认去为所有实现的结构体写一个name，并把所有的实现集成到一个数组中
#### 5、在单个接口多个实现时，会默认去为所有实现的结构体生成一个以name为key类型string的Map
#### 6、接口只有单个实现时，不会再生成Map

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
(新增，还会输出一个接口数组。注：接口只有单个实现时加入name标签，数组就会失效报错)
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

func NewControllerDemoAll(param demo.ControllerI) ([]demo.ControllerI, error) {
    var results []demo.ControllerI = make([]demo.ControllerI, 0, 1)
    results = append(results, param)
    return results, nil
}
```
#### Struct:接口有多个实现时
接口有多个实现时，就会默认给每个实现增加一个name输入（name默认使用结构体名称）
方法体实现同理
####  注意事项：在接口都实现时，要避免同时在同包和异包下一起实现。下述的所有实现，和ControllerI接口要么都在同一个包下，要么都不在同一个包下

Source Code:
```golang
type ControllerI interface {
}

//@autodig
type ControllerDemo1 struct {
	DigReturn  ControllerI
	Service *Service
}
//@autodig
type ControllerDemo2 struct {
    DigReturn  ControllerI
    Service *Service
}
//@autodig name:cd3
type ControllerDemo3 struct {
    DigReturn  ControllerI
    Service *Service
}

//@autodig
func NewControllerDemo4() ControllerI {
    return &ControllerDemo3{}
}

//@autodig name:ncd5
func NewControllerDemo5() ControllerI {
    return &ControllerDemo3{}
}
```
Output:
```golang

func main_NewControllerDemo4() ControllerI {
    return NewControllerDemo4()
}
func main_NewControllerDemo5() ControllerI {
    return NewControllerDemo5()
}

func NewdemoControllerDemo1(Service *demo.Service) (demo.ControllerI, error) {
	var autoDigErr error
	controllerdemo := demo.ControllerDemo1{Service: Service, Return: nil}
	return &controllerdemo, autoDigErr
}

func NewdemoControllerDemo2(Service *demo.Service) (demo.ControllerI, error) {
    var autoDigErr error
    controllerdemo := demo.ControllerDemo2{Service: Service, Return: nil}
    return &controllerdemo, autoDigErr
}

func NewdemoControllerDemo3(Service *demo.Service) (demo.ControllerI, error) {
    var autoDigErr error
    controllerdemo := demo.ControllerDemo3{Service: Service, Return: nil}
    return &controllerdemo, autoDigErr
}

func GetSliceBy_ControllerDemo1_ControllerDemo2_cd3_NewControllerDemo4_ncd5(ControllerIParam0 struct {
    dig.In
    ControllerI `name:"ControllerDemo1"`
}, ControllerIParam1 struct {
    dig.In
    ControllerI `name:"ControllerDemo2"`
}, ControllerIParam2 struct {
    dig.In
    ControllerI `name:"cd3"`
}, ControllerIParam3 struct {
    dig.In
    ControllerI `name:"NewControllerDemo4"`
}, ControllerIParam4 struct {
    dig.In
    ControllerI `name:"ncd5"`
}) ([]ControllerI, error) {
    var results []ControllerI = make([]ControllerI, 0, 5)
    results = append(results, ControllerIParam0)
    results = append(results, ControllerIParam1)
    results = append(results, ControllerIParam2)
    results = append(results, ControllerIParam3)
    results = append(results, ControllerIParam4)
    return results, nil
}
func GetNameMapBy_ControllerDemo1_ControllerDemo2_cd3_NewControllerDemo4_ncd5(param0 struct {
    dig.In
    ControllerI `name:"ControllerDemo1"`
}, param1 struct {
    dig.In
    ControllerI `name:"ControllerDemo2"`
}, param2 struct {
    dig.In
    ControllerI `name:"cd3"`
}, param3 struct {
    dig.In
    ControllerI `name:"NewControllerDemo4"`
}, param4 struct {
    dig.In
    ControllerI `name:"ncd5"`
}) map[string]ControllerI {
    var results map[string]ControllerI = make(map[string]ControllerI)
    results["ControllerDemo1"] = param0
    results["ControllerDemo2"] = param1
    results["cd3"] = param2
    results["NewControllerDemo4"] = param3
    results["ncd5"] = param4
    return results
}
func init(){
    dep.MustProvide([]interface {
    }{main_NewControllerDemo4}, dig.Name("NewControllerDemo4"))
    dep.MustProvide([]interface {
    }{main_NewControllerDemo5}, dig.Name("ncd5"))
    dep.MustProvide([]interface {
    }{NewdemoControllerDemo3}, dig.Name("cd3"))
    dep.MustProvide([]interface {
    }{NewdemoControllerDemo2}, dig.Name("ControllerDemo2"))
    dep.MustProvide([]interface {
    }{NewdemoControllerDemo1}, dig.Name("ControllerDemo1"))	
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
#### 条件扫描(作废，因为与接口多实现冲突，已删除功能)

#### Struct:忽略字段
想忽略某些Public Field时，在后面加上tag```autodig:"-"``` e.g.
Source Code:
```
//@autodig
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