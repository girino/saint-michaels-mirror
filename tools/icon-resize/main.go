// Copyright (c) 2025 Girino Vey.
//
// This software is licensed under the Girino License.
// See LICENSE file for full license text.
//
// Icon resize utility for Espelho de São Miguel.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

func resizeImage(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// applyRoundedMask multiplies the alpha channel of img by a rounded-rectangle
// mask. radius is in pixels. The mask uses a 1px antialias ramp for smooth edges.
func applyRoundedMask(img image.Image, radius float64) image.Image {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// helper clamp
	clamp := func(v, lo, hi float64) float64 {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}

	// anti-alias ramp width in pixels
	aa := 1.0

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// find nearest point inside the rounded rect center box
			fx := float64(x) + 0.5
			fy := float64(y) + 0.5
			nx := clamp(fx, radius, float64(w)-radius)
			ny := clamp(fy, radius, float64(h)-radius)
			dx := fx - nx
			dy := fy - ny
			dist := math.Hypot(dx, dy)

			// mask alpha calculation with 1px linear AA
			var mask float64
			if dist <= radius-aa {
				mask = 1.0
			} else if dist >= radius+aa {
				mask = 0.0
			} else {
				// linear ramp
				mask = (radius + aa - dist) / (2 * aa)
				if mask < 0 {
					mask = 0
				}
				if mask > 1 {
					mask = 1
				}
			}

			// sample source pixel
			sr, sg, sb, sa := img.At(x+b.Min.X, y+b.Min.Y).RGBA()
			// uint32 values are 0..0xffff; convert to uint8
			a8 := uint8((sa * 0xff) / 0xffff)
			// multiply alpha by mask
			newA := uint8(float64(a8) * mask)

			// if fully transparent after mask, write transparent pixel
			if newA == 0 {
				dst.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
				continue
			}

			// compute premultiplied colors for RGBA values
			r8 := uint8((sr * 0xff) / 0xffff)
			g8 := uint8((sg * 0xff) / 0xffff)
			b8 := uint8((sb * 0xff) / 0xffff)

			// If original alpha was 0, avoid division
			if a8 == 0 {
				dst.SetRGBA(x, y, color.RGBA{r8, g8, b8, newA})
			} else {
				// scale color channels by newA/origA to preserve color premultiplication
				scale := float64(newA) / float64(a8)
				nr := uint8(clamp(float64(r8)*scale, 0, 255))
				ng := uint8(clamp(float64(g8)*scale, 0, 255))
				nb := uint8(clamp(float64(b8)*scale, 0, 255))
				dst.SetRGBA(x, y, color.RGBA{nr, ng, nb, newA})
			}
		}
	}

	return dst
}

func main() {
	in := "cmd/saint-michaels-mirror/static/icon.png"
	f, err := os.Open(in)
	if err != nil {
		log.Fatalf("open icon: %v", err)
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		log.Fatalf("decode icon: %v", err)
	}

	sizes := []int{16, 32, 48, 64, 96, 128, 180}
	outdir := "cmd/saint-michaels-mirror/static/favicons"
	if err := os.MkdirAll(outdir, 0755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	for _, s := range sizes {
		res := resizeImage(src, s, s)
		// apply rounded corners: use radius as 40% of the smaller side for stronger rounding
		radius := math.Min(float64(s), float64(s)) * 0.40
		res = applyRoundedMask(res, radius)
		out := filepath.Join(outdir, fmt.Sprintf("favicon-%dx%d.png", s, s))
		of, err := os.Create(out)
		if err != nil {
			log.Fatalf("create: %v", err)
		}
		if err := png.Encode(of, res); err != nil {
			of.Close()
			log.Fatalf("encode: %v", err)
		}
		of.Close()
		log.Printf("wrote %s", out)
	}
}
