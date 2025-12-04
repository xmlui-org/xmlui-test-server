package cfgldr

import (
	"github.com/mikeschinkel/go-dt"
)

var workingDir dt.DirPath

func Initialize() (err error) {
	if workingDir != "" {
		goto end
	}
	workingDir, err = dt.Getwd()
end:
	return err
}

func WorkingDir() dt.DirPath {
	return workingDir
}
