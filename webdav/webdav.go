package webdav

import "encoding/xml"

type Multistatus struct {
	XMLName   xml.Name
	Responses []Response `xml:"response"`
}

type Response struct {
	XMLName xml.Name
	Href string `xml:"href"`
	Props Propstat `xml:"propstat"`
}

type Propstat struct {
	XMLName xml.Name
	Prop Prop `xml:"prop"`
}

type Prop struct {
	XMLName xml.Name
	ResourceType ResourceType `xml:"resourcetype"`
}

type ResourceType struct {
	XMLName xml.Name
	Collection *struct{} `xml:"collection"`
}