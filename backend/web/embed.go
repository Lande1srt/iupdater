package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

func GetDistFS() (fs.FS, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	return sub, nil
}
