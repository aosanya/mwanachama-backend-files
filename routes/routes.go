package routes

import (
	"github.com/aosanya/mwanachama-backend-shared/dispatch"
	"github.com/aosanya/mwanachama-backend-shared/httpwire"

	files "github.com/aosanya/mwanachama-backend-files"
)

type Route = httpwire.Route

type Mount = dispatch.Mount

var Sentinels = map[string]error{
	"ErrFolderNotFound": files.ErrFolderNotFound,
	"ErrFileNotFound":   files.ErrFileNotFound,
	"ErrInvalidFolder":  files.ErrInvalidFolder,
	"ErrInvalidFile":    files.ErrInvalidFile,
	"ErrInvalidVersion": files.ErrInvalidVersion,
	"ErrNoUploader":     files.ErrNoUploader,
}

var AnonymousActions = []string{}

var Table = dispatch.NewTable(files.Operations(), Sentinels, AnonymousActions...)

func Build(fm files.FileManager) ([]Route, error) { return Table.Build(fm, Mount{}) }

func BuildFor(fm files.FileManager, m Mount) ([]Route, error) { return Table.Build(fm, m) }

func RoutesFor(fm files.FileManager, m Mount) []Route { return Table.Routes(fm, m) }

func Split(fm files.FileManager, m Mount) dispatch.Split { return Table.Split(fm, m) }
