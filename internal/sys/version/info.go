package version

import (
	"fmt"
	"time"
)

var (
	Tag    string = "v0.2.3"
	Commit string = "none"
	Date   string = "unknow"
)

type Info struct{}

func (i *Info) String() string {
	displayDate := Date
	if t, err := time.Parse("2006-01-02T15:04:05-0700", Date); err == nil {
		displayDate = t.Format("2006/01/02")
	} else if t, err := time.Parse(time.RFC3339, Date); err == nil {
		displayDate = t.Format("2006/01/02")
	}
	return fmt.Sprintf("sing-helm %s (%s) built at %s", Tag, Commit, displayDate)
}
