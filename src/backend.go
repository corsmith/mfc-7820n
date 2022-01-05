package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"github.com/corsmith/image/tiff"
	"log"
	"net"
	"os"
	"time"
)

func Scan(brotherIP string, brotherPort int, resolution int, color string, rawinput string, debug bool) ([][]byte, int, int) {
	if rawinput == "" {
		log.Println("Valid IP address, opening socket...")

		socket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", brotherIP, brotherPort))

		HandleError(err)

		defer socket.Close()

		width, height := sendRequest(socket, resolution, color)
		bytes, err := getScanBytes(socket)

		HandleError(err)
		if debug {
			err = os.WriteFile(".rawbytes", bytes, 0644)
			HandleError(err)
		}
		return removeHeaders(bytes), width, height
	} else {
		log.Println("Bypassing socket...")
		width := 1648
		height := 2287

		bytes, err := os.ReadFile(rawinput)
		HandleError(err)
		return removeHeaders(bytes), width, height
	}
}

func sendRequest(socket net.Conn, resolution int, _mode string) (int, int) {

	mode, compression := getCompressionMode(_mode)

	log.Println("Reading scanner status...")

	status := readPacket(socket)[:7]

	if status != scanner.ready {
		HandleError(fmt.Errorf("invalid reply from scanner: %s", status))
	}

	log.Println("Leasing options...")

	request := []byte(fmt.Sprintf(formats.leaseRequest, resolution, resolution, mode))
	sendPacket(socket, request)

	offer := readPacket(socket)

	log.Println("Sending scan request...")

	width, height := 0, 0
	planeWidth, planeHeight := 0, 0
	dpiX, dpiY := 0, 0
	adfStatus := 0

	fmt.Sscanf(offer[2:], "%d,%d,%d,%d,%d,%d,%d", &dpiX, &dpiY, &adfStatus, &planeWidth, &width, &planeHeight, &height)

	log.Println("Sending scan request dpiX, dpiY", dpiX, dpiY)

	request = []byte(fmt.Sprintf(formats.scanRequest, dpiX, dpiY, mode, compression, width, height))

	sendPacket(socket, request)

	log.Println("Scanning started...")

	return width, height
}

func getScanBytes(socket net.Conn) ([]byte, error) {
	log.Println("Getting packets...")

	packet := make([]byte, 2048)
	scanBytes := make([]byte, 0)

readPackets:
	for {
		socket.SetDeadline(time.Now().Add(time.Second * 10))
		bytes, err := socket.Read(packet)

		switch err := err.(type) {
		case net.Error:
			if err.Timeout() {
				break readPackets
			}

		case nil:
			scanBytes = append(scanBytes, packet[:bytes]...)
			if bytes == 1 && packet[0] == scanner.endScan {
				log.Println("Scan received...")
				break readPackets
			}

		default:
			HandleError(err)
		}
	}

	if (len(scanBytes)) < 1 {
		return scanBytes, fmt.Errorf("no data received")
	}

	log.Println("Captured %d bytes...", len(scanBytes))

	return scanBytes, nil
}

func SaveImage(data []byte, width int, height int, name string, color string, debug bool, resolution int) {

	log.Println("Saving image...")

	/* fix height based on actual scan lines received */
	actualheight := (len(data) * 8) / width
	_, compression := getCompressionMode(color)

	if compression != scanner.compression.jpeg {

		img := image.NewGray(image.Rectangle{
			image.Point{0, 0},
			image.Point{width, actualheight},
		})

		for i := 0; i < len(data); i++ {
			transx := (i * 8) % width
			transy := ((i * 8) / width)
			for o := 0; o < 8; o++ {
				sample := data[i] & (1 << o)
				if sample > 0 {
					sample = 255
				}
				img.SetGray(transx + (7 - o), transy, colorToGray(sample))
			}
		}

		file, err := os.Create(name)
		HandleError(err)

		var options tiff.Options = tiff.Options{
			Compression: tiff.Deflate,
			Predictor: true,
			Resolution: uint32(resolution),
		}

		err = tiff.Encode(file, img, &options)

		if debug {
			rawName := fmt.Sprintf("%s.raw", name)
			err = os.WriteFile(rawName, data, 0644)
			HandleError(err)
		}
	} else {
		err := os.WriteFile(name, data, 0644)
		HandleError(err)
	}
}

func removeHeaders(data []byte) [][]byte {
	log.Println("Removing headers from bytes...")

	pages := make([][]byte, 0)
	page := make([]byte, 0)

	currentPage := 1
	i := 0

headersLoop:
	for {
		if data[i] == scanner.endScan {
			log.Println("End Scan...")
			pages = append(pages, page)
			break headersLoop
		} else if data[i] == scanner.endPage {
			log.Println("End Page...")
			pages = append(pages, page)

			if len(data) > i+1 && data[i+1] == scanner.endScan {
				break headersLoop
			}

			page = make([]byte, 0)

			currentPage++

			i += scanner.headerLen - 2
			continue headersLoop
		} else if data[i] == scanner.startGray {
			payloadLen := binary.LittleEndian.Uint16(data[i+1 : i+3])
			// log.Println("... process record", fmt.Sprintf("%#04x", payloadLen))
			chunkSize := int(payloadLen)

			page = append(page, data[i+scanner.headerLen:i+scanner.headerLen+chunkSize]...)

			i += chunkSize + scanner.headerLen
		} else {
			// This is an error
			log.Fatalln("Invalid header type.  Giving up...")
			break headersLoop
		}
	}

	return pages
}
