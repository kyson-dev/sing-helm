package version

import "fmt"

var (
	Tag    string = "dev"
	Commit string = "none"
	Date   string = "unknow"
)

type Info struct{}

func (i *Info) String() string {
	return fmt.Sprintf("sing-helm %s (%s) built at %s", Tag, Commit, Date)
}
