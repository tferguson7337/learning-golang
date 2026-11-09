package main

import "fmt"

type ByteUnit uint8

const (
	Byte ByteUnit = iota
	Kibibyte
	Mebibyte
	Gibibyte
	Tebibyte
	Pebibyte
	Exbibyte
	Zebibyte
	Yobibyte
	Robibyte
	Quebibyte
)

func (unit ByteUnit) String() string {
	switch unit {
	case Byte:
		return "B"
	case Kibibyte:
		return "KiB"
	case Mebibyte:
		return "MiB"
	case Gibibyte:
		return "GiB"
	case Tebibyte:
		return "TiB"
	case Pebibyte:
		return "PiB"
	case Exbibyte:
		return "EiB"
	case Zebibyte:
		return "ZiB"
	case Yobibyte:
		return "YiB"
	case Robibyte:
		return "RiB"
	case Quebibyte:
		return "QiB"
	}
	return "???"
}

type ScaledByteInfo struct {
	size float64
	unit ByteUnit
}

func NewScaledByteInfo(size uint64, unit ByteUnit) ScaledByteInfo {
	bi := ScaledByteInfo{
		size: float64(size),
		unit: unit,
	}

	for (bi.size >= 1024) && (bi.unit < Quebibyte) {
		bi.size /= 1024
		bi.unit++
	}

	return bi
}

func (sbi ScaledByteInfo) String() string {
	// Adjust the size based on the unit for display purposes
	if sbi.unit == Byte {
		return fmt.Sprintf("%d%v", uint(sbi.size), sbi.unit)
	}
	return fmt.Sprintf("%.2f%v", sbi.size, sbi.unit)
}
