package hydra

import "github.com/fsnotify/fsnotify"

type NotifyFunc func(path string, op fsnotify.Op)
