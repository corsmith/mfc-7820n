package main

type requests struct {
	leaseRequest string
	scanRequest  string
	nextPageRequest  string
}

type modes struct {
	color     string
	grayscale string
}

type encode struct {
	jpeg string
	none string
}

type constants struct {
	ready       string
	mode        modes
	compression encode
	headerLen   int
	A4height    int
	mmInch      float32
	endPage     byte
	endScan     byte
	startGray   byte
}

var scanner constants = constants{
	ready: "+OK 200",
	mode: modes{
		color:     "CGRAY",
		grayscale: "TEXT",
	},
	compression: encode{
		jpeg: "JPEG",
		none: "NONE",
	},
	headerLen: 3,
	A4height:  294,
	mmInch:    25.4,
	endPage:   0x81,
	endScan:   0x80,
	startGray: 0x40,
}

var formats requests = requests{
	leaseRequest: "\x1bI\nR=%d,%d\nM=%s\n\x80",
	scanRequest:  "\x1bX\nR=%d,%d\nM=%s\nC=%s\nD=SIN\nB=50\nN=50\nA=0,0,%d,%d\n\x80",
	nextPageRequest:  "\x1bX\x80",
}
