package extendtype


type Item interface {
	TypeName() string
}

type Rule interface {
	Check(Item) (bool,error)
}

type TypeRule struct {
	Rule
	Use bool
}

type ExtendType interface {
	CheckAll([]TypeRule,[]TypeRule) (bool,error)
}
