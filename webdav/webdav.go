package webdav

import (
	"encoding/xml"
	"time"
)

type Multistatus struct {
	XMLName   xml.Name
	Responses []Response `xml:"response"`
}

type Response struct {
	XMLName xml.Name
	Href string `xml:"href"`
	Props Propstat `xml:"propstat"` // TODO: seems this should be an array
}

type Propstat struct {
	XMLName xml.Name
	Prop Prop `xml:"prop"`
}

type Prop struct {
	XMLName xml.Name
	LastModified string `xml:"getlastmodified"`
	QuotaUsedBytes int64 `xml:"quota-used-bytes"`
	QuotaAvailableBytes int64 `xml:"quota-available-bytes"`
	ETag string `xml:"getetag"`
	ContentLength int64 `xml:"getcontentlength"`
	ContentType string `xml:"getcontenttype"`
	ResourceType ResourceType `xml:"resourcetype"`
}

func (p Prop) GetLastModifiedTime() time.Time {
	t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", p.LastModified)
	if err != nil {
		return time.Now()
	}
	return t
}

type ResourceType struct {
	XMLName xml.Name
	Collection *struct{} `xml:"collection"`
}