package storage

import (
	"github.com/dacalin/aifactory/internal/app/project"
	"github.com/dacalin/aifactory/internal/app/task"
)

type Database interface {
	Close() error
	Projects() project.Repository
	Tasks() task.Repository
}
