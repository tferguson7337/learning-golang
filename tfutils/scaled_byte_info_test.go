package tfutils

import "testing"

func TestScaledByteInfo(t *testing.T) {
	tests := []struct {
		name string
		size uint64
		unit ByteUnit

		expectedSize   float64
		expectedUnit   ByteUnit
		expectedString string
	}{
		{
			name:           "Invalid byte unit",
			size:           1024,
			unit:           ByteUnit(255),
			expectedSize:   1024,
			expectedUnit:   ByteUnit(255),
			expectedString: "1024.00???",
		},
		{
			name:           "Zero byte",
			size:           0,
			unit:           Byte,
			expectedSize:   0,
			expectedUnit:   Byte,
			expectedString: "0B",
		},
		{
			name:           "0 KiB",
			size:           0,
			unit:           Kibibyte,
			expectedSize:   0,
			expectedUnit:   Kibibyte,
			expectedString: "0.00KiB",
		},
		{
			name:           "1023 bytes",
			size:           1023,
			unit:           Byte,
			expectedSize:   1023,
			expectedUnit:   Byte,
			expectedString: "1023B",
		},
		{
			name:           "16k bytes and change",
			size:           16*1024 + 512,
			unit:           Byte,
			expectedSize:   16.50,
			expectedUnit:   Kibibyte,
			expectedString: "16.50KiB",
		},
		{
			name:           "1M bytes and change",
			size:           1024*1024 + (256 * 1024),
			unit:           Byte,
			expectedSize:   1.25,
			expectedUnit:   Mebibyte,
			expectedString: "1.25MiB",
		},
		{
			name:           "1 KiB",
			size:           1024,
			unit:           Byte,
			expectedSize:   1,
			expectedUnit:   Kibibyte,
			expectedString: "1.00KiB",
		},
		{
			name:           "1023 KiB",
			size:           1023 * 1024,
			unit:           Byte,
			expectedSize:   1023,
			expectedUnit:   Kibibyte,
			expectedString: "1023.00KiB",
		},
		{
			name:           "1 MiB",
			size:           1024 * 1024,
			unit:           Byte,
			expectedSize:   1,
			expectedUnit:   Mebibyte,
			expectedString: "1.00MiB",
		},
		{
			name:           "1023 MiB",
			size:           1023 * 1024 * 1024,
			unit:           Byte,
			expectedSize:   1023,
			expectedUnit:   Mebibyte,
			expectedString: "1023.00MiB",
		},
		{
			name:           "1 GiB",
			size:           1024 * 1024 * 1024,
			unit:           Byte,
			expectedSize:   1,
			expectedUnit:   Gibibyte,
			expectedString: "1.00GiB",
		},
		{
			name:           "1023 GiB",
			size:           1023 * 1024 * 1024 * 1024,
			unit:           Byte,
			expectedSize:   1023,
			expectedUnit:   Gibibyte,
			expectedString: "1023.00GiB",
		},
		{
			name:           "1 TiB",
			size:           1024 * 1024 * 1024 * 1024,
			unit:           Byte,
			expectedSize:   1,
			expectedUnit:   Tebibyte,
			expectedString: "1.00TiB",
		},
		{
			name:           "1023 TiB",
			size:           1023 * 1024 * 1024 * 1024 * 1024,
			unit:           Byte,
			expectedSize:   1023,
			expectedUnit:   Tebibyte,
			expectedString: "1023.00TiB",
		},
		{
			name:           "1 PiB",
			size:           1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Byte,
			expectedSize:   1,
			expectedUnit:   Pebibyte,
			expectedString: "1.00PiB",
		},
		{
			name:           "1023 PiB",
			size:           1023 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Byte,
			expectedSize:   1023,
			expectedUnit:   Pebibyte,
			expectedString: "1023.00PiB",
		},
		{
			name:           "1 EiB",
			size:           1024 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Byte,
			expectedSize:   1,
			expectedUnit:   Exbibyte,
			expectedString: "1.00EiB",
		},
		{
			name:           "1023 EiB",
			size:           1023 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Kibibyte,
			expectedSize:   1023,
			expectedUnit:   Exbibyte,
			expectedString: "1023.00EiB",
		},
		{
			name:           "1 ZiB",
			size:           1024 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Kibibyte,
			expectedSize:   1,
			expectedUnit:   Zebibyte,
			expectedString: "1.00ZiB",
		},
		{
			name:           "1023 ZiB",
			size:           1023 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Mebibyte,
			expectedSize:   1023,
			expectedUnit:   Zebibyte,
			expectedString: "1023.00ZiB",
		},
		{
			name:           "1 YiB",
			size:           1024 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Mebibyte,
			expectedSize:   1,
			expectedUnit:   Yobibyte,
			expectedString: "1.00YiB",
		},
		{
			name:           "1023 YiB",
			size:           1023 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Gibibyte,
			expectedSize:   1023,
			expectedUnit:   Yobibyte,
			expectedString: "1023.00YiB",
		},
		{
			name:           "1 RiB",
			size:           1024 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Gibibyte,
			expectedSize:   1,
			expectedUnit:   Robibyte,
			expectedString: "1.00RiB",
		},
		{
			name:           "1023 RiB",
			size:           1023 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Tebibyte,
			expectedSize:   1023,
			expectedUnit:   Robibyte,
			expectedString: "1023.00RiB",
		},
		{
			name:           "1 QiB",
			size:           1024 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Tebibyte,
			expectedSize:   1,
			expectedUnit:   Quebibyte,
			expectedString: "1.00QiB",
		},
		{
			name:           "1023 QiB",
			size:           1023 * 1024 * 1024 * 1024 * 1024 * 1024,
			unit:           Pebibyte,
			expectedSize:   1023,
			expectedUnit:   Quebibyte,
			expectedString: "1023.00QiB",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NewScaledByteInfo(test.size, test.unit)
			if result.size != test.expectedSize {
				t.Errorf("expected size %v, got %v", test.expectedSize, result.size)
			}
			if result.unit != test.expectedUnit {
				t.Errorf("expected unit %v, got %v", test.expectedUnit, result.unit)
			}
			if result.String() != test.expectedString {
				t.Errorf("expected string %v, got %v", test.expectedString, result.String())
			}
		})
	}
}
