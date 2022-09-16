package config

import (
	"os"
	"testing"
)

type Config struct {
	Str string `yaml:"str"`
	Int int    `yaml:"int"`
}

func TestReadFromEnv(t *testing.T) {
	os.Setenv("STR", "temp")
	os.Setenv("INT", "1")

	var c Config
	ReadFromEnv(&c, nil)

	if c.Str != "temp" || c.Int != 1 {
		t.FailNow()
	}
}
