package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/draw"
)

func resizeImage(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

func main() {
	in := "cmd/khatru-relay/static/icon.png"
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
	outdir := "cmd/khatru-relay/static/favicons"
	if err := os.MkdirAll(outdir, 0755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	for _, s := range sizes {
		res := resizeImage(src, s, s)
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
