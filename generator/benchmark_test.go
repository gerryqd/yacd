package generator

import (
	"testing"

	"github.com/gerryqd/yacd/types"
)

func BenchmarkGenerateCompilationDatabase(b *testing.B) {
	entries := make([]types.MakeLogEntry, 100)
	for i := range entries {
		entries[i] = types.MakeLogEntry{
			WorkingDir: "/project",
			Compiler:   "gcc",
			Args:       []string{"gcc", "-c", "-Wall", "-O2", "-DNDEBUG", "src/file.c", "-o", "build/file.o"},
			SourceFile: "src/file.c",
			OutputFile: "build/file.o",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateCompilationDatabase(entries, &types.ParseOptions{})
	}
}

func BenchmarkDeduplicateEntries(b *testing.B) {
	entries := make([]types.MakeLogEntry, 1000)
	for i := range entries {
		entries[i] = types.MakeLogEntry{
			WorkingDir: "/project",
			SourceFile: "src/file.c",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deduplicateEntries(entries)
	}
}
