// Code generated by protoc-gen-go.
// source: crawler.proto
// DO NOT EDIT!

/*
Package crawlerpb is a generated protocol buffer package.

It is generated from these files:
	crawler.proto

It has these top-level messages:
	Crawler
	Record
*/
package crawlerpb

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Crawler struct {
	Domain           *string  `protobuf:"bytes,1,req,name=domain" json:"domain,omitempty"`
	UserAgent        *string  `protobuf:"bytes,2,opt,name=user_agent" json:"user_agent,omitempty"`
	RobotsAgent      *string  `protobuf:"bytes,3,opt,name=robots_agent" json:"robots_agent,omitempty"`
	Seeds            []string `protobuf:"bytes,4,rep,name=seeds" json:"seeds,omitempty"`
	Accept           []string `protobuf:"bytes,5,rep,name=accept" json:"accept,omitempty"`
	Reject           []string `protobuf:"bytes,6,rep,name=reject" json:"reject,omitempty"`
	MaxVisit         *uint32  `protobuf:"varint,7,opt,name=max_visit" json:"max_visit,omitempty"`
	TimeToLive       *int64   `protobuf:"varint,8,opt,name=time_to_live" json:"time_to_live,omitempty"`
	Delay            *int64   `protobuf:"varint,9,opt,name=delay" json:"delay,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Crawler) Reset()         { *m = Crawler{} }
func (m *Crawler) String() string { return proto.CompactTextString(m) }
func (*Crawler) ProtoMessage()    {}

func (m *Crawler) GetDomain() string {
	if m != nil && m.Domain != nil {
		return *m.Domain
	}
	return ""
}

func (m *Crawler) GetUserAgent() string {
	if m != nil && m.UserAgent != nil {
		return *m.UserAgent
	}
	return ""
}

func (m *Crawler) GetRobotsAgent() string {
	if m != nil && m.RobotsAgent != nil {
		return *m.RobotsAgent
	}
	return ""
}

func (m *Crawler) GetSeeds() []string {
	if m != nil {
		return m.Seeds
	}
	return nil
}

func (m *Crawler) GetAccept() []string {
	if m != nil {
		return m.Accept
	}
	return nil
}

func (m *Crawler) GetReject() []string {
	if m != nil {
		return m.Reject
	}
	return nil
}

func (m *Crawler) GetMaxVisit() uint32 {
	if m != nil && m.MaxVisit != nil {
		return *m.MaxVisit
	}
	return 0
}

func (m *Crawler) GetTimeToLive() int64 {
	if m != nil && m.TimeToLive != nil {
		return *m.TimeToLive
	}
	return 0
}

func (m *Crawler) GetDelay() int64 {
	if m != nil && m.Delay != nil {
		return *m.Delay
	}
	return 0
}

type Record struct {
	URL              *string  `protobuf:"bytes,1,req" json:"URL,omitempty"`
	Key              []byte   `protobuf:"bytes,2,req,name=key" json:"key,omitempty"`
	Title            *string  `protobuf:"bytes,3,opt,name=title" json:"title,omitempty"`
	Language         *string  `protobuf:"bytes,4,opt,name=language" json:"language,omitempty"`
	Description      *string  `protobuf:"bytes,5,opt,name=description" json:"description,omitempty"`
	Author           *string  `protobuf:"bytes,6,opt,name=author" json:"author,omitempty"`
	Generator        *string  `protobuf:"bytes,7,opt,name=generator" json:"generator,omitempty"`
	Copyright        *string  `protobuf:"bytes,8,opt,name=copyright" json:"copyright,omitempty"`
	Keywords         []string `protobuf:"bytes,9,rep,name=keywords" json:"keywords,omitempty"`
	Robots           []string `protobuf:"bytes,10,rep,name=robots" json:"robots,omitempty"`
	External         []string `protobuf:"bytes,11,rep,name=external" json:"external,omitempty"`
	Links            []string `protobuf:"bytes,12,rep,name=links" json:"links,omitempty"`
	Scripts          []string `protobuf:"bytes,13,rep,name=scripts" json:"scripts,omitempty"`
	Body             []byte   `protobuf:"bytes,14,opt,name=body" json:"body,omitempty"`
	Checksum         []byte   `protobuf:"bytes,15,opt,name=checksum" json:"checksum,omitempty"`
	Data             []byte   `protobuf:"bytes,16,opt,name=data" json:"data,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Record) Reset()         { *m = Record{} }
func (m *Record) String() string { return proto.CompactTextString(m) }
func (*Record) ProtoMessage()    {}

func (m *Record) GetURL() string {
	if m != nil && m.URL != nil {
		return *m.URL
	}
	return ""
}

func (m *Record) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *Record) GetTitle() string {
	if m != nil && m.Title != nil {
		return *m.Title
	}
	return ""
}

func (m *Record) GetLanguage() string {
	if m != nil && m.Language != nil {
		return *m.Language
	}
	return ""
}

func (m *Record) GetDescription() string {
	if m != nil && m.Description != nil {
		return *m.Description
	}
	return ""
}

func (m *Record) GetAuthor() string {
	if m != nil && m.Author != nil {
		return *m.Author
	}
	return ""
}

func (m *Record) GetGenerator() string {
	if m != nil && m.Generator != nil {
		return *m.Generator
	}
	return ""
}

func (m *Record) GetCopyright() string {
	if m != nil && m.Copyright != nil {
		return *m.Copyright
	}
	return ""
}

func (m *Record) GetKeywords() []string {
	if m != nil {
		return m.Keywords
	}
	return nil
}

func (m *Record) GetRobots() []string {
	if m != nil {
		return m.Robots
	}
	return nil
}

func (m *Record) GetExternal() []string {
	if m != nil {
		return m.External
	}
	return nil
}

func (m *Record) GetLinks() []string {
	if m != nil {
		return m.Links
	}
	return nil
}

func (m *Record) GetScripts() []string {
	if m != nil {
		return m.Scripts
	}
	return nil
}

func (m *Record) GetBody() []byte {
	if m != nil {
		return m.Body
	}
	return nil
}

func (m *Record) GetChecksum() []byte {
	if m != nil {
		return m.Checksum
	}
	return nil
}

func (m *Record) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
}
