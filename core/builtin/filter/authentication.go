package filter

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"github.com/idealeak/goserver/core/builtin/protocol"
	"github.com/idealeak/goserver/core/netlib"
)

var (
	AuthenticationFilterName = "session-filter-auth"
	SessionAttributeAuth     = &AuthenticationFilter{}
)

type AuthenticationHandler func(s *netlib.Session, bSuc bool)
type AuthenticationFilter struct {
	SessionAuthHandler AuthenticationHandler
}

func (af *AuthenticationFilter) GetName() string {
	return AuthenticationFilterName
}

func (af *AuthenticationFilter) GetInterestOps() uint {
	return 1<<netlib.InterestOps_Opened | 1<<netlib.InterestOps_Received
}

func (af *AuthenticationFilter) OnSessionOpened(s *netlib.Session, bAccept bool) bool {
	timestamp := time.Now().Unix()
	h := md5.New()
	sc := s.GetSessionConfig()
	h.Write([]byte(fmt.Sprintf("%v;%v", timestamp, sc.AuthKey)))
	authPack := &protocol.SSPacketAuth{
		Timestamp: proto.Int64(timestamp),
		AuthKey:   proto.String(hex.EncodeToString(h.Sum(nil))),
	}
	proto.SetDefaults(authPack)
	s.Send(authPack)
	return true
}

func (af *AuthenticationFilter) OnSessionClosed(s *netlib.Session) bool {
	return true
}

func (af *AuthenticationFilter) OnSessionIdle(s *netlib.Session) bool {
	return true
}

func (af *AuthenticationFilter) OnPacketReceived(s *netlib.Session, packetid int, packet interface{}) bool {
	if s.GetAttribute(SessionAttributeAuth) == nil {
		if auth, ok := packet.(*protocol.SSPacketAuth); ok {
			h := md5.New()
			h.Write([]byte(fmt.Sprintf("%v;%v", auth.GetTimestamp(), s.GetSessionConfig().AuthKey)))
			if hex.EncodeToString(h.Sum(nil)) != auth.GetAuthKey() {
				if af.SessionAuthHandler != nil {
					af.SessionAuthHandler(s, false)
				}
				s.Close()
				return false
			}
			s.SetAttribute(SessionAttributeAuth, true)
			if af.SessionAuthHandler != nil {
				af.SessionAuthHandler(s, true)
			}
			return false
		} else {
			s.Close()
			return false
		}
	}
	return true
}

func (af *AuthenticationFilter) OnPacketSent(s *netlib.Session, data []byte) bool {
	return true
}

func init() {
	netlib.RegisterFactory(int(protocol.CoreBuiltinPacketID_PACKET_SS_AUTH), netlib.PacketFactoryWrapper(func() interface{} {
		return &protocol.SSPacketAuth{}
	}))
	netlib.RegisteSessionFilterCreator(AuthenticationFilterName, func() netlib.SessionFilter {
		return &AuthenticationFilter{}
	})
}
