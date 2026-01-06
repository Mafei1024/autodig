package dep

import "fmt"

type Cmd[T comparable] struct {
	Key      string
	Value    T
	Reminder string
}

type myFlag bool

func (f *myFlag) String() string {
	return fmt.Sprintf("%v", *f)
}

func (f *myFlag) Set(value string) error {
	// 只要调用Set方法，无论参数是什么，都设置为true
	*f = true
	return nil
}

func (f *myFlag) IsBoolFlag() bool {
	return true
}

func Reminder() {
	fmt.Println(fmt.Sprintf("\n-%s  %s\n\n-%s  %s\n\n-%s  %s\n",
		SameName.Key, SameName.Reminder,
		ScanDir.Key, ScanDir.Reminder,
		OutputFile.Key, OutputFile.Reminder))

}

var (
	ScanDir = &Cmd[string]{
		Key:      "scans",
		Reminder: "source code scan dirs, split with ',' [Example1: ./app2][Example2: ./app2/user,./app2/group]",
	}
	OutputFile = &Cmd[string]{
		Key:      "output",
		Reminder: "output file path  [Example: ./app2/entrypoint/autodig.go]",
	}
	SameName = &Cmd[myFlag]{
		Key:      "samename",
		Reminder: "support struct or func with the same name [Example: -samename]",
	}
	H = &Cmd[myFlag]{
		Key: "h",
	}
	Help = &Cmd[myFlag]{
		Key: "help",
	}
)
