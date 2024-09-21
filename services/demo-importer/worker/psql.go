package worker

import (
	"fmt"
	"github.com/kaytu-io/open-governance/services/demo-importer/types"
	"os"
	"os/exec"
)

func ImportPsqlData(cnf types.DemoImporterConfig, dataPath string) {
	databases := map[string]string{
		"pennywise":  dataPath + "/pennywise.sql",
		"workspace":  dataPath + "/workspace.sql",
		"auth":       dataPath + "/auth.sql",
		"migrator":   dataPath + "/migrator.sql",
		"describe":   dataPath + "/describe.sql",
		"onboard":    dataPath + "/onboard.sql",
		"inventory":  dataPath + "/inventory.sql",
		"compliance": dataPath + "/compliance.sql",
		"metadata":   dataPath + "/metadata.sql",
	}

	for dbName, sqlFilePath := range databases {
		fmt.Println("Importing database " + dbName)
		err := runPsqlCommand(cnf, dbName, sqlFilePath)
		if err != nil {
			fmt.Printf("Failed to import data for %s: %v\n", dbName, err)
		} else {
			fmt.Printf("Successfully imported data for %s\n", dbName)
		}
	}
}

func runPsqlCommand(cnf types.DemoImporterConfig, dbName, sqlFilePath string) error {
	postgresPassword := cnf.PostgreSQL.Password
	postgresHost := cnf.PostgreSQL.Host
	postgresPort := cnf.PostgreSQL.Port
	postgresUser := cnf.PostgreSQL.Username

	psqlPath, err := exec.LookPath("psql")
	if err != nil {
		return fmt.Errorf("psql not found in PATH: %v", err)
	}

	cmd := exec.Command(psqlPath,
		"--host="+postgresHost,
		"--port="+postgresPort,
		"--username="+postgresUser,
		"--dbname="+dbName,
		"-f", sqlFilePath)

	cmd.Env = append(os.Environ(), "PGPASSWORD="+postgresPassword)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running psql for database %s: %v, output: %s", dbName, err, string(output))
	}
	return nil
}
