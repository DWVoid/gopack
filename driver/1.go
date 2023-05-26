package driver

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/DWVoid/calm"
)

type Options struct {
	Args    []string
	Version string
	Pseudo  bool
	Filter  func(path string) bool
}

type driver struct {
	name string
	desc string
	eval func(opt Options, flags *flag.FlagSet) string
}

var (
	mu sync.Mutex
	d  = make(map[string]driver)
)

func drvInst(drv driver) {
	mu.Lock()
	d[drv.name] = drv
	mu.Unlock()
}

func drvCrash(e calm.Error) {
	calm.WrapT(fmt.Fprintln(os.Stderr, calm.PrintCleans(e)))
	os.Exit(1)
}

func errLine(s string) { calm.WrapT(fmt.Fprintln(os.Stderr, s)) }

func DrvDesc() {
	errLine("Available drivers: ")
	mu.Lock()
	// get the max size of name to compute padding
	pad := 0
	for _, v := range d {
		if len(v.name) > pad {
			pad = len(v.name)
		}
	}
	eFmt := fmt.Sprintf("  %%-%ds%%s", pad+4)
	for _, v := range d {
		errLine(fmt.Sprintf(eFmt, v.name, v.desc))
	}
	os.Exit(2)
}

func DrvEval(opt Options) string {
	if len(opt.Args) > 0 {
		mu.Lock()
		if drv, ok := d[opt.Args[0]]; ok {
			mu.Unlock()
			return drv.eval(
				Options{Args: opt.Args[1:], Version: opt.Version, Pseudo: opt.Pseudo, Filter: opt.Filter},
				flag.NewFlagSet(fmt.Sprintf("gopack %s", drv.name), flag.ExitOnError),
			)
		}
		mu.Unlock()
		errLine("Unknown driver: " + opt.Args[0])
	} else {
		errLine("Unknown driver")
	}
	DrvDesc()
	return calm.Unreachable[string]()
}
