package transparencia

import (
	_ "embed"
)

//go:embed data/estados.csv
var estadosCSV []byte

//go:embed data/municipios.csv
var municipiosCSV []byte

//go:embed data/megasoft_municipios.csv
var megasoftMunicipiosCSV []byte

//go:embed data/portais_municipais.csv
var portaisMunicipaisCSV []byte

