package cfgldr

import (
	"errors"

	"github.com/mikeschinkel/go-cliutil"
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/go-dt/dtx"
)

type APIConfig interface {
	Config()
	IsNil() bool
}

var ErrLoadingFile = errors.New("error loading file")

func LoadAPIFileIfExists(apiFile dt.Filepath) (api APIConfig, err error) {
	var apiV1 *APIDescription
	var apiV2 *APIConfigV2
	var status dt.EntryStatus
	var target dt.Filepath

	// LoadJSON the APIConfig description if provided
	if apiFile == "" {
		goto end
	}
	// Ignoring error because status==dt.IsEntryError will catch it
	status, _ = apiFile.Status()
	switch status {
	case dt.IsFileEntry:
		// All good! Load it below
	case dt.IsSymlinkEntry:
		target, err = apiFile.Readlink()
		if err != nil {
			err = NewErr(dt.ErrFailedReadingSymlink, err)
			goto end
		}
		api, err = LoadAPIFileIfExists(target)
		if err == nil {
			apiV2, err = dtx.AssertType[*APIConfigV2](api)
			if err != nil {
				goto end
			}
		}
		goto end
	default:
		err = dtx.EntryStatusError(status)
		goto end
	}
	apiV2, err = LoadAPIConfigV2(apiFile)
	if err != nil {
		err = NewErr(
			dt.ErrFailedToLoadFile,
			"config_version", "v2",
			err,
		)
		goto end
	}
	if apiV2 != nil {
		goto end
	}
	apiV1, err = LoadAPIDescriptionFromFile(apiFile)
	if err != nil || apiV1 == nil {
		err = NewErr(
			dt.ErrFailedToLoadFile,
			"config_version", "v1",
			err,
		)
		goto end
	}
	apiV2 = apiV1.Migrate()
end:
	if apiV2 != nil {
		cliutil.Printf("APIConfig loaded successfully: %s (v%d)",
			apiV2.Name,
			apiV2.Version,
		)
	}
	if err != nil {
		err = WithErr(err,
			ErrLoadingFile,
			"api_file", apiFile,
		)
	}
	return apiV2, err
}
