package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func getALT(data map[string]string) string {
	if data["ALT"] == "." {
		return "."
	}
	newAlt := ""
	infoParts := strings.Split(data["INFO"], ";")
	var RU string
	if len(infoParts) >= 4 && strings.HasPrefix(infoParts[3], "RU=") {
		RU = strings.TrimPrefix(infoParts[3], "RU=")
	}

	altCounts := strings.Split(data["ALT"], ",")
	for _, altCountSTR := range altCounts {
		altCount := strings.TrimPrefix(strings.TrimSuffix(altCountSTR, ">"), "<STR")
		count := 0
		fmt.Sscanf(altCount, "%d", &count)
		alt := strings.Repeat(RU, count)
		if newAlt != "" {
			newAlt += "," + alt
		} else {
			newAlt = alt
		}
	}
	return newAlt
}

func formatFormat(formatString string) string {
	return "GT"
}

func sampleFormat(sampleString string) string {
	return strings.Split(sampleString, ":")[0]
}

func main() {
	inputFilesPath := os.Args[1]

	files, err := ioutil.ReadDir(inputFilesPath)
	if err != nil {
		panic(err)
	}

	var filePaths []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".vcf") {
			filePaths = append(filePaths, filepath.Join(inputFilesPath, file.Name()))
		}
	}

	for chromosomeIndex, filePath := range filePaths {
		inputFile, err := os.Open(filePath)
		if err != nil {
			panic(err)
		}
		defer inputFile.Close()

		outputFile, err := os.Create(fmt.Sprintf("%d_formatted.vcf", chromosomeIndex+1))
		if err != nil {
			panic(err)
		}
		defer outputFile.Close()

		holdingFile, err := os.Create("holding_file")
		if err != nil {
			panic(err)
		}
		defer holdingFile.Close()

		scanner := bufio.NewScanner(inputFile)

		var header []string
		var dataLines int
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "#") {
				dataLines++
			}
		}
		inputFile.Seek(0, 0) // rewind to start
		scanner = bufio.NewScanner(inputFile)

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "##") {
				outputFile.WriteString(line + "\n")
			} else if strings.HasPrefix(line, "#CHROM") {
				header = strings.Split(strings.TrimSpace(line), "\t")
				header[9] = "SAMPLE"
			} else {
				lineParts := strings.Split(strings.TrimSpace(line), "\t")
				data := map[string]string{
					"CHROM":     lineParts[0],
					"POS":       lineParts[1],
					"ID":        lineParts[2],
					"REF":       lineParts[3],
					"ALT":       lineParts[4],
					"QUAL":      lineParts[5],
					"FILTER":    lineParts[6],
					"INFO":      lineParts[7],
					"FORMAT":    lineParts[8],
					"SAMPLE":    lineParts[9],
					"SAMPLE_ID": lineParts[10],
				}

				var newLine []string
				newLine = append(newLine, data["CHROM"])
				newLine = append(newLine, data["POS"])
				newLine = append(newLine, data["ID"])
				newLine = append(newLine, data["REF"])
				newLine = append(newLine, getALT(data))
				newLine = append(newLine, data["QUAL"])
				newLine = append(newLine, data["FILTER"])
				newLine = append(newLine, data["INFO"])
				newLine = append(newLine, formatFormat(data["FORMAT"]))

				// Placeholder samples
				for i := 0; i < dataLines+1; i++ {
					newLine = append(newLine, ".")
				}

				// Replace the last sample column with the real sample genotype
				if len(header) > 0 {
					sampleIndex := len(header) - 1
					newLine[sampleIndex] = sampleFormat(data["SAMPLE"])
					header = append(header, data["SAMPLE_ID"])
				}

				holdingFile.WriteString(strings.Join(newLine, "\t") + "\n")
			}
		}

		holdingFile.Close()
		holdingFile, _ = os.Open("holding_file")
		defer holdingFile.Close()

		outputFile.WriteString(strings.Join(header, "\t") + "\n")

		buf := make([]byte, 1024)
		for {
			n, err := holdingFile.Read(buf)
			if n > 0 {
				outputFile.Write(buf[:n])
			}
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}
		}
	}
}
