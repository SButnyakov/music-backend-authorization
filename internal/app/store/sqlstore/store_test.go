package sqlstore_test

import (
	"os"
	"testing"
)

var (
	databaseURL string
)

func TestMain(m *testing.M) {
	databaseURL = os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "jdbc:sqlserver://DESKTOP-IP5URND:1433;database=restapi_test;user=saaas;password=root;encrypt=true;trustServerCertificate=true;"
	}

	os.Exit(m.Run())
}
