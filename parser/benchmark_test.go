package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gerryqd/yacd/types"
)

func BenchmarkParseMakeLog(b *testing.B) {
	makeLog := `make: Entering directory '/home/user/project'
gcc -c -Wall -O2 -DNDEBUG -DUSE_HAL_DRIVER -ICore/Inc -IDrivers/Include main.c -o main.o
gcc -c -Wall -O2 -DNDEBUG -DUSE_HAL_DRIVER -ICore/Inc -IDrivers/Include util.c -o util.o
gcc -c -Wall -O2 -DNDEBUG -DUSE_HAL_DRIVER -ICore/Inc -IDrivers/Include helper.c -o helper.o
cd subdir && gcc -c -Wall -O2 -DNDEBUG -DUSE_HAL_DRIVER -ICore/Inc driver.c -o driver.o
make: Leaving directory '/home/user/project'`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, _ := NewParser(types.ParseOptions{BaseDir: "/project"})
		p.ParseMakeLog(strings.NewReader(makeLog))
	}
}

func BenchmarkSplitCommandLine(b *testing.B) {
	line := "gcc -c -Wall -O2 -DNDEBUG -DUSE_HAL_DRIVER -ICore/Inc -IDrivers/Include main.c -o main.o"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		types.SplitCommandLine(line)
	}
}

func BenchmarkParseMakeLogLarge(b *testing.B) {
	// Simulate a larger project with 1000 compilation units
	var sb strings.Builder
	sb.WriteString("make: Entering directory '/home/user/project'\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("gcc -c -Wall -O2 -DNDEBUG -DUSE_HAL_DRIVER -ICore/Inc -IDrivers/Include src/file")
		sb.WriteString(strings.Repeat("0", 4-len(fmt.Sprintf("%d", i))))
		sb.WriteString(fmt.Sprintf("%d.c -o build/file%03d.o\n", i, i))
	}
	sb.WriteString("make: Leaving directory '/home/user/project'\n")
	makeLog := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, _ := NewParser(types.ParseOptions{BaseDir: "/project"})
		p.ParseMakeLog(strings.NewReader(makeLog))
	}
}
