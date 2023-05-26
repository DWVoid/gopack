package driver

import (
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DWVoid/calm"

	"github.com/DWVoid/gopack/utils"
)

func fsGetModName(dir string) string {
	modFile := calm.WrapT(os.Open(dir + "/go.mod")).Get()
	modScan := bufio.NewScanner(modFile)
	modScan.Split(bufio.ScanLines)
	defer utils.SafeRun(modFile.Close)()
	for modScan.Scan() {
		line := modScan.Text()
		if after, found := strings.CutPrefix(strings.TrimSpace(line), "module"); found {
			return strings.TrimSpace(after)
		}
	}
	calm.ThrowClean(calm.ERequest, "invalid go.mod")
	return calm.Unreachable[string]()
}

func fsZipFiles(module, version, dir string, filter func(p string) bool) string {
	prefix := module + "@" + version + "/"
	archive := calm.WrapT(os.CreateTemp("", "*")).Get()
	defer utils.SafeRun(archive.Close)()
	zipWriter := zip.NewWriter(archive)
	defer utils.SafeRun(zipWriter.Close)()
	fmt.Printf("packaging plain directory from \"%s\" as %s@%s\n", dir, module, version)
	dir = calm.WrapT(filepath.Abs(dir)).Get()
	calm.Wrap(filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			after, _ := strings.CutPrefix(path, dir)
			after = strings.ReplaceAll(after, "\\", "/")
			after = strings.TrimLeft(after, "/")
			if filter(after) {
				w := calm.WrapT(zipWriter.Create(prefix + after)).Get()
				r := calm.WrapT(os.Open(path)).Get()
				calm.WrapT(io.Copy(w, r))
				calm.Wrap(r.Close())
			}
		}
		return nil
	})).Get()
	return archive.Name()
}

func fsPackage(dir, version string, snapshot bool, filter func(p string) bool) calm.ResultT[string] {
	return calm.RunT(func() string {
		if snapshot {
			version = fmt.Sprintf(
				"%s-%s-%s",
				version, time.Now().Format("20060102150405"), "000000000000",
			)
		}
		return fsZipFiles(fsGetModName(dir), version, dir, filter)
	})
}

func fsMain(opt Options, flags *flag.FlagSet) string {
	path := flags.String("d", ".", "module directory")
	calm.Wrap(flags.Parse(opt.Args))
	return fsPackage(*path, opt.Version, opt.Pseudo, opt.Filter).Unwrap(drvCrash)
}

func init() {
	drvInst(driver{
		name: "fs",
		desc: "package file system",
		eval: fsMain,
	})
}
