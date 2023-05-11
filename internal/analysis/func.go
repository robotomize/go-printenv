package analysis

import (
	"errors"
	"os"
	"path/filepath"
)

func CurrentVendorRootPth(rootPth string) (string, error) {
	pth := filepath.Join(rootPth, "vendor")
	if _, err := os.Stat(pth); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			goPath := os.Getenv("GOPATH")
			pth = filepath.Join(append(filepath.SplitList(goPath), []string{"pkg", "mod"}...)...)
			if _, err = os.Stat(pth); err != nil {
				return "", ErrGoPkgNotFound
			}

			return pth, nil
		}

		return "", ErrGoModNotFound
	}

	return pth, nil
}
