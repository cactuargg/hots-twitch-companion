//go:build ignore

// Generate the Windows icon from the existing H brand geometry, without fonts.
// Run from repository root: go run assets/generate.go
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	var images [][]byte
	sizes := []int{16, 32, 48, 64, 128, 256}
	for _, size := range sizes {
		im := image.NewNRGBA(image.Rect(0, 0, size, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				u, v := (float64(x)+.5)*64/float64(size), (float64(y)+.5)*64/float64(size)
				dx, dy := 0.0, 0.0
				if u < 14 {
					dx = 14 - u
				}
				if u > 50 {
					dx = u - 50
				}
				if v < 14 {
					dy = 14 - v
				}
				if v > 50 {
					dy = v - 50
				}
				if dx*dx+dy*dy > 196 {
					continue
				}
				c := color.NRGBA{48, 51, 96, 255}
				if v >= 16 && v < 48 && ((u >= 18 && u < 26) || (u >= 38 && u < 46) || (v >= 28 && v < 36 && u >= 26 && u < 38)) {
					c = color.NRGBA{214, 213, 255, 255}
				}
				im.SetNRGBA(x, y, c)
			}
		}
		var b bytes.Buffer
		if err := png.Encode(&b, im); err != nil {
			panic(err)
		}
		images = append(images, b.Bytes())
	}
	var ico bytes.Buffer
	write := func(v any) {
		if err := binary.Write(&ico, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	write(uint16(0))
	write(uint16(1))
	write(uint16(len(images)))
	offset := uint32(6 + 16*len(images))
	for i, pngData := range images {
		write(uint8(sizes[i] % 256))
		write(uint8(sizes[i] % 256))
		write(uint16(0))
		write(uint16(1))
		write(uint16(32))
		write(uint32(len(pngData)))
		write(offset)
		offset += uint32(len(pngData))
	}
	for _, b := range images {
		ico.Write(b)
	}
	if err := os.WriteFile("assets/app.ico", ico.Bytes(), 0644); err != nil {
		panic(err)
	}
}
