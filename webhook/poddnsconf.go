package webhook

type DnsConfig struct {
	NameServers []string `yaml:"nameservers"`
	Searches []string `yaml:"searches"`
	Options []Option `yaml:"options"`

}

type Option struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
