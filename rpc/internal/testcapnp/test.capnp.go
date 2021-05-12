// Code generated by capnpc-go. DO NOT EDIT.

package testcapnp

import (
	capnp "capnproto.org/go/capnp/v3"
	text "capnproto.org/go/capnp/v3/encoding/text"
	schemas "capnproto.org/go/capnp/v3/schemas"
	server "capnproto.org/go/capnp/v3/server"
	context "context"
)

type PingPong struct{ Client *capnp.Client }

// PingPong_TypeID is the unique identifier for the type PingPong.
const PingPong_TypeID = 0xf004c474c2f8ee7a

func (c PingPong) EchoNum(ctx context.Context, params func(PingPong_echoNum_Params) error) (PingPong_echoNum_Results_Future, capnp.ReleaseFunc) {
	s := capnp.Send{
		Method: capnp.Method{
			InterfaceID:   0xf004c474c2f8ee7a,
			MethodID:      0,
			InterfaceName: "test.capnp:PingPong",
			MethodName:    "echoNum",
		},
	}
	if params != nil {
		s.ArgsSize = capnp.ObjectSize{DataSize: 8, PointerCount: 0}
		s.PlaceArgs = func(s capnp.Struct) error { return params(PingPong_echoNum_Params{Struct: s}) }
	}
	ans, release := c.Client.SendCall(ctx, s)
	return PingPong_echoNum_Results_Future{Future: ans.Future()}, release
}

func (c PingPong) AddRef() PingPong {
	return PingPong{
		Client: c.Client.AddRef(),
	}
}

func (c PingPong) Release() {
	c.Client.Release()
}

// A PingPong_Server is a PingPong with a local implementation.
type PingPong_Server interface {
	EchoNum(context.Context, PingPong_echoNum) error
}

// PingPong_NewServer creates a new Server from an implementation of PingPong_Server.
func PingPong_NewServer(s PingPong_Server, policy *server.Policy) *server.Server {
	c, _ := s.(server.Shutdowner)
	return server.New(PingPong_Methods(nil, s), s, c, policy)
}

// PingPong_ServerToClient creates a new Client from an implementation of PingPong_Server.
// The caller is responsible for calling Release on the returned Client.
func PingPong_ServerToClient(s PingPong_Server, policy *server.Policy) PingPong {
	return PingPong{Client: capnp.NewClient(PingPong_NewServer(s, policy))}
}

// PingPong_Methods appends Methods to a slice that invoke the methods on s.
// This can be used to create a more complicated Server.
func PingPong_Methods(methods []server.Method, s PingPong_Server) []server.Method {
	if cap(methods) == 0 {
		methods = make([]server.Method, 0, 1)
	}

	methods = append(methods, server.Method{
		Method: capnp.Method{
			InterfaceID:   0xf004c474c2f8ee7a,
			MethodID:      0,
			InterfaceName: "test.capnp:PingPong",
			MethodName:    "echoNum",
		},
		Impl: func(ctx context.Context, call *server.Call) error {
			return s.EchoNum(ctx, PingPong_echoNum{call})
		},
	})

	return methods
}

// PingPong_echoNum holds the state for a server call to PingPong.echoNum.
// See server.Call for documentation.
type PingPong_echoNum struct {
	*server.Call
}

// Args returns the call's arguments.
func (c PingPong_echoNum) Args() PingPong_echoNum_Params {
	return PingPong_echoNum_Params{Struct: c.Call.Args()}
}

// AllocResults allocates the results struct.
func (c PingPong_echoNum) AllocResults() (PingPong_echoNum_Results, error) {
	r, err := c.Call.AllocResults(capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return PingPong_echoNum_Results{Struct: r}, err
}

type PingPong_echoNum_Params struct{ capnp.Struct }

// PingPong_echoNum_Params_TypeID is the unique identifier for the type PingPong_echoNum_Params.
const PingPong_echoNum_Params_TypeID = 0xd797e0a99edf0921

func NewPingPong_echoNum_Params(s *capnp.Segment) (PingPong_echoNum_Params, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return PingPong_echoNum_Params{st}, err
}

func NewRootPingPong_echoNum_Params(s *capnp.Segment) (PingPong_echoNum_Params, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return PingPong_echoNum_Params{st}, err
}

func ReadRootPingPong_echoNum_Params(msg *capnp.Message) (PingPong_echoNum_Params, error) {
	root, err := msg.Root()
	return PingPong_echoNum_Params{root.Struct()}, err
}

func (s PingPong_echoNum_Params) String() string {
	str, _ := text.Marshal(0xd797e0a99edf0921, s.Struct)
	return str
}

func (s PingPong_echoNum_Params) N() int64 {
	return int64(s.Struct.Uint64(0))
}

func (s PingPong_echoNum_Params) SetN(v int64) {
	s.Struct.SetUint64(0, uint64(v))
}

// PingPong_echoNum_Params_List is a list of PingPong_echoNum_Params.
type PingPong_echoNum_Params_List struct{ capnp.List }

// NewPingPong_echoNum_Params creates a new list of PingPong_echoNum_Params.
func NewPingPong_echoNum_Params_List(s *capnp.Segment, sz int32) (PingPong_echoNum_Params_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0}, sz)
	return PingPong_echoNum_Params_List{l}, err
}

func (s PingPong_echoNum_Params_List) At(i int) PingPong_echoNum_Params {
	return PingPong_echoNum_Params{s.List.Struct(i)}
}

func (s PingPong_echoNum_Params_List) Set(i int, v PingPong_echoNum_Params) error {
	return s.List.SetStruct(i, v.Struct)
}

func (s PingPong_echoNum_Params_List) String() string {
	str, _ := text.MarshalList(0xd797e0a99edf0921, s.List)
	return str
}

// PingPong_echoNum_Params_Future is a wrapper for a PingPong_echoNum_Params promised by a client call.
type PingPong_echoNum_Params_Future struct{ *capnp.Future }

func (p PingPong_echoNum_Params_Future) Struct() (PingPong_echoNum_Params, error) {
	s, err := p.Future.Struct()
	return PingPong_echoNum_Params{s}, err
}

type PingPong_echoNum_Results struct{ capnp.Struct }

// PingPong_echoNum_Results_TypeID is the unique identifier for the type PingPong_echoNum_Results.
const PingPong_echoNum_Results_TypeID = 0x85ddfd96db252600

func NewPingPong_echoNum_Results(s *capnp.Segment) (PingPong_echoNum_Results, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return PingPong_echoNum_Results{st}, err
}

func NewRootPingPong_echoNum_Results(s *capnp.Segment) (PingPong_echoNum_Results, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0})
	return PingPong_echoNum_Results{st}, err
}

func ReadRootPingPong_echoNum_Results(msg *capnp.Message) (PingPong_echoNum_Results, error) {
	root, err := msg.Root()
	return PingPong_echoNum_Results{root.Struct()}, err
}

func (s PingPong_echoNum_Results) String() string {
	str, _ := text.Marshal(0x85ddfd96db252600, s.Struct)
	return str
}

func (s PingPong_echoNum_Results) N() int64 {
	return int64(s.Struct.Uint64(0))
}

func (s PingPong_echoNum_Results) SetN(v int64) {
	s.Struct.SetUint64(0, uint64(v))
}

// PingPong_echoNum_Results_List is a list of PingPong_echoNum_Results.
type PingPong_echoNum_Results_List struct{ capnp.List }

// NewPingPong_echoNum_Results creates a new list of PingPong_echoNum_Results.
func NewPingPong_echoNum_Results_List(s *capnp.Segment, sz int32) (PingPong_echoNum_Results_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 0}, sz)
	return PingPong_echoNum_Results_List{l}, err
}

func (s PingPong_echoNum_Results_List) At(i int) PingPong_echoNum_Results {
	return PingPong_echoNum_Results{s.List.Struct(i)}
}

func (s PingPong_echoNum_Results_List) Set(i int, v PingPong_echoNum_Results) error {
	return s.List.SetStruct(i, v.Struct)
}

func (s PingPong_echoNum_Results_List) String() string {
	str, _ := text.MarshalList(0x85ddfd96db252600, s.List)
	return str
}

// PingPong_echoNum_Results_Future is a wrapper for a PingPong_echoNum_Results promised by a client call.
type PingPong_echoNum_Results_Future struct{ *capnp.Future }

func (p PingPong_echoNum_Results_Future) Struct() (PingPong_echoNum_Results, error) {
	s, err := p.Future.Struct()
	return PingPong_echoNum_Results{s}, err
}

const schema_ef12a34b9807e19c = "x\xda\x12\xe8v`2d\xcdgb`\x08\x94ae" +
	"\xfb\xa7\xa6z{\xda\xdf\xbb\xad\x81\"\x8c\x8c\x0c\x0c," +
	"\xec\x0c\x0c\xc6\xb2\x8cJ\x8c\x0c\x8c\xc2\xaa\x8c\xf6\x0c\x8c" +
	"\xff\x159\xef\xcf[\xf9`\xfau\x06$\x05\xae\x8cR" +
	" \x05\xbe`\x05U\xef~\x1c*9\xc2\xf2\x81A\x90" +
	"\x9b\xf9\xff\x9c\x87\xec3\xbc\x17\x0b\xbdg``\x14\xce" +
	"e\\$\\\xca\xc8\xce\xc0 \\\xc8\xe8.<\x93\x91" +
	"\x9dA\xe7\x7fIjq\x89^rb\x01s^\x81U" +
	"@f^z@~^\xba^jrF\xbe_i\xae" +
	"JPjq){NIq \x0b3\x0b\x03\x03\x0b" +
	"#\x03\x83 \xaf\x10\x03C \x073c\xa0\x08\x13#" +
	"c\x1e#+\x03\x13#+\x03#~c\x02\x12\x8b\x12" +
	"\x99sI0\x85\x11f\x0a{~^z\x00#c " +
	"\x0b3+\x03\x03\xdc\xe7\x8c\x0c\xd0 \x12\x14tb`" +
	"\x12de\xaf\x87\xda\xe4\xc0\x18\xc0\xc8\x08\x08\x00\x00\xff" +
	"\xff9\x1eU\xc6"

func init() {
	schemas.Register(schema_ef12a34b9807e19c,
		0x85ddfd96db252600,
		0xd797e0a99edf0921,
		0xf004c474c2f8ee7a)
}
