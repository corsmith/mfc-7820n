package main

import (
	"flag"
	"fmt"
	"net"
)

func main() {
	const brotherPort int = 54921

	brotherIP := flag.String("a", "192.168.0.157", "IP address of the Brother scanner")
	resolution := flag.Int("r", 300, "Resolution of the scan")
	color := flag.String("c", "CGRAY", "Color mode of the scan (CGRAY, TEXT)")
	name := flag.String("n", "scan.tiff", "Name of the output file")
	rawinput := flag.String("i", "", "raw input file to parse instead of socket")

	flag.Parse()

	if net.ParseIP(*brotherIP) == nil {
		HandleError(fmt.Errorf("invalid IP address: %s", *brotherIP))
	}

	rawImages, width, height := Scan(*brotherIP, brotherPort, *resolution, *color, *rawinput)

	for i, rawImage := range rawImages {
		if i == len(rawImages)-1 {
			SaveImage(rawImage, width, height, fmt.Sprintf("%s-%d.tiff", *name, i), *color)

		} else {
			go SaveImage(rawImage, width, height, fmt.Sprintf("%s-%d.tiff", *name, i), *color)
		}
	}
}
