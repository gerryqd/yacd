package types

import "testing"

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"Simple", "gcc -c main.c -o main.o", []string{"gcc", "-c", "main.c", "-o", "main.o"}},
		{"Double quoted", `gcc -DMESSAGE="Hello World" -c main.c`, []string{"gcc", "-DMESSAGE=Hello World", "-c", "main.c"}},
		{"Single quoted", `gcc -DMESSAGE='Hello World' -c main.c`, []string{"gcc", "-DMESSAGE=Hello World", "-c", "main.c"}},
		{"Escaped", `gcc -DMESSAGE=\"Hello\" -c main.c`, []string{"gcc", `-DMESSAGE="Hello"`, "-c", "main.c"}},
		{"Empty", "", nil},
		{"Single word", "gcc", []string{"gcc"}},
		{"Extra spaces", "  gcc   -c   main.c  ", []string{"gcc", "-c", "main.c"}},
		{"Backslash in single quotes", `gcc -DPATH='C:\path' -c main.c`, []string{"gcc", `-DPATH=C:\path`, "-c", "main.c"}},
		{"Windows drive path", `C:\mingw\bin\gcc.exe -c foo.c`, []string{`C:\mingw\bin\gcc.exe`, "-c", "foo.c"}},
		{"Windows path with quotes", `gcc -I"C:\Program Files\include" -c foo.c`, []string{"gcc", `-IC:\Program Files\include`, "-c", "foo.c"}},
		{"Escaped space still works", `gcc -c my\ file.c`, []string{"gcc", "-c", "my file.c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitCommandLine(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("Length = %d, expected %d\nGot: %v\nExp: %v", len(result), len(tt.expected), result, tt.expected)
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("[%d] = %q, expected %q", i, result[i], exp)
				}
			}
		})
	}
}

func TestParseOptionsDefaults(t *testing.T) {
	opts := ParseOptions{}
	if opts.UseArguments {
		t.Error("UseArguments should default to false")
	}
	if opts.Deduplicate {
		t.Error("Deduplicate should default to false")
	}
}
