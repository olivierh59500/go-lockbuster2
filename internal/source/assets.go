// Package source decodes the native intro's artwork and authored copy maps.
package source

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

type Copy struct{ Source, Destination int16 }

// ExportAssets keeps data and display maps; it does not embed executable code.
func ExportAssets(prg []byte, directory string) error {
	if len(prg) < 28+43842 || binary.BigEndian.Uint16(prg) != 0x601a {
		return fmt.Errorf("source: incomplete Lockbuster 2 executable")
	}
	b := prg[28 : 28+43842]
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	write := func(name string, data []byte) error { return os.WriteFile(filepath.Join(directory, name), data, 0644) }
	for _, part := range []struct {
		name       string
		start, end int
	}{
		{"font.bin", 0x1f90, 0x1f90 + 59*400}, {"raster-old.bin", 0x1330, 0x1446}, {"raster-new.bin", 0x144a, 0x1536},
	} {
		if err := write(part.name, b[part.start:part.end]); err != nil {
			return err
		}
	}
	for i, offset := range []int{0x1536, 0x1558, 0x1584, 0x1eb8, 0x1638, 0x1700, 0x17ac, 0x1900, 0x1ad2, 0x1cda} {
		end := offset
		for end < len(b) && b[end] != 0 {
			end++
		}
		if end == len(b) {
			return fmt.Errorf("source: unterminated message")
		}
		if err := write(fmt.Sprintf("message-%d.bin", i), b[offset:end]); err != nil {
			return err
		}
	}
	// Unrolled MOVE.B instructions encode each horizontal/deformed copy path.
	starts := []int{0x738, 0xfba, 0x8d8, 0xe08, 0xc28, 0xa82}
	for mode, start := range starts {
		var ops []byte
		for pc := start; pc < len(b); {
			op := binary.BigEndian.Uint16(b[pc:])
			pc += 2
			switch op {
			case 0x1568:
				ops = append(ops, b[pc:pc+4]...)
				pc += 4
			case 0x14a8:
				ops = append(ops, b[pc:pc+2]...)
				ops = append(ops, 0, 0)
				pc += 2
			case 0x1551:
				ops = append(ops, 0x7f, 0xff)
				ops = append(ops, b[pc:pc+2]...)
				pc += 2
			default:
				if op != 0x5489 {
					return fmt.Errorf("source: unexpected copy opcode %04x", op)
				}
				goto complete
			}
		}
	complete:
		if len(ops) != 40*4 {
			return fmt.Errorf("source: copy path %d has %d steps", mode, len(ops)/4)
		}
		if err := write(fmt.Sprintf("copy-map-%d.bin", mode), ops); err != nil {
			return err
		}
	}
	img := image.NewNRGBA(image.Rect(0, 0, 16*32, 4*25))
	for c := 0; c < 59; c++ {
		for y := 0; y < 25; y++ {
			for x := 0; x < 32; x++ {
				var v byte
				for p := 0; p < 4; p++ {
					w := binary.BigEndian.Uint16(b[0x1f90+c*400+y*16+x/16*8+p*2:])
					v |= byte(w>>uint(15-x%16)&1) << p
				}
				img.SetNRGBA(c%16*32+x, c/16*25+y, color.NRGBA{R: v * 17, A: 255})
			}
		}
	}
	f, err := os.Create(filepath.Join(directory, "font.png"))
	if err != nil {
		return err
	}
	err = png.Encode(f, img)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
