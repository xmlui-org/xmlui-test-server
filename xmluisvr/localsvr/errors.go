package localsvr

import (
	"errors"
)

var (
	ErrPathIsDir        = errors.New("path is a directory")
	ErrPathIsFile       = errors.New("path is a file (not a directory)")
	ErrFileDoesNotExist = errors.New("file does not exist")
	ErrFileExists       = errors.New("file exists")
)

var (
	ErrNoAPIProvided = errors.New("no api provided")
)

var (
	ErrURLPathMustNotBeEmpty        = errors.New("URL path must not be empty")
	ErrURLPathMustNotBeginWithSlash = errors.New("URL path must not begin with a slash ('/')")
	ErrInvalidURLPath               = errors.New("invalid URL path")
	ErrHostMustNotBeEmpty           = errors.New("host must not be empty")
	ErrPortMustNotBeZero            = errors.New("port must not be zero")
	ErrPortMustBeZero               = errors.New("port must be zero")
)
var (
	ErrValueDoesNotBeginWithAnIdentifier = errors.New("value can not be begin with an identifier")
	ErrInvalidIdentifier                 = errors.New("invalid identifier")
	ErrInvalidNameSpec                   = errors.New("invalid name spec")
	ErrMustBeginWithLetterOrUnderscore   = errors.New("must begin with letter or underscore")
)
