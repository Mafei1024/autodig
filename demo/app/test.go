package app

type Object interface {
	Get()
	GetName() string
}

type Stu struct {
	Object
}

func (s *Stu) Get() {
}

func (s *Stu) GetName() string {
	return "stunihao"
}

// @autodig
func NewObject() Object {
	return &Stu{}
}
