package provisiond

import (
	"embed"
	"io/fs"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

//go:embed swagger
var static embed.FS

var swaggerFs fs.FS = &fsWithPrefix{
	FS:     static,
	prefix: "swagger",
}

type fsWithPrefix struct {
	fs.FS
	prefix string
}

func (f *fsWithPrefix) Open(name string) (fs.File, error) {
	newName := filepath.Join(f.prefix, name)
	file, err := f.FS.Open(newName)
	if err != nil {
		log.Error().Err(err).Msg("failed to open file")
	}
	return file, err
}
