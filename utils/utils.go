package utils

import "github.com/DWVoid/calm"

func SafeRun(fn func() error) func() { return func() { calm.Wrap(fn()) } }
