package got

import (
	"github.com/gotvc/got/pkg/gdat"
	"github.com/gotvc/got/pkg/gotfs"
	"github.com/gotvc/got/pkg/gotrepo"
	"github.com/gotvc/got/pkg/gotvc"
)

type (
	Repo     = gotrepo.Repo
	Root     = gotfs.Root
	Ref      = gdat.Ref
	SnapInfo = gotvc.SnapInfo
	Snap     = gotvc.Snap
)

func InitRepo(p string) error {
	return gotrepo.Init(p)
}

func OpenRepo(p string) (*Repo, error) {
	return gotrepo.Open(p)
}
