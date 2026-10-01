package source

import (
	"encoding/binary"
	"fmt"
)

const ScreenBytes = 32000
const origin = 32768
const bufferSize = 131072

// Clock translates the six native byte-copy scroll engines into bounded Go
// operations. Two retained buffers preserve their original feedback topology.
type Clock struct {
	Buffers                                                             [2][]byte
	Front, Tick, Phase, Frame, BytePhase, Offset, Remaining, HeaderTick int
	Font                                                                []byte
	Messages                                                            [10][]byte
	Maps                                                                [6][]Copy
	Compiled                                                            []byte
	Header                                                              [2][]byte
	Gold, ColorText, NewRaster                                          bool
}

func NewClock(font []byte, messages [10][]byte, maps [6][]byte) (*Clock, error) {
	if len(font) != 59*400 {
		return nil, fmt.Errorf("source: incomplete planar font")
	}
	c := &Clock{Font: font, Messages: messages, Phase: -1}
	for i := range c.Buffers {
		c.Buffers[i] = make([]byte, bufferSize)
	}
	for i, b := range maps {
		if len(b) != 160 {
			return nil, fmt.Errorf("source: incomplete copy map")
		}
		for j := 0; j < len(b); j += 4 {
			c.Maps[i] = append(c.Maps[i], Copy{int16(binary.BigEndian.Uint16(b[j:])), int16(binary.BigEndian.Uint16(b[j+2:]))})
		}
	}
	for i := range c.Header {
		c.Header[i] = make([]byte, 17000)
		position := 320
		line := 0
		for _, r := range messages[i] {
			if r == ' ' {
				line++
				position = 320 + line*4000
				continue
			}
			glyph := max(0, min(58, int(r)-32)) * 400
			for y := 0; y < 25; y++ {
				if position+y*160+16 <= len(c.Header[i]) {
					copy(c.Header[i][position+y*160:], font[glyph+y*16:glyph+y*16+16])
				}
			}
			position += 16
		}
	}
	return c, nil
}

func (c *Clock) Screen() []byte { return c.Buffers[c.Front][origin : origin+ScreenBytes] }
func (c *Clock) Step() {
	c.Tick++
	if c.HeaderTick < 610 {
		bank := c.HeaderTick / 305
		frame := c.HeaderTick%305 + 1
		start := -99 + frame
		length := 101 * 160
		if bank == 1 {
			c.Gold = true
			start = 202 - frame
			length = 106 * 160
		}
		target := 1 - c.Front
		copy(c.Buffers[target][origin+start*160:], c.Header[bank][:length])
		c.Front = target
		c.HeaderTick++
		return
	}
	if c.Remaining == 0 {
		c.Phase = (c.Phase + 1) % 8
		c.Frame, c.BytePhase, c.Offset = 0, 0, 0
		text := c.Messages[2+c.Phase]
		c.Remaining = len(text) * 4
		c.Compiled = make([]byte, len(text)*400+416)
		for i, r := range text {
			glyph := max(0, min(58, int(r)-32)) * 400
			copy(c.Compiled[i*400:], c.Font[glyph:glyph+400])
		}
		switch c.Phase {
		case 0:
			c.ColorText = false
		case 1:
			c.Gold, c.ColorText = true, true
		case 2:
			c.ColorText = false
		case 3:
			c.Gold = true
		case 4:
			c.ColorText, c.NewRaster = false, false
		case 5:
			c.Gold, c.ColorText = true, true
		}
	}
	mode := [8]int{0, 1, 2, 3, 4, 5, 5, 5}[c.Phase]
	if c.Frame == 4 {
		c.Frame = 0
		c.Offset += 384
	}
	base := origin + 87*160
	if mode == 3 {
		base += 15 * 160
	}
	if mode == 4 {
		base -= 20 * 160
	}
	if mode == 5 {
		base -= 15 * 160
	}
	from, to := c.Buffers[c.Front], c.Buffers[1-c.Front]
	height := 25
	if mode == 5 {
		height = 24
	}
	for y := 0; y < height; y++ {
		for plane := 0; plane < 4; plane++ {
			start := base + y*160 + plane*2
			for _, op := range c.Maps[mode] {
				dst := start + int(op.Destination)
				if op.Source == 0x7fff {
					off := c.Offset + y*16 + plane*2
					if off < len(c.Compiled) {
						to[dst] = c.Compiled[off]
					}
				} else {
					to[dst] = from[start+int(op.Source)]
				}
			}
		}
	}
	c.Front = 1 - c.Front
	c.BytePhase ^= 1
	if c.BytePhase == 0 {
		c.Offset += 6
	}
	c.Offset++
	c.Frame++
	c.Remaining--
	buf := c.Buffers[c.Front]
	// Reflection/duplicate passes run after the display-buffer exchange.
	copyRows := func(src, dst, rows, step int, double bool) {
		for y := 0; y < rows; y++ {
			copy(buf[dst:dst+160], buf[src:src+160])
			if double {
				copy(buf[dst+160:dst+320], buf[src:src+160])
			}
			src += 160
			dst += step
		}
	}
	switch mode {
	case 0:
		copyRows(base, base+0x38e0, 24, -320, false)
	case 1:
		copyRows(base+160, base+0x3700, 23, -320, true)
	case 2:
		copyRows(base, base+0x46a0, 35, -320, false)
	case 3:
		src, dst, step := base, base-0x2a80, 160
		for count := 22; count >= 0; count-- {
			copy(buf[dst:dst+160], buf[src:src+160])
			copy(buf[dst+160:dst+320], buf[src:src+160])
			src += 160
			dst += 160
			if count == 20 {
				step += 160
			}
			dst += step
		}
	case 4:
		copyRows(base, base+0x3e80, 29, -320, false)
		pos, delta := origin+0x50f0, 238
		for count := 19; count >= 0; count-- {
			n := (count + 21) * 2
			clear(buf[pos : pos+n])
			pos += n - delta
			delta -= 2
		}
	case 5:
		for y := 0; y < 24; y++ {
			s := base + y*160
			copy(buf[s+0x2bc0:s+0x2bc0+160], buf[s:s+160])
			copy(buf[s+0x2bc0-0x5140:s+0x2bc0-0x5140+160], buf[s:s+160])
		}
	}
}

// Pixels converts four planar bytes per eight pixels into an indexed RGBA image.
// The caller owns and reuses dst; this method allocates no memory.
func (c *Clock) Pixels(dst []byte) {
	b := c.Screen()
	for y := 0; y < 200; y++ {
		for block := 0; block < 20; block++ {
			at := y*160 + block*8
			for half := 0; half < 2; half++ {
				p0, p1, p2, p3 := b[at+half], b[at+half+2], b[at+half+4], b[at+half+6]
				for bit := 0; bit < 8; bit++ {
					shift := uint(7 - bit)
					v := (p0 >> shift & 1) | (p1>>shift&1)<<1 | (p2>>shift&1)<<2 | (p3>>shift&1)<<3
					o := (y*320 + block*16 + half*8 + bit) * 4
					dst[o], dst[o+1], dst[o+2], dst[o+3] = v*17, 0, 0, 255
				}
			}
		}
	}
}
