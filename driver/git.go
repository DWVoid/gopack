package driver

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/DWVoid/calm"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"

	"github.com/DWVoid/gopack/utils"
)

func gitOpenRepo(repoPath string) (*filesystem.Storage, *git.Repository) {
	fs := osfs.New(repoPath)
	calm.WrapT(fs.Stat(git.GitDirName)).Get()
	fs = calm.WrapT(fs.Chroot(git.GitDirName)).Get()
	s := filesystem.NewStorageWithOptions(fs, cache.NewObjectLRUDefault(), filesystem.Options{KeepDescriptors: true})
	r := calm.WrapT(git.Open(s, fs)).Get()
	return s, r
}

func gitLsFile(r *git.Repository, h *plumbing.Hash, treePath string) map[string]*object.File {
	commit := calm.WrapT(r.CommitObject(*h)).Get()
	tree := calm.WrapT(commit.Tree()).Get()
	if treePath != "" {
		tree = calm.WrapT(tree.Tree(treePath)).Get()
	}
	result := make(map[string]*object.File)
	calm.Wrap(tree.Files().ForEach(func(file *object.File) error {
		if file.Mode.IsFile() {
			result[file.Name] = file
		}
		return nil
	}))
	return result
}

func gitGetModName(files map[string]*object.File) string {
	goMod := files["go.mod"]
	for _, line := range calm.WrapT(goMod.Lines()).Get() {
		if after, found := strings.CutPrefix(strings.TrimSpace(line), "module"); found {
			return strings.TrimSpace(after)
		}
	}
	calm.ThrowClean(calm.ERequest, "invalid go.mod")
	return calm.Unreachable[string]()
}

func gitZipFiles(module string, version string, files map[string]*object.File, filter func(p string) bool) string {
	prefix := module + "@" + version + "/"
	archive := calm.WrapT(os.CreateTemp("", "*")).Get()
	defer utils.SafeRun(archive.Close)()
	zipWriter := zip.NewWriter(archive)
	defer utils.SafeRun(zipWriter.Close)()
	for entry, file := range files {
		if filter(entry) {
			w := calm.WrapT(zipWriter.Create(prefix + entry)).Get()
			r := calm.WrapT(file.Reader()).Get()
			calm.WrapT(io.Copy(w, r))
			calm.Wrap(r.Close())
		}
	}
	return archive.Name()
}

func gitPackage(repo, rev, tree, version string, snapshot bool, filter func(p string) bool) calm.ResultT[string] {
	return calm.RunT(func() string {
		s, r := gitOpenRepo(repo)
		defer utils.SafeRun(s.Close)()
		hash := calm.WrapT(r.ResolveRevision(plumbing.Revision(rev))).Get()
		files := gitLsFile(r, hash, tree)
		if snapshot {
			version = fmt.Sprintf(
				"%s-%s-%s",
				version, time.Now().Format("20060102150405"), hash.String()[:12],
			)
		}
		module := gitGetModName(files)
		fmt.Printf(
			"packaging git revision %s from \"%s>>%s\" as %s@%s\n",
			hash.String(), repo, tree, module, version,
		)
		return gitZipFiles(module, version, files, filter)
	})
}

func gitMain(opt Options, flags *flag.FlagSet) string {
	repo := flags.String("d", ".", "(optional) git repository directory")
	rev := flags.String("r", "HEAD", "(optional) revision in git repository")
	tree := flags.String("t", "", "(optional) tree in git repository")
	calm.Wrap(flags.Parse(opt.Args))
	return gitPackage(*repo, *rev, *tree, opt.Version, opt.Pseudo, opt.Filter).Unwrap(drvCrash)
}

func init() {
	drvInst(driver{
		name: "git",
		desc: "package from git driver repository",
		eval: gitMain,
	})
}
