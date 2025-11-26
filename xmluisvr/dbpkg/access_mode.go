package dbpkg

import (
	"github.com/mikeschinkel/go-sqlparams"
)

// Constants for AccessMode values
// IMPORTANT: Number values of the AccessModes are critical to the algorithm
const (
	UnspecifiedAccessMode = AccessMode(sqlparams.UnspecifiedDBAccessMode)
	ReadOnlyMode          = AccessMode(sqlparams.DBReadOnlyMode)
	ReadWriteMode         = AccessMode(sqlparams.DBReadWriteMode)
	AdminMode             = AccessMode(sqlparams.DBAdminMode)
	SuperAdminMode        = AccessMode(sqlparams.DBSuperAdminMode)
)

type AccessMode int

func (m AccessMode) Name() string {
	switch m {
	case SuperAdminMode:
		return "Super Admin"
	case AdminMode:
		return "Admin"
	case ReadWriteMode:
		return "Read-Write"
	case ReadOnlyMode:
		return "Read-Only"
	case UnspecifiedAccessMode:
		fallthrough
	default:
		return "Unspecified"
	}
}

func ParseAccessMode(mode int) (am AccessMode, err error) {
	// TODO Validate
	return AccessMode(mode), err
}
