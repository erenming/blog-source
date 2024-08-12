package pa

import "fmt"

var globalFlag bool

func (t *privateType) printGlobalFlag() {
	fmt.Println("[method.privateType] globalFlag: ", globalFlag)
}

type privateType struct {
	intField    int
	stringField string
	flag        bool
}

func GetPrivateType() *privateType {
	return &privateType{}
}
