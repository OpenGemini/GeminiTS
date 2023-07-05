// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: github.com/openGemini/openGemini/lib/netstorage/data/data.proto

package netstorage_data

import (
	fmt "fmt"
	math "math"

	proto "github.com/gogo/protobuf/proto"
	proto1 "github.com/openGemini/openGemini/open_src/influx/meta/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type SeriesKeysRequest struct {
	Db                   *string  `protobuf:"bytes,1,req,name=Db" json:"Db,omitempty"`
	PtIDs                []uint32 `protobuf:"varint,2,rep,name=PtIDs" json:"PtIDs,omitempty"`
	Measurements         []string `protobuf:"bytes,3,rep,name=Measurements" json:"Measurements,omitempty"`
	Condition            *string  `protobuf:"bytes,4,opt,name=condition" json:"condition,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SeriesKeysRequest) Reset()         { *m = SeriesKeysRequest{} }
func (m *SeriesKeysRequest) String() string { return proto.CompactTextString(m) }
func (*SeriesKeysRequest) ProtoMessage()    {}
func (*SeriesKeysRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{0}
}
func (m *SeriesKeysRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SeriesKeysRequest.Unmarshal(m, b)
}
func (m *SeriesKeysRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SeriesKeysRequest.Marshal(b, m, deterministic)
}
func (m *SeriesKeysRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SeriesKeysRequest.Merge(m, src)
}
func (m *SeriesKeysRequest) XXX_Size() int {
	return xxx_messageInfo_SeriesKeysRequest.Size(m)
}
func (m *SeriesKeysRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SeriesKeysRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SeriesKeysRequest proto.InternalMessageInfo

func (m *SeriesKeysRequest) GetDb() string {
	if m != nil && m.Db != nil {
		return *m.Db
	}
	return ""
}

func (m *SeriesKeysRequest) GetPtIDs() []uint32 {
	if m != nil {
		return m.PtIDs
	}
	return nil
}

func (m *SeriesKeysRequest) GetMeasurements() []string {
	if m != nil {
		return m.Measurements
	}
	return nil
}

func (m *SeriesKeysRequest) GetCondition() string {
	if m != nil && m.Condition != nil {
		return *m.Condition
	}
	return ""
}

type SeriesKeysResponse struct {
	Series               []string `protobuf:"bytes,1,rep,name=Series" json:"Series,omitempty"`
	Err                  *string  `protobuf:"bytes,2,opt,name=Err" json:"Err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SeriesKeysResponse) Reset()         { *m = SeriesKeysResponse{} }
func (m *SeriesKeysResponse) String() string { return proto.CompactTextString(m) }
func (*SeriesKeysResponse) ProtoMessage()    {}
func (*SeriesKeysResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{1}
}
func (m *SeriesKeysResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SeriesKeysResponse.Unmarshal(m, b)
}
func (m *SeriesKeysResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SeriesKeysResponse.Marshal(b, m, deterministic)
}
func (m *SeriesKeysResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SeriesKeysResponse.Merge(m, src)
}
func (m *SeriesKeysResponse) XXX_Size() int {
	return xxx_messageInfo_SeriesKeysResponse.Size(m)
}
func (m *SeriesKeysResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_SeriesKeysResponse.DiscardUnknown(m)
}

var xxx_messageInfo_SeriesKeysResponse proto.InternalMessageInfo

func (m *SeriesKeysResponse) GetSeries() []string {
	if m != nil {
		return m.Series
	}
	return nil
}

func (m *SeriesKeysResponse) GetErr() string {
	if m != nil && m.Err != nil {
		return *m.Err
	}
	return ""
}

type CreateDataBaseRequest struct {
	Db                   *string  `protobuf:"bytes,1,req,name=Db" json:"Db,omitempty"`
	Pt                   *uint32  `protobuf:"varint,2,req,name=pt" json:"pt,omitempty"`
	Rp                   *string  `protobuf:"bytes,3,req,name=rp" json:"rp,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreateDataBaseRequest) Reset()         { *m = CreateDataBaseRequest{} }
func (m *CreateDataBaseRequest) String() string { return proto.CompactTextString(m) }
func (*CreateDataBaseRequest) ProtoMessage()    {}
func (*CreateDataBaseRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{2}
}
func (m *CreateDataBaseRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateDataBaseRequest.Unmarshal(m, b)
}
func (m *CreateDataBaseRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateDataBaseRequest.Marshal(b, m, deterministic)
}
func (m *CreateDataBaseRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateDataBaseRequest.Merge(m, src)
}
func (m *CreateDataBaseRequest) XXX_Size() int {
	return xxx_messageInfo_CreateDataBaseRequest.Size(m)
}
func (m *CreateDataBaseRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateDataBaseRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CreateDataBaseRequest proto.InternalMessageInfo

func (m *CreateDataBaseRequest) GetDb() string {
	if m != nil && m.Db != nil {
		return *m.Db
	}
	return ""
}

func (m *CreateDataBaseRequest) GetPt() uint32 {
	if m != nil && m.Pt != nil {
		return *m.Pt
	}
	return 0
}

func (m *CreateDataBaseRequest) GetRp() string {
	if m != nil && m.Rp != nil {
		return *m.Rp
	}
	return ""
}

type CreateDataBaseResponse struct {
	Err                  *string  `protobuf:"bytes,1,opt,name=Err" json:"Err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CreateDataBaseResponse) Reset()         { *m = CreateDataBaseResponse{} }
func (m *CreateDataBaseResponse) String() string { return proto.CompactTextString(m) }
func (*CreateDataBaseResponse) ProtoMessage()    {}
func (*CreateDataBaseResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{3}
}
func (m *CreateDataBaseResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CreateDataBaseResponse.Unmarshal(m, b)
}
func (m *CreateDataBaseResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CreateDataBaseResponse.Marshal(b, m, deterministic)
}
func (m *CreateDataBaseResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CreateDataBaseResponse.Merge(m, src)
}
func (m *CreateDataBaseResponse) XXX_Size() int {
	return xxx_messageInfo_CreateDataBaseResponse.Size(m)
}
func (m *CreateDataBaseResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_CreateDataBaseResponse.DiscardUnknown(m)
}

var xxx_messageInfo_CreateDataBaseResponse proto.InternalMessageInfo

func (m *CreateDataBaseResponse) GetErr() string {
	if m != nil && m.Err != nil {
		return *m.Err
	}
	return ""
}

type SysCtrlRequest struct {
	Mod                  *string           `protobuf:"bytes,1,req,name=Mod" json:"Mod,omitempty"`
	Param                map[string]string `protobuf:"bytes,2,rep,name=Param" json:"Param,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *SysCtrlRequest) Reset()         { *m = SysCtrlRequest{} }
func (m *SysCtrlRequest) String() string { return proto.CompactTextString(m) }
func (*SysCtrlRequest) ProtoMessage()    {}
func (*SysCtrlRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{4}
}
func (m *SysCtrlRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SysCtrlRequest.Unmarshal(m, b)
}
func (m *SysCtrlRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SysCtrlRequest.Marshal(b, m, deterministic)
}
func (m *SysCtrlRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SysCtrlRequest.Merge(m, src)
}
func (m *SysCtrlRequest) XXX_Size() int {
	return xxx_messageInfo_SysCtrlRequest.Size(m)
}
func (m *SysCtrlRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_SysCtrlRequest.DiscardUnknown(m)
}

var xxx_messageInfo_SysCtrlRequest proto.InternalMessageInfo

func (m *SysCtrlRequest) GetMod() string {
	if m != nil && m.Mod != nil {
		return *m.Mod
	}
	return ""
}

func (m *SysCtrlRequest) GetParam() map[string]string {
	if m != nil {
		return m.Param
	}
	return nil
}

type SysCtrlResponse struct {
	Err                  *string           `protobuf:"bytes,1,req,name=Err" json:"Err,omitempty"`
	Result               map[string]string `protobuf:"bytes,2,rep,name=Result" json:"Result,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *SysCtrlResponse) Reset()         { *m = SysCtrlResponse{} }
func (m *SysCtrlResponse) String() string { return proto.CompactTextString(m) }
func (*SysCtrlResponse) ProtoMessage()    {}
func (*SysCtrlResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{5}
}
func (m *SysCtrlResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SysCtrlResponse.Unmarshal(m, b)
}
func (m *SysCtrlResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SysCtrlResponse.Marshal(b, m, deterministic)
}
func (m *SysCtrlResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SysCtrlResponse.Merge(m, src)
}
func (m *SysCtrlResponse) XXX_Size() int {
	return xxx_messageInfo_SysCtrlResponse.Size(m)
}
func (m *SysCtrlResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_SysCtrlResponse.DiscardUnknown(m)
}

var xxx_messageInfo_SysCtrlResponse proto.InternalMessageInfo

func (m *SysCtrlResponse) GetErr() string {
	if m != nil && m.Err != nil {
		return *m.Err
	}
	return ""
}

func (m *SysCtrlResponse) GetResult() map[string]string {
	if m != nil {
		return m.Result
	}
	return nil
}

type GetShardSplitPointsRequest struct {
	DB                   *string  `protobuf:"bytes,1,req,name=DB" json:"DB,omitempty"`
	PtID                 *uint32  `protobuf:"varint,2,req,name=PtID" json:"PtID,omitempty"`
	ShardID              *uint64  `protobuf:"varint,3,req,name=ShardID" json:"ShardID,omitempty"`
	Idxes                []int64  `protobuf:"varint,4,rep,name=Idxes" json:"Idxes,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetShardSplitPointsRequest) Reset()         { *m = GetShardSplitPointsRequest{} }
func (m *GetShardSplitPointsRequest) String() string { return proto.CompactTextString(m) }
func (*GetShardSplitPointsRequest) ProtoMessage()    {}
func (*GetShardSplitPointsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{6}
}
func (m *GetShardSplitPointsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetShardSplitPointsRequest.Unmarshal(m, b)
}
func (m *GetShardSplitPointsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetShardSplitPointsRequest.Marshal(b, m, deterministic)
}
func (m *GetShardSplitPointsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetShardSplitPointsRequest.Merge(m, src)
}
func (m *GetShardSplitPointsRequest) XXX_Size() int {
	return xxx_messageInfo_GetShardSplitPointsRequest.Size(m)
}
func (m *GetShardSplitPointsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetShardSplitPointsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetShardSplitPointsRequest proto.InternalMessageInfo

func (m *GetShardSplitPointsRequest) GetDB() string {
	if m != nil && m.DB != nil {
		return *m.DB
	}
	return ""
}

func (m *GetShardSplitPointsRequest) GetPtID() uint32 {
	if m != nil && m.PtID != nil {
		return *m.PtID
	}
	return 0
}

func (m *GetShardSplitPointsRequest) GetShardID() uint64 {
	if m != nil && m.ShardID != nil {
		return *m.ShardID
	}
	return 0
}

func (m *GetShardSplitPointsRequest) GetIdxes() []int64 {
	if m != nil {
		return m.Idxes
	}
	return nil
}

type GetShardSplitPointsResponse struct {
	SplitPoints          []string `protobuf:"bytes,1,rep,name=SplitPoints" json:"SplitPoints,omitempty"`
	Err                  *string  `protobuf:"bytes,2,opt,name=Err" json:"Err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetShardSplitPointsResponse) Reset()         { *m = GetShardSplitPointsResponse{} }
func (m *GetShardSplitPointsResponse) String() string { return proto.CompactTextString(m) }
func (*GetShardSplitPointsResponse) ProtoMessage()    {}
func (*GetShardSplitPointsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{7}
}
func (m *GetShardSplitPointsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetShardSplitPointsResponse.Unmarshal(m, b)
}
func (m *GetShardSplitPointsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetShardSplitPointsResponse.Marshal(b, m, deterministic)
}
func (m *GetShardSplitPointsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetShardSplitPointsResponse.Merge(m, src)
}
func (m *GetShardSplitPointsResponse) XXX_Size() int {
	return xxx_messageInfo_GetShardSplitPointsResponse.Size(m)
}
func (m *GetShardSplitPointsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_GetShardSplitPointsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_GetShardSplitPointsResponse proto.InternalMessageInfo

func (m *GetShardSplitPointsResponse) GetSplitPoints() []string {
	if m != nil {
		return m.SplitPoints
	}
	return nil
}

func (m *GetShardSplitPointsResponse) GetErr() string {
	if m != nil && m.Err != nil {
		return *m.Err
	}
	return ""
}

type DeleteRequest struct {
	DB                   *string  `protobuf:"bytes,1,req,name=DB" json:"DB,omitempty"`
	Rp                   *string  `protobuf:"bytes,2,opt,name=Rp" json:"Rp,omitempty"`
	Mst                  *string  `protobuf:"bytes,3,opt,name=Mst" json:"Mst,omitempty"`
	ShardIDs             []uint64 `protobuf:"varint,4,rep,name=ShardIDs" json:"ShardIDs,omitempty"`
	DeleteType           *int32   `protobuf:"varint,5,req,name=DeleteType" json:"DeleteType,omitempty"`
	PtId                 *uint32  `protobuf:"varint,6,opt,name=PtId" json:"PtId,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteRequest) Reset()         { *m = DeleteRequest{} }
func (m *DeleteRequest) String() string { return proto.CompactTextString(m) }
func (*DeleteRequest) ProtoMessage()    {}
func (*DeleteRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{8}
}
func (m *DeleteRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteRequest.Unmarshal(m, b)
}
func (m *DeleteRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteRequest.Marshal(b, m, deterministic)
}
func (m *DeleteRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteRequest.Merge(m, src)
}
func (m *DeleteRequest) XXX_Size() int {
	return xxx_messageInfo_DeleteRequest.Size(m)
}
func (m *DeleteRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteRequest proto.InternalMessageInfo

func (m *DeleteRequest) GetDB() string {
	if m != nil && m.DB != nil {
		return *m.DB
	}
	return ""
}

func (m *DeleteRequest) GetRp() string {
	if m != nil && m.Rp != nil {
		return *m.Rp
	}
	return ""
}

func (m *DeleteRequest) GetMst() string {
	if m != nil && m.Mst != nil {
		return *m.Mst
	}
	return ""
}

func (m *DeleteRequest) GetShardIDs() []uint64 {
	if m != nil {
		return m.ShardIDs
	}
	return nil
}

func (m *DeleteRequest) GetDeleteType() int32 {
	if m != nil && m.DeleteType != nil {
		return *m.DeleteType
	}
	return 0
}

func (m *DeleteRequest) GetPtId() uint32 {
	if m != nil && m.PtId != nil {
		return *m.PtId
	}
	return 0
}

type DeleteResponse struct {
	Err                  *string  `protobuf:"bytes,1,opt,name=Err" json:"Err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DeleteResponse) Reset()         { *m = DeleteResponse{} }
func (m *DeleteResponse) String() string { return proto.CompactTextString(m) }
func (*DeleteResponse) ProtoMessage()    {}
func (*DeleteResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{9}
}
func (m *DeleteResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DeleteResponse.Unmarshal(m, b)
}
func (m *DeleteResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DeleteResponse.Marshal(b, m, deterministic)
}
func (m *DeleteResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DeleteResponse.Merge(m, src)
}
func (m *DeleteResponse) XXX_Size() int {
	return xxx_messageInfo_DeleteResponse.Size(m)
}
func (m *DeleteResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DeleteResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DeleteResponse proto.InternalMessageInfo

func (m *DeleteResponse) GetErr() string {
	if m != nil && m.Err != nil {
		return *m.Err
	}
	return ""
}

type ShowTagValuesRequest struct {
	Db                   *string       `protobuf:"bytes,1,req,name=Db" json:"Db,omitempty"`
	PtIDs                []uint32      `protobuf:"varint,2,rep,name=PtIDs" json:"PtIDs,omitempty"`
	TagKeys              []*MapTagKeys `protobuf:"bytes,3,rep,name=TagKeys" json:"TagKeys,omitempty"`
	Condition            *string       `protobuf:"bytes,4,opt,name=Condition" json:"Condition,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *ShowTagValuesRequest) Reset()         { *m = ShowTagValuesRequest{} }
func (m *ShowTagValuesRequest) String() string { return proto.CompactTextString(m) }
func (*ShowTagValuesRequest) ProtoMessage()    {}
func (*ShowTagValuesRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{10}
}
func (m *ShowTagValuesRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ShowTagValuesRequest.Unmarshal(m, b)
}
func (m *ShowTagValuesRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ShowTagValuesRequest.Marshal(b, m, deterministic)
}
func (m *ShowTagValuesRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ShowTagValuesRequest.Merge(m, src)
}
func (m *ShowTagValuesRequest) XXX_Size() int {
	return xxx_messageInfo_ShowTagValuesRequest.Size(m)
}
func (m *ShowTagValuesRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ShowTagValuesRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ShowTagValuesRequest proto.InternalMessageInfo

func (m *ShowTagValuesRequest) GetDb() string {
	if m != nil && m.Db != nil {
		return *m.Db
	}
	return ""
}

func (m *ShowTagValuesRequest) GetPtIDs() []uint32 {
	if m != nil {
		return m.PtIDs
	}
	return nil
}

func (m *ShowTagValuesRequest) GetTagKeys() []*MapTagKeys {
	if m != nil {
		return m.TagKeys
	}
	return nil
}

func (m *ShowTagValuesRequest) GetCondition() string {
	if m != nil && m.Condition != nil {
		return *m.Condition
	}
	return ""
}

type ShowTagValuesResponse struct {
	Err                  *string           `protobuf:"bytes,1,opt,name=Err" json:"Err,omitempty"`
	Values               []*TagValuesSlice `protobuf:"bytes,2,rep,name=Values" json:"Values,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ShowTagValuesResponse) Reset()         { *m = ShowTagValuesResponse{} }
func (m *ShowTagValuesResponse) String() string { return proto.CompactTextString(m) }
func (*ShowTagValuesResponse) ProtoMessage()    {}
func (*ShowTagValuesResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{11}
}
func (m *ShowTagValuesResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ShowTagValuesResponse.Unmarshal(m, b)
}
func (m *ShowTagValuesResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ShowTagValuesResponse.Marshal(b, m, deterministic)
}
func (m *ShowTagValuesResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ShowTagValuesResponse.Merge(m, src)
}
func (m *ShowTagValuesResponse) XXX_Size() int {
	return xxx_messageInfo_ShowTagValuesResponse.Size(m)
}
func (m *ShowTagValuesResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ShowTagValuesResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ShowTagValuesResponse proto.InternalMessageInfo

func (m *ShowTagValuesResponse) GetErr() string {
	if m != nil && m.Err != nil {
		return *m.Err
	}
	return ""
}

func (m *ShowTagValuesResponse) GetValues() []*TagValuesSlice {
	if m != nil {
		return m.Values
	}
	return nil
}

type MapTagKeys struct {
	Measurement          *string  `protobuf:"bytes,1,req,name=Measurement" json:"Measurement,omitempty"`
	Keys                 []string `protobuf:"bytes,2,rep,name=Keys" json:"Keys,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MapTagKeys) Reset()         { *m = MapTagKeys{} }
func (m *MapTagKeys) String() string { return proto.CompactTextString(m) }
func (*MapTagKeys) ProtoMessage()    {}
func (*MapTagKeys) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{12}
}
func (m *MapTagKeys) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MapTagKeys.Unmarshal(m, b)
}
func (m *MapTagKeys) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MapTagKeys.Marshal(b, m, deterministic)
}
func (m *MapTagKeys) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MapTagKeys.Merge(m, src)
}
func (m *MapTagKeys) XXX_Size() int {
	return xxx_messageInfo_MapTagKeys.Size(m)
}
func (m *MapTagKeys) XXX_DiscardUnknown() {
	xxx_messageInfo_MapTagKeys.DiscardUnknown(m)
}

var xxx_messageInfo_MapTagKeys proto.InternalMessageInfo

func (m *MapTagKeys) GetMeasurement() string {
	if m != nil && m.Measurement != nil {
		return *m.Measurement
	}
	return ""
}

func (m *MapTagKeys) GetKeys() []string {
	if m != nil {
		return m.Keys
	}
	return nil
}

type TagValuesSlice struct {
	Measurement          *string  `protobuf:"bytes,1,req,name=Measurement" json:"Measurement,omitempty"`
	Keys                 []string `protobuf:"bytes,2,rep,name=Keys" json:"Keys,omitempty"`
	Values               []string `protobuf:"bytes,3,rep,name=Values" json:"Values,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TagValuesSlice) Reset()         { *m = TagValuesSlice{} }
func (m *TagValuesSlice) String() string { return proto.CompactTextString(m) }
func (*TagValuesSlice) ProtoMessage()    {}
func (*TagValuesSlice) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{13}
}
func (m *TagValuesSlice) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TagValuesSlice.Unmarshal(m, b)
}
func (m *TagValuesSlice) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TagValuesSlice.Marshal(b, m, deterministic)
}
func (m *TagValuesSlice) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TagValuesSlice.Merge(m, src)
}
func (m *TagValuesSlice) XXX_Size() int {
	return xxx_messageInfo_TagValuesSlice.Size(m)
}
func (m *TagValuesSlice) XXX_DiscardUnknown() {
	xxx_messageInfo_TagValuesSlice.DiscardUnknown(m)
}

var xxx_messageInfo_TagValuesSlice proto.InternalMessageInfo

func (m *TagValuesSlice) GetMeasurement() string {
	if m != nil && m.Measurement != nil {
		return *m.Measurement
	}
	return ""
}

func (m *TagValuesSlice) GetKeys() []string {
	if m != nil {
		return m.Keys
	}
	return nil
}

func (m *TagValuesSlice) GetValues() []string {
	if m != nil {
		return m.Values
	}
	return nil
}

type ExactCardinalityResponse struct {
	Cardinality          map[string]uint64 `protobuf:"bytes,1,rep,name=Cardinality" json:"Cardinality,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
	Err                  *string           `protobuf:"bytes,2,opt,name=Err" json:"Err,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ExactCardinalityResponse) Reset()         { *m = ExactCardinalityResponse{} }
func (m *ExactCardinalityResponse) String() string { return proto.CompactTextString(m) }
func (*ExactCardinalityResponse) ProtoMessage()    {}
func (*ExactCardinalityResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{14}
}
func (m *ExactCardinalityResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ExactCardinalityResponse.Unmarshal(m, b)
}
func (m *ExactCardinalityResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ExactCardinalityResponse.Marshal(b, m, deterministic)
}
func (m *ExactCardinalityResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ExactCardinalityResponse.Merge(m, src)
}
func (m *ExactCardinalityResponse) XXX_Size() int {
	return xxx_messageInfo_ExactCardinalityResponse.Size(m)
}
func (m *ExactCardinalityResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ExactCardinalityResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ExactCardinalityResponse proto.InternalMessageInfo

func (m *ExactCardinalityResponse) GetCardinality() map[string]uint64 {
	if m != nil {
		return m.Cardinality
	}
	return nil
}

func (m *ExactCardinalityResponse) GetErr() string {
	if m != nil && m.Err != nil {
		return *m.Err
	}
	return ""
}

type PtRequest struct {
	Pt                   *proto1.DbPt `protobuf:"bytes,1,req,name=Pt" json:"Pt,omitempty"`
	MigrateType          *int32       `protobuf:"varint,2,req,name=MigrateType" json:"MigrateType,omitempty"`
	OpId                 *uint64      `protobuf:"varint,3,req,name=OpId" json:"OpId,omitempty"`
	AliveConnId          *uint64      `protobuf:"varint,4,opt,name=AliveConnId" json:"AliveConnId,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *PtRequest) Reset()         { *m = PtRequest{} }
func (m *PtRequest) String() string { return proto.CompactTextString(m) }
func (*PtRequest) ProtoMessage()    {}
func (*PtRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{15}
}
func (m *PtRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PtRequest.Unmarshal(m, b)
}
func (m *PtRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PtRequest.Marshal(b, m, deterministic)
}
func (m *PtRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PtRequest.Merge(m, src)
}
func (m *PtRequest) XXX_Size() int {
	return xxx_messageInfo_PtRequest.Size(m)
}
func (m *PtRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_PtRequest.DiscardUnknown(m)
}

var xxx_messageInfo_PtRequest proto.InternalMessageInfo

func (m *PtRequest) GetPt() *proto1.DbPt {
	if m != nil {
		return m.Pt
	}
	return nil
}

func (m *PtRequest) GetMigrateType() int32 {
	if m != nil && m.MigrateType != nil {
		return *m.MigrateType
	}
	return 0
}

func (m *PtRequest) GetOpId() uint64 {
	if m != nil && m.OpId != nil {
		return *m.OpId
	}
	return 0
}

func (m *PtRequest) GetAliveConnId() uint64 {
	if m != nil && m.AliveConnId != nil {
		return *m.AliveConnId
	}
	return 0
}

type PtResponse struct {
	Err                  *string  `protobuf:"bytes,1,opt,name=Err" json:"Err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *PtResponse) Reset()         { *m = PtResponse{} }
func (m *PtResponse) String() string { return proto.CompactTextString(m) }
func (*PtResponse) ProtoMessage()    {}
func (*PtResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_f083076cc39891e5, []int{16}
}
func (m *PtResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PtResponse.Unmarshal(m, b)
}
func (m *PtResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PtResponse.Marshal(b, m, deterministic)
}
func (m *PtResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PtResponse.Merge(m, src)
}
func (m *PtResponse) XXX_Size() int {
	return xxx_messageInfo_PtResponse.Size(m)
}
func (m *PtResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_PtResponse.DiscardUnknown(m)
}

var xxx_messageInfo_PtResponse proto.InternalMessageInfo

func (m *PtResponse) GetErr() string {
	if m != nil && m.Err != nil {
		return *m.Err
	}
	return ""
}

func init() {
	proto.RegisterType((*SeriesKeysRequest)(nil), "netstorage.data.SeriesKeysRequest")
	proto.RegisterType((*SeriesKeysResponse)(nil), "netstorage.data.SeriesKeysResponse")
	proto.RegisterType((*CreateDataBaseRequest)(nil), "netstorage.data.CreateDataBaseRequest")
	proto.RegisterType((*CreateDataBaseResponse)(nil), "netstorage.data.CreateDataBaseResponse")
	proto.RegisterType((*SysCtrlRequest)(nil), "netstorage.data.SysCtrlRequest")
	proto.RegisterMapType((map[string]string)(nil), "netstorage.data.SysCtrlRequest.ParamEntry")
	proto.RegisterType((*SysCtrlResponse)(nil), "netstorage.data.SysCtrlResponse")
	proto.RegisterMapType((map[string]string)(nil), "netstorage.data.SysCtrlResponse.ResultEntry")
	proto.RegisterType((*GetShardSplitPointsRequest)(nil), "netstorage.data.GetShardSplitPointsRequest")
	proto.RegisterType((*GetShardSplitPointsResponse)(nil), "netstorage.data.GetShardSplitPointsResponse")
	proto.RegisterType((*DeleteRequest)(nil), "netstorage.data.DeleteRequest")
	proto.RegisterType((*DeleteResponse)(nil), "netstorage.data.DeleteResponse")
	proto.RegisterType((*ShowTagValuesRequest)(nil), "netstorage.data.ShowTagValuesRequest")
	proto.RegisterType((*ShowTagValuesResponse)(nil), "netstorage.data.ShowTagValuesResponse")
	proto.RegisterType((*MapTagKeys)(nil), "netstorage.data.MapTagKeys")
	proto.RegisterType((*TagValuesSlice)(nil), "netstorage.data.TagValuesSlice")
	proto.RegisterType((*ExactCardinalityResponse)(nil), "netstorage.data.ExactCardinalityResponse")
	proto.RegisterMapType((map[string]uint64)(nil), "netstorage.data.ExactCardinalityResponse.CardinalityEntry")
	proto.RegisterType((*PtRequest)(nil), "netstorage.data.PtRequest")
	proto.RegisterType((*PtResponse)(nil), "netstorage.data.PtResponse")
}

func init() {
	proto.RegisterFile("github.com/openGemini/openGemini/lib/netstorage/data/data.proto", fileDescriptor_f083076cc39891e5)
}

var fileDescriptor_f083076cc39891e5 = []byte{
	// 786 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x55, 0x6d, 0x6b, 0x2b, 0x45,
	0x14, 0x66, 0x67, 0x93, 0x5c, 0x73, 0xd6, 0xe6, 0xf6, 0x8e, 0xf7, 0x5e, 0x96, 0x54, 0xea, 0xb2,
	0x20, 0x84, 0x22, 0x1b, 0x08, 0x88, 0xb5, 0x42, 0xd1, 0x64, 0x4b, 0x09, 0x12, 0x8c, 0x93, 0xe2,
	0x07, 0x11, 0x65, 0x92, 0x1d, 0xd3, 0xa5, 0xdb, 0xdd, 0x75, 0x66, 0x52, 0x1b, 0xfc, 0xe2, 0x5f,
	0xe8, 0x1f, 0xf0, 0x93, 0xbf, 0xc6, 0x5f, 0x25, 0xf3, 0x92, 0x64, 0x93, 0x34, 0x48, 0xfd, 0x12,
	0x66, 0x9e, 0x3d, 0x2f, 0xcf, 0x39, 0xe7, 0x39, 0x13, 0x38, 0xcd, 0xd2, 0x69, 0x37, 0x67, 0x52,
	0xc8, 0x82, 0xd3, 0x39, 0xeb, 0x26, 0x54, 0x52, 0xfd, 0x13, 0x95, 0xbc, 0x90, 0x05, 0x7e, 0xbd,
	0xf9, 0x16, 0x29, 0xb8, 0xfd, 0x69, 0x51, 0xb2, 0xfc, 0x17, 0xc1, 0x67, 0xdd, 0x34, 0xff, 0x35,
	0x5b, 0x3c, 0x76, 0xef, 0x99, 0xa4, 0x5d, 0x6d, 0xac, 0x8f, 0xc6, 0x2f, 0xfc, 0x03, 0xde, 0x4c,
	0x18, 0x4f, 0x99, 0xf8, 0x96, 0x2d, 0x05, 0x61, 0xbf, 0x2d, 0x98, 0x90, 0xb8, 0x05, 0x28, 0x9e,
	0xfa, 0x4e, 0x80, 0x3a, 0x4d, 0x82, 0xe2, 0x29, 0x7e, 0x0b, 0xf5, 0xb1, 0x1c, 0xc6, 0xc2, 0x47,
	0x81, 0xdb, 0x39, 0x22, 0xe6, 0x82, 0x43, 0xf8, 0x70, 0xc4, 0xa8, 0x58, 0x70, 0x76, 0xcf, 0x72,
	0x29, 0x7c, 0x37, 0x70, 0x3b, 0x4d, 0xb2, 0x85, 0xe1, 0x8f, 0xa1, 0x39, 0x2b, 0xf2, 0x24, 0x95,
	0x69, 0x91, 0xfb, 0xb5, 0xc0, 0xe9, 0x34, 0xc9, 0x06, 0x08, 0x2f, 0x01, 0x57, 0x93, 0x8b, 0xb2,
	0xc8, 0x05, 0xc3, 0xef, 0xa1, 0x61, 0x50, 0xdf, 0xd1, 0x11, 0xed, 0x0d, 0x1f, 0x83, 0x7b, 0xc5,
	0xb9, 0x8f, 0x74, 0x14, 0x75, 0x0c, 0xaf, 0xe1, 0xdd, 0x80, 0x33, 0x2a, 0x59, 0x4c, 0x25, 0xed,
	0x53, 0xc1, 0x0e, 0x15, 0xd0, 0x02, 0x54, 0x4a, 0x1f, 0x05, 0xa8, 0x73, 0x44, 0x50, 0xa9, 0xbf,
	0xf3, 0xd2, 0x77, 0xcd, 0x77, 0x5e, 0x86, 0x67, 0xf0, 0x7e, 0x37, 0x90, 0x25, 0x63, 0x93, 0x3a,
	0x9b, 0xa4, 0x7f, 0x39, 0xd0, 0x9a, 0x2c, 0xc5, 0x40, 0xf2, 0x6c, 0x95, 0xee, 0x18, 0xdc, 0x51,
	0x91, 0xd8, 0x7c, 0xea, 0x88, 0xbf, 0x86, 0xfa, 0x98, 0x72, 0x7a, 0xaf, 0x3b, 0xe6, 0xf5, 0xce,
	0xa2, 0x9d, 0xf1, 0x44, 0xdb, 0x11, 0x22, 0x6d, 0x7c, 0x95, 0x4b, 0xbe, 0x24, 0xc6, 0xb1, 0x7d,
	0x0e, 0xb0, 0x01, 0x55, 0x86, 0x3b, 0xb6, 0x5c, 0xd1, 0xb8, 0x63, 0x4b, 0x35, 0x93, 0x07, 0x9a,
	0x2d, 0x98, 0xed, 0x87, 0xb9, 0x5c, 0xa0, 0x73, 0x27, 0xfc, 0xdb, 0x81, 0xd7, 0xeb, 0xf0, 0xbb,
	0x65, 0x20, 0x5b, 0x06, 0x8e, 0xa1, 0x41, 0x98, 0x58, 0x64, 0xd2, 0x52, 0xfc, 0xec, 0x30, 0x45,
	0x13, 0x23, 0x32, 0xe6, 0x86, 0xa4, 0xf5, 0x6d, 0x7f, 0x09, 0x5e, 0x05, 0x7e, 0x11, 0xcd, 0x12,
	0xda, 0xd7, 0x4c, 0x4e, 0x6e, 0x29, 0x4f, 0x26, 0x65, 0x96, 0xca, 0x71, 0x91, 0xe6, 0x72, 0x4b,
	0x82, 0xfd, 0xf5, 0x04, 0xfb, 0x18, 0x43, 0x4d, 0xa9, 0xce, 0xce, 0x50, 0x9f, 0xb1, 0x0f, 0xaf,
	0xb4, 0xfb, 0x30, 0xd6, 0xa3, 0xac, 0x91, 0xd5, 0x55, 0x65, 0x1d, 0x26, 0x8f, 0x4c, 0xf8, 0xb5,
	0xc0, 0xed, 0xb8, 0xc4, 0x5c, 0xc2, 0xef, 0xe1, 0xe4, 0xd9, 0x8c, 0xb6, 0x47, 0x01, 0x78, 0x15,
	0xd8, 0x8a, 0xaf, 0x0a, 0x3d, 0xa3, 0xc0, 0x27, 0x07, 0x8e, 0x62, 0x96, 0x31, 0xc9, 0x0e, 0x11,
	0x6f, 0x01, 0x22, 0xa5, 0x75, 0x41, 0xa4, 0xd4, 0x5a, 0x11, 0xd2, 0x77, 0x4d, 0x8c, 0x91, 0x90,
	0xb8, 0x0d, 0x1f, 0x58, 0xde, 0x86, 0x6f, 0x8d, 0xac, 0xef, 0xf8, 0x14, 0xc0, 0x84, 0xbf, 0x59,
	0x96, 0xcc, 0xaf, 0x07, 0xa8, 0x53, 0x27, 0x15, 0xc4, 0xb6, 0x25, 0xf1, 0x1b, 0x81, 0x63, 0xdb,
	0x92, 0x84, 0x21, 0xb4, 0x56, 0x94, 0x0e, 0x8a, 0xf8, 0xc9, 0x81, 0xb7, 0x93, 0xdb, 0xe2, 0xf7,
	0x1b, 0x3a, 0xff, 0x41, 0x4d, 0xe4, 0x85, 0xab, 0xff, 0x39, 0xbc, 0xba, 0xa1, 0x73, 0xb5, 0xb5,
	0x7a, 0xeb, 0xbd, 0xde, 0xc9, 0x9e, 0x7a, 0x46, 0xb4, 0xb4, 0x26, 0x64, 0x65, 0xab, 0x5e, 0x83,
	0xc1, 0xee, 0x6b, 0xb0, 0x06, 0xc2, 0x29, 0xbc, 0xdb, 0xa1, 0x74, 0x88, 0x3e, 0xfe, 0x02, 0x1a,
	0xc6, 0xc6, 0x8a, 0xf7, 0x93, 0xbd, 0xf4, 0xeb, 0x28, 0x93, 0x2c, 0x9d, 0x31, 0x62, 0xcd, 0xc3,
	0x3e, 0xc0, 0x86, 0x98, 0x9a, 0x78, 0xe5, 0xb5, 0xb2, 0x55, 0x57, 0x21, 0xd5, 0x5f, 0x5d, 0x25,
	0xd2, 0x62, 0xd0, 0xe7, 0xf0, 0x67, 0x68, 0x6d, 0x47, 0xff, 0x7f, 0x71, 0xd4, 0x3b, 0x67, 0x8b,
	0x30, 0x2f, 0xe7, 0x8a, 0xe3, 0x3f, 0x0e, 0xf8, 0x57, 0x8f, 0x74, 0x26, 0x07, 0x94, 0x27, 0x69,
	0x4e, 0xb3, 0x54, 0x2e, 0xd7, 0xbd, 0xf8, 0x09, 0xbc, 0x0a, 0xac, 0x45, 0xea, 0xf5, 0x2e, 0xf6,
	0xca, 0x3f, 0xe4, 0x1f, 0x55, 0x30, 0xb3, 0xc9, 0xd5, 0x70, 0xfb, 0x02, 0x6f, 0x5f, 0xc2, 0xf1,
	0xae, 0xcb, 0x7f, 0x6d, 0x79, 0xad, 0xba, 0xe5, 0x7f, 0x3a, 0xd0, 0x1c, 0xcb, 0x95, 0xba, 0x4e,
	0x00, 0x8d, 0x4d, 0x7f, 0xbc, 0x9e, 0x67, 0xfe, 0x81, 0xa2, 0x78, 0x3a, 0x96, 0x04, 0x8d, 0xa5,
	0xee, 0x62, 0x3a, 0xe7, 0xd4, 0x8a, 0x1d, 0x69, 0xb1, 0x57, 0x21, 0xd5, 0xc5, 0xef, 0xca, 0x61,
	0x62, 0xb7, 0x5d, 0x9f, 0x95, 0xd7, 0x37, 0x59, 0xfa, 0xc0, 0x06, 0x45, 0x9e, 0x0f, 0x13, 0xad,
	0xaa, 0x1a, 0xa9, 0x42, 0xe1, 0x29, 0x80, 0x62, 0x70, 0x48, 0x4c, 0xfd, 0x8f, 0x7e, 0x7c, 0x13,
	0x7d, 0xb5, 0xd3, 0xc0, 0x7f, 0x03, 0x00, 0x00, 0xff, 0xff, 0x12, 0x59, 0xcc, 0xb7, 0x70, 0x07,
	0x00, 0x00,
}
