package commands

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestReadPayload(t *testing.T) {
	dir := t.TempDir()
	nonEmptyPath := filepath.Join(dir, "payload.txt")
	if err := os.WriteFile(nonEmptyPath, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyPath := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(emptyPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		file    string
		inline  string
		fileSet bool
		dataSet bool
		want    []byte
		wantErr string
	}{
		{
			name:    "neither flag set",
			wantErr: "provide --file or --data",
		},
		{
			name:    "data unset (empty string value ignored)",
			inline:  "",
			dataSet: false,
			wantErr: "provide --file or --data",
		},
		{
			name:    "data set and empty",
			inline:  "",
			dataSet: true,
			want:    []byte{},
		},
		{
			name:    "data set and non-empty",
			inline:  "hello",
			dataSet: true,
			want:    []byte("hello"),
		},
		{
			name:    "file unset (empty string value ignored)",
			file:    "",
			fileSet: false,
			wantErr: "provide --file or --data",
		},
		{
			name:    "file set and empty",
			file:    "",
			fileSet: true,
			want:    []byte{},
		},
		{
			name:    "file set and non-empty",
			file:    nonEmptyPath,
			fileSet: true,
			want:    []byte("hello"),
		},
		{
			name:    "file set to empty on-disk file",
			file:    emptyPath,
			fileSet: true,
			want:    []byte{},
		},
		{
			name:    "file and data both set",
			file:    nonEmptyPath,
			inline:  "hello",
			fileSet: true,
			dataSet: true,
			wantErr: "--file and --data are mutually exclusive",
		},
		{
			name:    "file and data both set even when empty",
			file:    "",
			inline:  "",
			fileSet: true,
			dataSet: true,
			wantErr: "--file and --data are mutually exclusive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readPayload(tc.file, tc.inline, tc.fileSet, tc.dataSet)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil (payload=%q)", tc.wantErr, got)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("error = %q; want %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("payload = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestReadPayloadFlagVisit(t *testing.T) {
	// End-to-end through FlagSet.Visit: unset vs set-but-empty vs set-non-empty.
	parse := func(t *testing.T, args ...string) (file, data string, fileSet, dataSet bool) {
		t.Helper()
		fs := flag.NewFlagSet("send-test", flag.ContinueOnError)
		filePtr := fs.String("file", "", "")
		dataPtr := fs.String("data", "", "")
		if err := fs.Parse(args); err != nil {
			t.Fatalf("parse: %v", err)
		}
		fs.Visit(func(f *flag.Flag) {
			switch f.Name {
			case "file":
				fileSet = true
			case "data":
				dataSet = true
			}
		})
		return *filePtr, *dataPtr, fileSet, dataSet
	}

	cases := []struct {
		name    string
		args    []string
		want    []byte
		wantErr string
	}{
		{name: "unset", wantErr: "provide --file or --data"},
		{name: "data set empty", args: []string{"--data", ""}, want: []byte{}},
		{name: "data set non-empty", args: []string{"--data", "ping"}, want: []byte("ping")},
		{name: "file set empty", args: []string{"--file", ""}, want: []byte{}},
		{name: "mutual exclusion", args: []string{"--file", "", "--data", ""}, wantErr: "--file and --data are mutually exclusive"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, data, fileSet, dataSet := parse(t, tc.args...)
			got, err := readPayload(file, data, fileSet, dataSet)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("err=%v; want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("got %q; want %q", got, tc.want)
			}
		})
	}

	t.Run("file set non-empty path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "p.bin")
		if err := os.WriteFile(path, []byte{0x01, 0x02}, 0o600); err != nil {
			t.Fatal(err)
		}
		file, data, fileSet, dataSet := parse(t, "--file", path)
		got, err := readPayload(file, data, fileSet, dataSet)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, []byte{0x01, 0x02}) {
			t.Fatalf("got %q", got)
		}
	})
}
