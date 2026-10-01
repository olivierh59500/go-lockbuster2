# Lockbuster 2 Go

A Go/Ebitengine conversion of Lockbuster’s second Atari ST intro for DMA,
using Demo Construction Kit **v1.0.13**.

The original 32 × 25 pixel font, text and raster tables are decoded from the
native executable. Six scroll engines reproduce its byte-copy deformations,
mirrored text and duplicate rows. The two title sequences and eight message
sections run at the original **50 Hz** on desktop and Android.

DCK handles the indexed palette and YM playback. The production-specific Go
clock preserves the two screen buffers and the 68000 copy order. Seven display
checkpoints, including all six scroll engines, match the original screen memory
byte for byte. A full sequence and restart are covered by regression tests.

## Run

```sh
go run ./cmd/lockbuster2
```

- F1 / F2: gold / metallic font palette.
- F3 / F4: colored / black text background.
- F5 / F6: alternate / original raster table.
- Escape or Space: exit the intro.

On a touch screen, tap the left half to change the font palette or the right
half to switch the text background. The game and trainer mentioned in the
original scrolltext are historical text; the intro does not launch a game.

## Android

```sh
./scripts/run-android.sh
```

The script builds an ARM64 APK, installs it on one authorized USB device and
launches **Lockbuster 2**. Use `--build-only` to skip installation. Android SDK
36, NDK 28.2, Java 17 and the included Gradle wrapper are used. Landscape mode,
immersive display and screen wakefulness are handled by the Android host.

## Soundtrack

The included replacement soundtrack is **Roll out 1** by **Mad Max
(Jochen Hippel)**. DCK opens the YM file and loops it independently of the
50 Hz visual clock. Use `-mute` to disable device audio.

## Verification and extraction

```sh
go test ./...
go vet ./...
go run ./cmd/lockbuster2 -capture captures -frame 1200 -mute
go run ./cmd/extract -input /path/to/intro.prg -assets /path/to/export
```

Captures use native resolution. Asset extraction exports artwork and authored
copy maps, rather than embedding the original executable.

Android validation: the ARM64 build was installed and checked on a Pixel 10a.
The application maintains approximately 50 simulation updates and 60 displayed
frames per second. APK signing and 16 KB ZIP/ELF alignment checks pass.

## Video export

```sh
go run ./cmd/video
```

This creates a three-minute 50 fps H.264/AAC MP4, a PNG poster and a JSON report
under `recordings/`. DCK exports only the game canvas and its own audio, using
one simulation clock. Graphics are scaled by an integer factor of two. The
recordings are local generated media. Duration and poster time can be changed
with `-duration` and `-poster-at`.
