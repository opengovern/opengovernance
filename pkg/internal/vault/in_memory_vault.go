package vault

type InMemoryVaultSourceConfig struct {
	data map[string]map[string]any
}

func NewInMemoryVaultSourceConfig() *InMemoryVaultSourceConfig {
	return &InMemoryVaultSourceConfig{
		data: make(map[string]map[string]any),
	}
}

func (v *InMemoryVaultSourceConfig) Write(pathRef string, config map[string]any) (err error) {
	v.data[pathRef] = config
	return nil
}

func (v *InMemoryVaultSourceConfig) Read(pathRef string) (config map[string]any, err error) {
	return v.data[pathRef], nil
}

func (v *InMemoryVaultSourceConfig) Delete(pathRef string) error {
	delete(v.data, pathRef)
	return nil
}
