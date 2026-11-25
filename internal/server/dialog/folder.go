package dialog

import (
	"context"
)

type FolderPicker interface {
	SelectFolder(context.Context, string) (string, error)
}
