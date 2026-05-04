package migrate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitSQLStatements(t *testing.T) {
	in := "CREATE TABLE a (id INT);\nCREATE TABLE b (id INT);\n"
	out := splitSQLStatements(in)
	require.Len(t, out, 2)
	require.Contains(t, out[0], "CREATE TABLE a")
	require.Contains(t, out[1], "CREATE TABLE b")
}
