package main

const ADC = "ADC"

func Generate() map[uint8]string {
	opcodeTable := map[uint8]string{
		0x61: ADC,
		0x65: ADC,
		0x69: ADC,
		0x6D: ADC,
		0x71: ADC,
		0x75: ADC,
		0x79: ADC,
		0x7D: ADC,
	}
	return opcodeTable
}
