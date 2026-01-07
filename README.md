# autodig - 自动生成依赖注入代码工具

autodig是基于go-ast的自动生成[dig](https://github.com/uber-go/dig)的依赖注入代码的工具。

## 与上一个版本的差异
##### 新增命令 -samename -h （-h 查找命令指南）
#### 1、删除了tag标签的使用与扫描
#### 2、删了ingroup与outgroup（兼容outgroup，会生成以不同name去兼容）
#### 3、保留了多个结构体使用name的区分方式
#### 4、在单个接口多个实现时，会默认去为所有实现的结构体写一个name，并把所有的实现集成到一个数组中
#### 5、在单个接口多个实现时，会默认去为所有实现的结构体生成一个以name为key类型string的Map
#### 6、接口只有单个实现时，不会再生成Map
#### 7、增加同名struct支持需要 -samename支持（默认不支持同名struct）

## 基础用法

```go get github.com/Mafei1024/autodig```

```go install github.com/Mafei1024/autodig```

``` $(GOPATH)/bin/autodig -scans ./app -output ./app```

``` 注：-output ./app  可以自定义去生成文件名称 autodig_gen.go,-output ./app/autodig_gen.go ```

会在./app下生成autodig.go文件，里面包含所有./app下标记了 ```//@autodig```的方法/类的相关依赖注入代码

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
## 使用示例
go
```
// @autodig name:Demo1 (注：name按需使用)
type Demo struct{
}
// @autodig name:Demo1 (注：name按需使用)
func NewDemo()*Demo{
}
```
按照 [代码参考](https://github.com/Mafei1024/autodig/tree/v1.5.9/demo) 进行操作。
