package utils

func HandleRemove(remove func(string) error, fileName string) {
	err := remove(fileName)
	if err != nil {
		println("can't remove file")
	}
}

func HandleClose(close func() error) {
	err := close()
	if err != nil {
		println("can't close resource file")
	}
}
