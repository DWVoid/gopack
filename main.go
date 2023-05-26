package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/DWVoid/calm"

	"github.com/DWVoid/gopack/driver"
	"github.com/DWVoid/gopack/utils"
)

func PutFile(remote, file string) calm.Result {
	return calm.Run(func() {
		file := calm.WrapT(os.Open(file)).Get()
		defer utils.SafeRun(file.Close)()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part := calm.WrapT(writer.CreateFormFile("file", filepath.Base(file.Name()))).Get()
		calm.WrapT(io.Copy(part, file)).Get()
		calm.Wrap(writer.Close())

		request := calm.WrapT(http.NewRequest("POST", remote, body)).Get()
		request.Header.Add("Content-Type", writer.FormDataContentType())
		client := &http.Client{}
		response := calm.WrapT(client.Do(request)).Get()
		if response.StatusCode != 200 {
			calm.ThrowDetail(calm.ERequest, response.Status, string(calm.WrapT(io.ReadAll(response.Body)).Get()))
		}
	})
}

func failIgnore(e calm.Error) {
	calm.WrapT(fmt.Fprintln(os.Stderr, calm.PrintDetails(e, calm.TrimShortPrint)))
}

func main() {
	fSet := flag.NewFlagSet("gopack", flag.ContinueOnError)
	zipPath := fSet.String("o", "", "(optional) location to store the package")
	srvPath := fSet.String("u", "", "(optional) hive server to push the package")
	version := fSet.String("v", "v0.0.0", "(optional) package version string")
	snapshot := fSet.Bool("s", false, "(optional) if a pseudo version should be appended")
	allFile := fSet.Bool("a", false, "(optional) if hidden files should be included")
	if fSet.Parse(os.Args[1:]) != nil {
		driver.DrvDesc()
	}
	if *version == "" {
		fSet.Usage()
		driver.DrvDesc()
	}
	if !calm.WrapT(regexp.Compile(`^v(\d+)\.(\d+)\.(\d+)(-\S+)?$`)).Get().MatchString(*version) {
		fmt.Printf("invalid package version: %s\n", *version)
		os.Exit(1)
	}
	opt := driver.Options{
		Args:    fSet.Args(),
		Version: *version,
		Pseudo:  *snapshot,
		Filter: func(path string) bool {
			return (!strings.Contains(path, "/.")) && (!strings.HasPrefix(path, "."))
		},
	}
	if *allFile {
		opt.Filter = func(x string) bool { return true }
	}
	tmp := driver.DrvEval(opt)
	if *srvPath != "" {
		fmt.Printf("uploading package to %s\n", *srvPath)
		PutFile(*srvPath, tmp).Unwrap(failIgnore)
	}
	if *zipPath != "" {
		fmt.Printf("storing package to %s\n", *zipPath)
		calm.Wrap(os.Rename(tmp, *zipPath)).Unwrap(failIgnore)
	}
	if _, e := os.Stat(tmp); e == nil {
		calm.Wrap(os.Remove(tmp)).Unwrap(failIgnore)
	}
}
