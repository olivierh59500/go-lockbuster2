package source

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/olivierh59500/go-lockbuster2/assets"
	"io"
	"os"
	"testing"
)

func originalClock(t *testing.T) *Clock {
	t.Helper()
	read := func(name string) []byte {
		b, e := assets.Files.ReadFile("original/" + name)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	var messages [10][]byte
	var maps [6][]byte
	for i := range messages {
		messages[i] = read(fmt.Sprintf("message-%d.bin", i))
	}
	for i := range maps {
		maps[i] = read(fmt.Sprintf("copy-map-%d.bin", i))
	}
	c, e := NewClock(read("font.bin"), messages, maps)
	if e != nil {
		t.Fatal(e)
	}
	return c
}

// The expected screens are memory snapshots of the original 68000 routines.
func TestNativeDisplayCheckpoints(t *testing.T) {
	b, e := os.ReadFile("testdata/checkpoints.json")
	if e != nil {
		t.Fatal(e)
	}
	var points []struct {
		Name string
		Tick int
	}
	if e = json.Unmarshal(b, &points); e != nil {
		t.Fatal(e)
	}
	c := originalClock(t)
	for _, p := range points {
		for c.Tick < p.Tick {
			c.Step()
		}
		t.Run(p.Name, func(t *testing.T) {
			f, e := os.Open("testdata/" + p.Name + ".bin.gz")
			if e != nil {
				t.Fatal(e)
			}
			defer f.Close()
			z, e := gzip.NewReader(f)
			if e != nil {
				t.Fatal(e)
			}
			defer z.Close()
			expected, e := io.ReadAll(z)
			if e != nil {
				t.Fatal(e)
			}
			actual := c.Screen()
			if !bytes.Equal(actual, expected) {
				differences := 0
				for i := range actual {
					if actual[i] != expected[i] {
						differences++
					}
				}
				t.Fatalf("%d native display bytes differ at tick %d", differences, p.Tick)
			}
		})
	}
}

func TestCompleteSequenceAndRestart(t *testing.T) {
	c := originalClock(t)
	pixels := make([]byte, 320*200*4)
	for c.Tick < 22000 {
		c.Step()
		if c.Tick%251 == 0 {
			c.Pixels(pixels)
			for i := 0; i < len(pixels); i += 4 {
				if pixels[i]%17 != 0 || pixels[i+3] != 255 {
					t.Fatal("invalid indexed output")
				}
			}
		}
	}
	if c.Phase < 0 || c.Phase > 7 || c.HeaderTick != 610 {
		t.Fatal("invalid sequence after two cycles")
	}
}
