package options

import "flag"

const (
	configFileName = "conf.yaml"
)

type options struct {
	CertDir              string
	Port                 int
	EnableLeaderElection bool
	ConfDir              string
	ConfFile             string
}

func NewOptions() *options {
	return &options{
		ConfFile: configFileName,
	}
}

var Options options

func init() {
	Options = *NewOptions()

	flag.IntVar(&Options.Port, "port", 8443, "Webhook server port.")
	flag.StringVar(&Options.CertDir, "certDir", "/etc/webhook/certs", "certDir is the directory that contains the server key and certificate.")
	flag.StringVar(&Options.ConfDir, "confDir", "/etc/webhook/conf", "the config file dir, config file name must be conf.yaml.")
	flag.BoolVar(&Options.EnableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
}
