package main

// go-webbrck is a lightweight package that is used to control a variety the legacy webbrick products

import (
	"bytes"
	"code.google.com/p/go-charset/charset" // For XML conversion
	_ "code.google.com/p/go-charset/data"  // Specs for dataset conversion
	"encoding/xml"                         // For XML work
	"fmt"                                  // For outputting stuff
)

type WebbrickConfig struct {
	Version string `xml:"Ver,attr"`
	Name    string `xml:"NN"`
	IP      IPMac  `xml:"SI"`
	CDs     struct{ CD []CD }
	CTs     struct{ CT []CT }
	CIs     struct{ CI []CI }
	NOs     struct{ NO []NO }
	NAs     struct{ NA []NA }
}

type IPMac struct {
	IPString  string `xml:"ip,attr"`
	MACString string `xml:"mac,attr"`
}

// <CD id="3" Name="Bath Floor" Opt="2">
type CD struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
	Opt  int    `xml:"Opt,attr"`
	Trg  Trg
}

type CT struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
	TrgL TrgL
	TrgH TrgH
}

type CI struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
	TrgL TrgL
	TrgH TrgH
}

type NO struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
}

type NA struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
}

// Trigger Types
type Trg struct {
	B1 int `xml:"B1,attr"`
	B2 int `xml:"B2,attr"`
	B3 int `xml:"B3,attr"`
	B4 int `xml:"B4,attr"`
}

type TrgL struct {
	Lo int `xml:"Lo,attr"`
	B1 int `xml:"B1,attr"`
	B2 int `xml:"B2,attr"`
	B3 int `xml:"B3,attr"`
	B4 int `xml:"B4,attr"`
}

type TrgH struct {
	Hi int `xml:"Hi,attr"`
	B1 int `xml:"B1,attr"`
	B2 int `xml:"B2,attr"`
	B3 int `xml:"B3,attr"`
	B4 int `xml:"B4,attr"`
}

func main() {
	fmt.Println("Hello, playground")
	GetWBStatus()
}

// Get WB Status on Initilisation
func GetWBStatus() bool {

	//var msg string
	var strMsg string
	strMsg = `<?xml version="1.0" encoding="ISO-8859-1"?>
<WebbrickConfig Ver="6.1.614">
	<NN>Documen</NN>
	<SI ip="10.100.100.101" mac="00:03:75:0F:83:99"/>
	<SN>25</SN>
	<SRs>
		<SR id="0" Value="8"/>
		<SR id="1" Value="8"/>
	</SRs>
	<SF>4</SF>
	<CDs>
		<CD id="0" Name="Door" Opt="2">
		<Trg B1="68" B2="0" B3="0" B4="0"/>
		</CD>
		<CD id="1" Name="Stair Lgt" Opt="2">
		<Trg B1="68" B2="1" B3="0" B4="0"/>
		</CD>
		<CD id="2" Name="Lounge" Opt="2">
		<Trg B1="68" B2="2" B3="0" B4="0"/>
		</CD>
		<CD id="3" Name="Bath Floor" Opt="2">
		<Trg B1="68" B2="3" B3="0" B4="165"/>
		</CD>
		<CD id="4" Name="Kitch Flr" Opt="2">
		<Trg B1="68" B2="4" B3="0" B4="0"/>
		</CD>
		<CD id="5" Name="Gar Door" Opt="2">
		<Trg B1="68" B2="5" B3="0" B4="0"/>
		</CD>
		<CD id="6" Name="Boost" Opt="2">
		<Trg B1="68" B2="6" B3="0" B4="0"/>
		</CD>
		<CD id="7" Name="Spare" Opt="2">
		<Trg B1="68" B2="7" B3="0" B4="165"/>
		</CD>
		<CD id="8" Name="Sw-9" Opt="3">
		<Trg B1="64" B2="0" B3="0" B4="0"/>
		</CD>
		<CD id="9" Name="Sw-10" Opt="3">
		<Trg B1="64" B2="0" B3="0" B4="0"/>
		</CD>
		<CD id="10" Name="Sw-11" Opt="3">
		<Trg B1="64" B2="0" B3="0" B4="0"/>
		</CD>
		<CD id="11" Name="Sw-12" Opt="3">
		<Trg B1="64" B2="0" B3="0" B4="0"/>
		</CD>
	</CDs>
	<CCs>
		<CC id="0" Dm="255" Ds="85" Am="15" Av="9302"/>
		<CC id="1" Dm="255" Ds="170" Am="0" Av="0"/>
		<CC id="2" Dm="0" Ds="0" Am="15" Av="10039"/>
		<CC id="3" Dm="0" Ds="0" Am="15" Av="0"/>
		<CC id="4" Dm="0" Ds="0" Am="0" Av="0"/>
		<CC id="5" Dm="0" Ds="0" Am="0" Av="0"/>
		<CC id="6" Dm="0" Ds="0" Am="0" Av="0"/>
		<CC id="7" Dm="0" Ds="0" Am="0" Av="0"/>
	</CCs>
	<CWs>
		<CW id="0">30</CW>
		<CW id="1">2</CW>
		<CW id="2">60</CW>
		<CW id="3">3600</CW>
		<CW id="4">300</CW>
		<CW id="5">600</CW>
		<CW id="6">900</CW>
		<CW id="7">1200</CW>
	</CWs>
	<CSs>
		<CS id="0">0</CS>
		<CS id="1">14</CS>
		<CS id="2">28</CS>
		<CS id="3">42</CS>
		<CS id="4">57</CS>
		<CS id="5">71</CS>
		<CS id="6">85</CS>
		<CS id="7">100</CS>
	</CSs>
	<CTs>
		<CT id="0" Name="Zone 1">
			<TrgL Lo="-800" B1="2" B2="0" B3="0" B4="165"/>
			<TrgH Hi="384" B1="1" B2="0" B3="0" B4="165"/>
		</CT>
		<CT id="1" Name="Zone 2">
			<TrgL Lo="-800" B1="192" B2="0" B3="0" B4="0"/>
			<TrgH Hi="1600" B1="192" B2="0" B3="0" B4="0"/>
		</CT>
		<CT id="2" Name="Hot Water">
			<TrgL Lo="-800" B1="192" B2="0" B3="0" B4="0"/>
			<TrgH Hi="1600" B1="192" B2="0" B3="0" B4="0"/>
		</CT>
		<CT id="3" Name="External">
			<TrgL Lo="-800" B1="192" B2="0" B3="0" B4="0"/>
			<TrgH Hi="1600" B1="192" B2="0" B3="0" B4="0"/>
		</CT>
		<CT id="4" Name="Spare">
			<TrgL Lo="-800" B1="192" B2="0" B3="0" B4="0"/>
			<TrgH Hi="1600" B1="192" B2="0" B3="0" B4="0"/>
		</CT>
	</CTs>
	<CIs>
		<CI id="0" Name="Water Lev">
			<TrgL Lo="0" B1="192" B2="0" B3="0" B4="165"/>
			<TrgH Hi="100" B1="0" B2="0" B3="0" B4="165"/>
		</CI>
		<CI id="1" Name="Salt Lev">
			<TrgL Lo="0" B1="192" B2="0" B3="0" B4="0"/>
			<TrgH Hi="100" B1="0" B2="0" B3="0" B4="0"/>
		</CI>
		<CI id="2" Name="Wind">
			<TrgL Lo="0" B1="192" B2="0" B3="0" B4="0"/>
			<TrgH Hi="100" B1="0" B2="0" B3="0" B4="0"/>
		</CI>
		<CI id="3" Name="Rain Gaug">
			<TrgL Lo="0" B1="192" B2="0" B3="0" B4="0"/>
			<TrgH Hi="100" B1="0" B2="0" B3="0" B4="0"/>
		</CI>
	</CIs>
	<CEs>
		<CE id="0" Days="127" Hours="8" Mins="59">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="1" Days="127" Hours="9" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="2" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="3" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="4" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="5" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="6" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="7" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="8" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="9" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="10" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="11" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="12" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="13" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="14" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
		<CE id="15" Days="0" Hours="0" Mins="0">
			<Trg B1="0" B2="64" B3="0" B4="0"/>
		</CE>
	</CEs>
	<NOs>
		<NO id="0" Name="Boiler"/>
		<NO id="1" Name="Hot Water"/>
		<NO id="2" Name="Sec Light"/>
		<NO id="3" Name="Garage"/>
		<NO id="4" Name="Up Lights"/>
		<NO id="5" Name="Down Ligh"/>
		<NO id="6" Name="Heat Flr"/>
		<NO id="7" Name="Spare"/>
	</NOs>
	<NAs>
		<NA id="0" Name="HallWay"/>
		<NA id="1" Name="External"/>
		<NA id="2" Name="Master Be"/>
		<NA id="3" Name="Library"/>
	</NAs>
	<MM lo="2" hi="63" dig="1985229328" an="-1" fr="8"/>
</WebbrickConfig>`

	fmt.Println("\n\n*** Setting up\n==============\n\n")
	fmt.Printf("%v", strMsg)
	msg := []byte(strMsg)

	// // Decode XML encoding
	var _wbs WebbrickConfig                   // create container to load the xml
	reader := bytes.NewReader(msg)            // create a new reader for transcoding to utf-8
	decoder := xml.NewDecoder(reader)         // create a new xml decoder
	decoder.CharsetReader = charset.NewReader // bind the reader to the decoder
	fmt.Println("*** Decoding\n")
	xmlerr := decoder.Decode(&_wbs) // unmarshall the xml
	if xmlerr != nil {
		fmt.Printf("error: %v", xmlerr)
		return false
	}

	fmt.Println("*** Result\n")

	fmt.Printf("%+v\n\n\n", _wbs)

	return true

}
