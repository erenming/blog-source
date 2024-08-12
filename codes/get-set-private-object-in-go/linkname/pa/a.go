package pa

import "fmt"

var globalFlag bool

func PrintGlobalFlag() {
	fmt.Println("[var] globalFlag: ", globalFlag)
}

func printGlobalFlag() {
	fmt.Println("[func] globalFlag: ", globalFlag)
}

func (t *ExportedType) printGlobalFlag() {
	fmt.Println("[method] globalFlag: ", globalFlag)
}

type ExportedType struct {
	intField    int
	stringField string
	flag        bool
}

func (t *ExportedType) String() string {
	return fmt.Sprintf("ExportedType{flag: %v}", t.flag)
}
