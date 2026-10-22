package files

import _ "embed"

//go:embed files.operations.json
var operationsJSON []byte

func Operations() []byte { return operationsJSON }
