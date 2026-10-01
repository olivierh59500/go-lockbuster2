// Package demo presents the original planar scroll engines with DCK audio.
package demo

import (
	"encoding/binary"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/sound"
	playback "github.com/olivierh59500/democonstructionkit/sound/ebiten"
	"github.com/olivierh59500/go-lockbuster2/assets"
	"github.com/olivierh59500/go-lockbuster2/internal/source"
	"image/color"
)

const Width, Height, FPS = 320, 200, 50

type Game struct {
	clock    *source.Clock
	image    *ebiten.Image
	colored  *ebiten.Image
	lookup   *composite.IndexedPalette
	shader   *ebiten.Shader
	player   *playback.Player
	pixels   []byte
	palette  [16]color.NRGBA
	raster   [1024]float32
	rasters  [2][]byte
	uniforms map[string]any
	closed   bool
}

func data(name string) ([]byte, error) { return assets.Files.ReadFile("original/" + name) }
func NewGame(mute bool) (_ *Game, err error) {
	g := &Game{pixels: make([]byte, Width*Height*4)}
	defer func() {
		if err != nil {
			g.Close()
		}
	}()
	font, err := data("font.bin")
	if err != nil {
		return nil, err
	}
	var messages [10][]byte
	var maps [6][]byte
	for i := range messages {
		messages[i], err = data(fmt.Sprintf("message-%d.bin", i))
		if err != nil {
			return nil, err
		}
	}
	for i := range maps {
		maps[i], err = data(fmt.Sprintf("copy-map-%d.bin", i))
		if err != nil {
			return nil, err
		}
	}
	g.clock, err = source.NewClock(font, messages, maps)
	if err != nil {
		return nil, err
	}
	for i, name := range []string{"raster-old.bin", "raster-new.bin"} {
		g.rasters[i], err = data(name)
		if err != nil {
			return nil, err
		}
	}
	g.image = ebiten.NewImage(Width, Height)
	g.colored = ebiten.NewImage(Width, Height)
	g.lookup, err = composite.NewIndexedPalette(composite.IndexedPaletteConfig{Palette: g.palette[:], Channel: composite.BitplaneRed, Blend: ebiten.BlendCopy})
	if err != nil {
		return nil, err
	}
	g.shader, err = ebiten.NewShader([]byte(nativeShader))
	if err != nil {
		return nil, err
	}
	g.uniforms = map[string]any{"Rasters": g.raster[:], "Colored": float32(0)}
	if !mute {
		music, e := data("music.ym")
		if e != nil {
			return nil, e
		}
		g.player, err = playback.Open(nil, "music.ym", music, sound.Options{SampleRate: 48000, BlockFrames: 960, Loop: true})
		if err != nil {
			return nil, err
		}
		g.player.Play()
	}
	g.refresh()
	return g, nil
}
func (g *Game) refresh() {
	words := [16]uint16{0, 0, 0x134, 0x235, 0x124, 0x023, 0x346, 0x245, 0x356, 0x457, 0x467, 0x567, 0x677, 0x577, 0x701, 0x777}
	if g.clock.Gold {
		words = [16]uint16{0, 0, 0x341, 0x532, 0x421, 0x320, 0x643, 0x542, 0x653, 0x754, 0x764, 0x765, 0x776, 0x775, 0x107, 0x777}
	}
	put := func(dst []float32, w uint16) {
		dst[0], dst[1], dst[2], dst[3] = float32(w>>8&7)*34/255, float32(w>>4&7)*34/255, float32(w&7)*34/255, 1
	}
	for i, w := range words {
		g.palette[i] = color.NRGBA{R: byte(w>>8&7) * 34, G: byte(w>>4&7) * 34, B: byte(w&7) * 34, A: 255}
	}
	_ = g.lookup.SetPalette(g.palette[:])
	bank := 0
	if g.clock.NewRaster {
		bank = 1
	}
	b := g.rasters[bank]
	for i := 0; i < 256; i++ {
		var w uint16
		if i*2+1 < len(b) {
			w = binary.BigEndian.Uint16(b[i*2:])
		}
		put(g.raster[i*4:], w)
	}
	colored := float32(0)
	if g.clock.ColorText {
		colored = 1
	}
	g.uniforms["Colored"] = colored
	g.clock.Pixels(g.pixels)
	g.image.WritePixels(g.pixels)
}
func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		return ebiten.Termination
	}
	g.clock.Step()
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.clock.Gold = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.clock.Gold = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.clock.ColorText = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF4) {
		g.clock.ColorText = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF5) {
		g.clock.NewRaster = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF6) {
		g.clock.NewRaster = false
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, _ := ebiten.TouchPosition(id)
		if x < Width/2 {
			g.clock.Gold = !g.clock.Gold
		} else {
			g.clock.ColorText = !g.clock.ColorText
		}
	}
	g.refresh()
	return nil
}
func (g *Game) Draw(dst *ebiten.Image) {
	_ = g.lookup.Draw(g.colored, g.image)
	op := ebiten.DrawRectShaderOptions{Images: [4]*ebiten.Image{g.image, g.colored}, Uniforms: g.uniforms, Blend: ebiten.BlendCopy}
	dst.DrawRectShader(Width, Height, g.shader, &op)
}
func (*Game) Layout(int, int) (int, int) { return Width, Height }
func (g *Game) Tick() int                { return g.clock.Tick }
func (g *Game) Close() {
	if g == nil || g.closed {
		return
	}
	g.closed = true
	if g.player != nil {
		g.player.Close()
	}
	if g.image != nil {
		g.image.Deallocate()
	}
	if g.colored != nil {
		g.colored.Deallocate()
	}
	if g.lookup != nil {
		g.lookup.Close()
	}
	if g.shader != nil {
		g.shader.Deallocate()
	}
}

const nativeShader = `//kage:unit pixels
package main
var Rasters [256]vec4
var Colored float
func Fragment(position vec4,source vec2,color vec4)vec4{
 p:=source-imageSrc0Origin()
 index:=int(clamp(floor(imageSrc0At(source).r*15+0.5),0,15))
 row:=int(clamp(floor(p.y/2),0,246))
 if index==0 {return Rasters[row]*color}
 if index==1 && Colored>0 {return Rasters[row+9]*color}
 return imageSrc1At(source)*color
}
`
