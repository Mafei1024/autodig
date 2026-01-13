# autodig - Automated Dependency Injection Code Generation Tool

autodig is a tool based on go-ast for automatically generating dependency injection code for [dig](https://github.com/uber-go/dig).

## Differences from the Previous Version
##### New command: `-samename -h` (`-h` for command help)
#### 1. Removed the use and scanning of tag labels.
#### 2. Removed `ingroup` and `outgroup` (compatible with `outgroup`, it will generate with different names for compatibility).
#### 3. Retained the method of using `name` to distinguish between multiple structs.
#### 4. When a single interface has multiple implementations, it will automatically assign a `name` to all implementing structs and aggregate all implementations into an array.
#### 5. When a single interface has multiple implementations, it will automatically generate a `Map` with a string `name` as the key for all implementing structs.
#### 6. When an interface has only a single implementation, a `Map` will **not** be generated.
#### 7. Added support for structs with the same name, requires the `-samename` flag (disabled by default).

## Basic Usage

```bash
go get github.com/Mafei1024/autodig
```

```bash
go install github.com/Mafei1024/autodig
```

```bash
$(GOPATH)/bin/autodig -scans ./app -output ./app
```

```Note: -output ./app   allows you to customize the generated file name, e.g., autodig_gen.go: -output ./app/autodig_gen.go```

This will generate an `autodig.go` file under `./app`, containing all dependency injection code related to methods/classes marked with ```//@autodig``` within the `./app` directory.

## Command Line Arguments
```
Usage of autodig:
  -output string
        output file path (default "./app/entrypoint/autodig.go")
  -scans string
        source code scan dirs, split with ',' (default "./app")
```
If no arguments are provided, it defaults to scanning `./app` and generating the file at `./app/entrypoint/autodig.go`.

## Integration Method
Add a `dig` command to your `makefile`. Running `make dig` will execute it. You can optionally add ```@make dig``` under the `build` command to have it run automatically before each deployment.
```makefile
GOBINVAR=${GOBIN}
dig:
	autodig
	$(GOBINVAR)/autodig
```
## Usage Example
Go code:
```go
// @autodig name:Demo1 (Note: Use `name` as needed)
type Demo struct{
	D2 *Demo2
}
// @autodig name:Demo1 (Note: Use `name` as needed)
func NewDemo()*Demo{
}
// @autodig
type Demo2 struct{
}
func main(){
	d,err:=dep.Invoke[*Demo]()
}
```
Refer to the [code example](https://github.com/Mafei1024/autodig/tree/feature/up_2.2/demo) for operation.
