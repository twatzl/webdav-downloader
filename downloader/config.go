package downloader

type Config struct {
	// http or https
	Protocol string
	// hostname
	Host string
	// base path for host (e.g. /remote.php/webdav)
	BaseDir string
	// working dir on the local machine
	LocalDir string
	// username
	User string
	// password
	Pass           string
	DeltaMode      bool
	DeltaFlags     map[string]bool
	InteraciveMode bool
}
