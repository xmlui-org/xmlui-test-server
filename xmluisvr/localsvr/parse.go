package localsvr

import (
	"strconv"
	"time"

	"github.com/mikeschinkel/go-doterr"
	"github.com/mikeschinkel/go-dt"
)

func ParseHost(h string) (_ Host, err error) {
	if h == "" {
		err = ErrHostMustNotBeEmpty
	}
	// TODO Add some validation here
	return Host(h), err
}

type ZeroHandling int

const (
	UnspecifiedZeroHandling ZeroHandling = iota
	ZeroOk
	ZeroInvalid
	ZeroRequired
)

func ParseServerPort(p int, zh ZeroHandling) (sp ServerPort, err error) {
	switch zh {
	case ZeroOk:
		// S'all good, man
	case ZeroInvalid:
		if p == 0 {
			err = ErrPortMustNotBeZero
		}
	case ZeroRequired:
		// I cannot imagine this will ever be used, but here for symmetry
		if p != 0 {
			err = ErrPortMustBeZero
		}
	case UnspecifiedZeroHandling:
		fallthrough
	default:
		logger.Error("ParseServerPort: invalid zero handling value", "value", zh)
		goto end
	}
	// TODO Add some validation here
	sp = ServerPort(p)
end:
	return sp, err
}

func ParseFilepaths(files []string) (fps []dt.Filepath, _ error) {
	var errs []error
	fps = make([]dt.Filepath, 0, len(files))
	for _, file := range files {
		fp, err := dt.ParseFilepath(file)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		fps = append(fps, fp)
	}
	return fps, doterr.CombineErrs(errs)
}

func ParseConnectString(s string) (cs ConnectString, err error) {
	// TODO MAYBE add some validation here
	cs = ConnectString(s)
	return cs, err
}

func ParseDirPath(f string) (dp dt.DirPath, err error) {
	// TODO Add some validation here
	dp = dt.DirPath(f)
	return dp, err
}

// ParseTimeDurationEx parses a string as EITHER a Go duration format
// (like "3s", "10m", "1h30m") OR as an integer representing seconds.
func ParseTimeDurationEx(s string) (td time.Duration, err error) {
	var seconds int
	var errs []error

	// First try parsing as integer seconds
	seconds, err = strconv.Atoi(s)
	if err == nil {
		td = time.Duration(seconds) * time.Second
		goto end
	}
	errs = append(errs, err)

	// If that fails, try parsing as a standard Go duration
	td, err = time.ParseDuration(s)
	if err == nil {
		goto end
	}
	errs = append(errs, err)

end:
	return td, doterr.CombineErrs(errs)
}
