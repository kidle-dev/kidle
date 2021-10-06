package version

var (
	Revision  string
	BuildUser string
	BuildDate string
	Branch    string
	Version   = "development"
)

type VersionInfos struct {
	Revision  string
	BuildUser string
	BuildDate string
	Branch    string
	Version   string
}

func GetVersionInfos() VersionInfos {
	return VersionInfos{
		Revision:  Revision,
		BuildUser: BuildUser,
		BuildDate: BuildDate,
		Branch:    Branch,
		Version:   Version,
	}
}
