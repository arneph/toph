package imports

import (
	"os"
	"testing"

	"github.com/arneph/toph/builder/packages/internal/x/tools/testenv"
)

func TestMain(m *testing.M) {
	testenv.ExitIfSmallMachine()
	os.Exit(m.Run())
}
