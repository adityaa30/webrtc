//go:build !js
// +build !js

package webrtc

import (
	"regexp"
	"strings"
	"testing"

	"github.com/pion/sdp/v3"
	"github.com/pion/transport/test"
	"github.com/stretchr/testify/assert"
)

// pion/webrtc#1078
func TestOpusCase(t *testing.T) {
	pc, err := NewPeerConnection(Configuration{})
	assert.NoError(t, err)

	_, err = pc.AddTransceiverFromKind(RTPCodecTypeAudio)
	assert.NoError(t, err)

	offer, err := pc.CreateOffer(nil)
	assert.NoError(t, err)

	assert.True(t, regexp.MustCompile(`(?m)^a=rtpmap:\d+ opus/48000/2`).MatchString(offer.SDP))
	assert.NoError(t, pc.Close())
}

// pion/example-webrtc-applications#89
func TestVideoCase(t *testing.T) {
	pc, err := NewPeerConnection(Configuration{})
	assert.NoError(t, err)

	_, err = pc.AddTransceiverFromKind(RTPCodecTypeVideo)
	assert.NoError(t, err)

	offer, err := pc.CreateOffer(nil)
	assert.NoError(t, err)

	assert.True(t, regexp.MustCompile(`(?m)^a=rtpmap:\d+ H264/90000`).MatchString(offer.SDP))
	assert.True(t, regexp.MustCompile(`(?m)^a=rtpmap:\d+ VP8/90000`).MatchString(offer.SDP))
	assert.True(t, regexp.MustCompile(`(?m)^a=rtpmap:\d+ VP9/90000`).MatchString(offer.SDP))
	assert.NoError(t, pc.Close())
}

func TestMediaEngineRemoteDescription(t *testing.T) {
	mustParse := func(raw string) sdp.SessionDescription {
		s := sdp.SessionDescription{}
		assert.NoError(t, s.Unmarshal([]byte(raw)))
		return s
	}

	t.Run("No Media", func(t *testing.T) {
		const noMedia = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
`
		m := MediaEngine{}
		assert.NoError(t, m.RegisterDefaultCodecs())
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(noMedia)))

		assert.False(t, m.negotiatedVideo)
		assert.False(t, m.negotiatedAudio)
	})

	t.Run("Enable Opus", func(t *testing.T) {
		const opusSamePayload = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=audio 9 UDP/TLS/RTP/SAVPF 111
a=rtpmap:111 opus/48000/2
a=fmtp:111 minptime=10; useinbandfec=1
`

		m := MediaEngine{}
		assert.NoError(t, m.RegisterDefaultCodecs())
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(opusSamePayload)))

		assert.False(t, m.negotiatedVideo)
		assert.True(t, m.negotiatedAudio)

		opusCodec, _, err := m.getCodecByPayload(111)
		assert.NoError(t, err)
		assert.Equal(t, opusCodec.MimeType, MimeTypeOpus)
	})

	t.Run("Change Payload Type", func(t *testing.T) {
		const opusSamePayload = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=audio 9 UDP/TLS/RTP/SAVPF 112
a=rtpmap:112 opus/48000/2
a=fmtp:112 minptime=10; useinbandfec=1
`

		m := MediaEngine{}
		assert.NoError(t, m.RegisterDefaultCodecs())
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(opusSamePayload)))

		assert.False(t, m.negotiatedVideo)
		assert.True(t, m.negotiatedAudio)

		_, _, err := m.getCodecByPayload(111)
		assert.Error(t, err)

		opusCodec, _, err := m.getCodecByPayload(112)
		assert.NoError(t, err)
		assert.Equal(t, opusCodec.MimeType, MimeTypeOpus)
	})

	t.Run("Ambiguous Payload Type", func(t *testing.T) {
		const opusSamePayload = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=audio 9 UDP/TLS/RTP/SAVPF 96
a=rtpmap:96 opus/48000/2
a=fmtp:96 minptime=10; useinbandfec=1
`

		m := MediaEngine{}
		assert.NoError(t, m.RegisterDefaultCodecs())
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(opusSamePayload)))

		assert.False(t, m.negotiatedVideo)
		assert.True(t, m.negotiatedAudio)

		opusCodec, _, err := m.getCodecByPayload(96)
		assert.NoError(t, err)
		assert.Equal(t, opusCodec.MimeType, MimeTypeOpus)
	})

	t.Run("Case Insensitive", func(t *testing.T) {
		const opusUpcase = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=audio 9 UDP/TLS/RTP/SAVPF 111
a=rtpmap:111 OPUS/48000/2
a=fmtp:111 minptime=10; useinbandfec=1
`

		m := MediaEngine{}
		assert.NoError(t, m.RegisterDefaultCodecs())
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(opusUpcase)))

		assert.False(t, m.negotiatedVideo)
		assert.True(t, m.negotiatedAudio)

		opusCodec, _, err := m.getCodecByPayload(111)
		assert.NoError(t, err)
		assert.Equal(t, opusCodec.MimeType, "audio/OPUS")
	})

	t.Run("Handle different fmtp", func(t *testing.T) {
		const opusNoFmtp = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=audio 9 UDP/TLS/RTP/SAVPF 111
a=rtpmap:111 opus/48000/2
`

		m := MediaEngine{}
		assert.NoError(t, m.RegisterDefaultCodecs())
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(opusNoFmtp)))

		assert.False(t, m.negotiatedVideo)
		assert.True(t, m.negotiatedAudio)

		opusCodec, _, err := m.getCodecByPayload(111)
		assert.NoError(t, err)
		assert.Equal(t, opusCodec.MimeType, MimeTypeOpus)
	})

	t.Run("Header Extensions", func(t *testing.T) {
		const headerExtensions = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=audio 9 UDP/TLS/RTP/SAVPF 111
a=extmap:7 urn:ietf:params:rtp-hdrext:sdes:mid
a=extmap:5 urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id
a=rtpmap:111 opus/48000/2
`

		m := MediaEngine{}
		assert.NoError(t, m.RegisterDefaultCodecs())
		registerSimulcastHeaderExtensions(&m, RTPCodecTypeAudio)
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(headerExtensions)))

		assert.False(t, m.negotiatedVideo)
		assert.True(t, m.negotiatedAudio)

		absID, absAudioEnabled, absVideoEnabled := m.getHeaderExtensionID(RTPHeaderExtensionCapability{sdp.ABSSendTimeURI})
		assert.Equal(t, absID, 0)
		assert.False(t, absAudioEnabled)
		assert.False(t, absVideoEnabled)

		midID, midAudioEnabled, midVideoEnabled := m.getHeaderExtensionID(RTPHeaderExtensionCapability{sdp.SDESMidURI})
		assert.Equal(t, midID, 7)
		assert.True(t, midAudioEnabled)
		assert.False(t, midVideoEnabled)
	})

	t.Run("Prefers exact codec matches", func(t *testing.T) {
		const profileLevels = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=video 60323 UDP/TLS/RTP/SAVPF 96 98
a=rtpmap:96 H264/90000
a=fmtp:96 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=640c1f
a=rtpmap:98 H264/90000
a=fmtp:98 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f
`
		m := MediaEngine{}
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", nil},
			PayloadType:        127,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(profileLevels)))

		assert.True(t, m.negotiatedVideo)
		assert.False(t, m.negotiatedAudio)

		supportedH264, _, err := m.getCodecByPayload(98)
		assert.NoError(t, err)
		assert.Equal(t, supportedH264.MimeType, MimeTypeH264)

		_, _, err = m.getCodecByPayload(96)
		assert.Error(t, err)
	})

	t.Run("Does not match when fmtpline is set and does not match", func(t *testing.T) {
		const profileLevels = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=video 60323 UDP/TLS/RTP/SAVPF 96 98
a=rtpmap:96 H264/90000
a=fmtp:96 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=640c1f
`
		m := MediaEngine{}
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", nil},
			PayloadType:        127,
		}, RTPCodecTypeVideo))
		assert.Error(t, m.updateFromRemoteDescription(mustParse(profileLevels)))

		_, _, err := m.getCodecByPayload(96)
		assert.Error(t, err)
	})

	t.Run("Matches when fmtpline is not set in offer, but exists in mediaengine", func(t *testing.T) {
		const profileLevels = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=video 60323 UDP/TLS/RTP/SAVPF 96
a=rtpmap:96 VP9/90000
`
		m := MediaEngine{}
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeVP9, 90000, 0, "profile-id=0", nil},
			PayloadType:        98,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(profileLevels)))

		assert.True(t, m.negotiatedVideo)

		_, _, err := m.getCodecByPayload(96)
		assert.NoError(t, err)
	})

	t.Run("Matches when fmtpline exists in neither", func(t *testing.T) {
		const profileLevels = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=video 60323 UDP/TLS/RTP/SAVPF 96
a=rtpmap:96 VP8/90000
`
		m := MediaEngine{}
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeVP8, 90000, 0, "", nil},
			PayloadType:        96,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(profileLevels)))

		assert.True(t, m.negotiatedVideo)

		_, _, err := m.getCodecByPayload(96)
		assert.NoError(t, err)
	})

	t.Run("Matches when rtx apt for exact match codec", func(t *testing.T) {
		const profileLevels = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=video 60323 UDP/TLS/RTP/SAVPF 94 96 97
a=rtpmap:94 VP8/90000
a=rtpmap:96 VP9/90000
a=fmtp:96 profile-id=2
a=rtpmap:97 rtx/90000
a=fmtp:97 apt=96
`
		m := MediaEngine{}
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeVP8, 90000, 0, "", nil},
			PayloadType:        94,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeVP9, 90000, 0, "profile-id=2", nil},
			PayloadType:        96,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{"video/rtx", 90000, 0, "apt=96", nil},
			PayloadType:        97,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(profileLevels)))

		assert.True(t, m.negotiatedVideo)

		_, _, err := m.getCodecByPayload(97)
		assert.NoError(t, err)
	})

	t.Run("Matches when rtx apt for partial match codec", func(t *testing.T) {
		const profileLevels = `v=0
o=- 4596489990601351948 2 IN IP4 127.0.0.1
s=-
t=0 0
m=video 60323 UDP/TLS/RTP/SAVPF 94 96 97
a=rtpmap:94 VP8/90000
a=rtpmap:96 VP9/90000
a=fmtp:96 profile-id=2
a=rtpmap:97 rtx/90000
a=fmtp:97 apt=96
`
		m := MediaEngine{}
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeVP8, 90000, 0, "", nil},
			PayloadType:        94,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeVP9, 90000, 0, "profile-id=1", nil},
			PayloadType:        96,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.RegisterCodec(RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{"video/rtx", 90000, 0, "apt=96", nil},
			PayloadType:        97,
		}, RTPCodecTypeVideo))
		assert.NoError(t, m.updateFromRemoteDescription(mustParse(profileLevels)))

		assert.True(t, m.negotiatedVideo)

		_, _, err := m.getCodecByPayload(97)
		assert.ErrorIs(t, err, ErrCodecNotFound)
	})
}

func TestMediaEngineHeaderExtensionDirection(t *testing.T) {
	report := test.CheckRoutines(t)
	defer report()

	registerCodec := func(m *MediaEngine) {
		assert.NoError(t, m.RegisterCodec(
			RTPCodecParameters{
				RTPCodecCapability: RTPCodecCapability{MimeTypeOpus, 48000, 0, "", nil},
				PayloadType:        111,
			}, RTPCodecTypeAudio))
	}

	t.Run("No Direction", func(t *testing.T) {
		m := &MediaEngine{}
		registerCodec(m)
		assert.NoError(t, m.RegisterHeaderExtension(RTPHeaderExtensionCapability{"pion-header-test"}, RTPCodecTypeAudio))

		params := m.getRTPParametersByKind(RTPCodecTypeAudio, []RTPTransceiverDirection{RTPTransceiverDirectionRecvonly})

		assert.Equal(t, 1, len(params.HeaderExtensions))
	})

	t.Run("Same Direction", func(t *testing.T) {
		m := &MediaEngine{}
		registerCodec(m)
		assert.NoError(t, m.RegisterHeaderExtension(RTPHeaderExtensionCapability{"pion-header-test"}, RTPCodecTypeAudio, RTPTransceiverDirectionRecvonly))

		params := m.getRTPParametersByKind(RTPCodecTypeAudio, []RTPTransceiverDirection{RTPTransceiverDirectionRecvonly})

		assert.Equal(t, 1, len(params.HeaderExtensions))
	})

	t.Run("Different Direction", func(t *testing.T) {
		m := &MediaEngine{}
		registerCodec(m)
		assert.NoError(t, m.RegisterHeaderExtension(RTPHeaderExtensionCapability{"pion-header-test"}, RTPCodecTypeAudio, RTPTransceiverDirectionSendonly))

		params := m.getRTPParametersByKind(RTPCodecTypeAudio, []RTPTransceiverDirection{RTPTransceiverDirectionRecvonly})

		assert.Equal(t, 0, len(params.HeaderExtensions))
	})

	t.Run("Invalid Direction", func(t *testing.T) {
		m := &MediaEngine{}
		registerCodec(m)

		assert.ErrorIs(t, m.RegisterHeaderExtension(RTPHeaderExtensionCapability{"pion-header-test"}, RTPCodecTypeAudio, RTPTransceiverDirectionSendrecv), ErrRegisterHeaderExtensionInvalidDirection)
		assert.ErrorIs(t, m.RegisterHeaderExtension(RTPHeaderExtensionCapability{"pion-header-test"}, RTPCodecTypeAudio, RTPTransceiverDirectionInactive), ErrRegisterHeaderExtensionInvalidDirection)
		assert.ErrorIs(t, m.RegisterHeaderExtension(RTPHeaderExtensionCapability{"pion-header-test"}, RTPCodecTypeAudio, RTPTransceiverDirection(0)), ErrRegisterHeaderExtensionInvalidDirection)
	})
}

// If a user attempts to register a codec twice we should just discard duplicate calls
func TestMediaEngineDoubleRegister(t *testing.T) {
	m := MediaEngine{}

	assert.NoError(t, m.RegisterCodec(
		RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeOpus, 48000, 0, "", nil},
			PayloadType:        111,
		}, RTPCodecTypeAudio))

	assert.NoError(t, m.RegisterCodec(
		RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeOpus, 48000, 0, "", nil},
			PayloadType:        111,
		}, RTPCodecTypeAudio))

	assert.Equal(t, len(m.audioCodecs), 1)
}

// The cloned MediaEngine instance should be able to update negotiated header extensions.
func TestUpdateHeaderExtenstionToClonedMediaEngine(t *testing.T) {
	src := MediaEngine{}

	assert.NoError(t, src.RegisterCodec(
		RTPCodecParameters{
			RTPCodecCapability: RTPCodecCapability{MimeTypeOpus, 48000, 0, "", nil},
			PayloadType:        111,
		}, RTPCodecTypeAudio))

	assert.NoError(t, src.RegisterHeaderExtension(RTPHeaderExtensionCapability{"test-extension"}, RTPCodecTypeAudio))

	validate := func(m *MediaEngine) {
		assert.NoError(t, m.updateHeaderExtension(2, "test-extension", RTPCodecTypeAudio))

		id, audioNegotiated, videoNegotiated := m.getHeaderExtensionID(RTPHeaderExtensionCapability{URI: "test-extension"})
		assert.Equal(t, 2, id)
		assert.True(t, audioNegotiated)
		assert.False(t, videoNegotiated)
	}

	validate(&src)
	validate(src.copy())
}

func TestMediaEngine_updateFromRemoteDescription(t *testing.T) {
	me, me2 := &MediaEngine{HMSHotFix: true}, &MediaEngine{}
	se := SettingEngine{}
	se.DisableMediaEngineCopy(true)

	feedback := []RTCPFeedback{
		{Type: TypeRTCPFBTransportCC},
		{Type: TypeRTCPFBCCM, Parameter: "fir"},
		{Type: TypeRTCPFBNACK},
		{Type: TypeRTCPFBNACK, Parameter: "pli"},
	}
	vp8 := RTPCodecCapability{MimeTypeVP8, 90000, 0, "", feedback}
	h264 := RTPCodecCapability{MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", feedback}

	codecVp8 := RTPCodecParameters{RTPCodecCapability: vp8, PayloadType: 96}
	codecH264 := RTPCodecParameters{RTPCodecCapability: h264, PayloadType: 108}

	assert.NoError(t, me2.RegisterCodec(codecVp8, RTPCodecTypeVideo))
	assert.NoError(t, me2.RegisterCodec(codecH264, RTPCodecTypeVideo))

	config := Configuration{SDPSemantics: SDPSemanticsUnifiedPlan}
	offerer, err := NewAPI(WithMediaEngine(me), WithSettingEngine(se)).NewPeerConnection(config)
	assert.NoError(t, err)

	answerer, err := NewAPI(WithMediaEngine(me2)).NewPeerConnection(config)
	assert.NoError(t, err)

	trackVP8, err := NewTrackLocalStaticSample(vp8, "video", "pion-vp8")
	assert.NoError(t, err)

	trackH264, err := NewTrackLocalStaticSample(h264, "video", "pion-h264")
	assert.NoError(t, err)

	codecs := []RTPCodecParameters{codecH264, codecVp8}
	for i, track := range []TrackLocal{trackH264, trackVP8} {
		assert.NoError(t, me.RegisterCodec(codecs[i], RTPCodecTypeVideo))

		transceiver, err := offerer.AddTransceiverFromTrack(track, RTPTransceiverInit{
			Direction: RTPTransceiverDirectionSendonly,
		})
		assert.NoError(t, err)

		// Have to do this to force the codec
		assert.NoError(t, transceiver.SetCodecPreferences([]RTPCodecParameters{codecs[i]}))

		_, err = answerer.AddTrack(track)
		assert.NoError(t, err)

		assert.NoError(t, signalPair(offerer, answerer))
	}

	assert.NoError(t, offerer.Close())
	assert.NoError(t, answerer.Close())
}

func Test_Something(t *testing.T) {
	me, se := &MediaEngine{}, SettingEngine{}
	se.DisableMediaEngineCopy(true)

	feedback := []RTCPFeedback{
		{Type: TypeRTCPFBTransportCC},
		{Type: TypeRTCPFBCCM, Parameter: "fir"},
		{Type: TypeRTCPFBNACK},
		{Type: TypeRTCPFBNACK, Parameter: "pli"},
	}

	vp8 := RTPCodecCapability{MimeTypeVP8, 90000, 0, "", feedback}
	h264 := RTPCodecCapability{MimeTypeH264, 90000, 0, "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f", feedback}

	codecVp8 := RTPCodecParameters{RTPCodecCapability: vp8, PayloadType: 96}
	codecH264 := RTPCodecParameters{RTPCodecCapability: h264, PayloadType: 108}

	assert.NoError(t, me.RegisterCodec(codecVp8, RTPCodecTypeVideo))
	assert.NoError(t, me.RegisterCodec(codecH264, RTPCodecTypeVideo))
	assert.NoError(t, me.RegisterHeaderExtension(RTPHeaderExtensionCapability{URI: sdp.TransportCCURI}, RTPCodecTypeVideo))
	for _, fb := range feedback {
		me.RegisterFeedback(fb, RTPCodecTypeVideo)
	}

	config := Configuration{SDPSemantics: SDPSemanticsUnifiedPlan}
	pc, err := NewAPI(WithMediaEngine(me), WithSettingEngine(se)).NewPeerConnection(config)
	assert.NoError(t, err)

	sdp := `v=0
o=mozilla..x.xTHIS_IS_SDPARTA-99.0 4347299155470183221 1 IN IP4 0.0.0.0
s=-
t=0 0
a=fingerprint:sha-256 B2:CC:43:D3:A9:C2:53:30:E6:88:3A:F8:4A:0E:AB:FA:1F:47:7A:7D:96:01:8B:E3:34:82:62:1F:24:18:36:49
a=ice-options:trickle
a=msid-semantic: WMS *
a=group:BUNDLE 0 1
m=application 61056 UDP/DTLS/SCTP webrtc-datachannel
c=IN IP4 172.20.x.x
a=setup:actpass
a=mid:0
a=sendrecv
a=ice-ufrag:60679741
a=ice-pwd:f74dcba74e253618ae0d856076abf1e4
a=candidate:0 1 UDP 2122187007 172.20.12.21 61056 typ host
a=candidate:3 1 UDP 2122252543 2409:4071:4e0d:3282:a479:a6e3:5a18:6d7a 64930 typ host
a=candidate:4 1 TCP 2105458943 172.20.12.22 9 typ host tcptype active
a=candidate:5 1 TCP 2105524479 2409:4071:4e0d:3282:a479:a6e3:5a18:6d7a 9 typ host tcptype active
a=sctp-port:5000
a=max-message-size:1073741823
m=video 9 UDP/TLS/RTP/SAVPF 120 124 121 125 126 127 97 98
c=IN IP4 0.0.0.0
a=rtpmap:120 VP8/90000
a=rtpmap:124 rtx/90000
a=rtpmap:121 VP9/90000
a=rtpmap:125 rtx/90000
a=rtpmap:126 H264/90000
a=rtpmap:127 rtx/90000
a=rtpmap:97 H264/90000
a=rtpmap:98 rtx/90000
a=fmtp:126 profile-level-id=42e01f;level-asymmetry-allowed=1;packetization-mode=1
a=fmtp:97 profile-level-id=42e01f;level-asymmetry-allowed=1
a=fmtp:120 max-fs=12288;max-fr=60
a=fmtp:124 apt=120
a=fmtp:121 max-fs=12288;max-fr=60
a=fmtp:125 apt=121
a=fmtp:127 apt=126
a=fmtp:98 apt=97
a=rtcp-fb:120 nack
a=rtcp-fb:120 nack pli
a=rtcp-fb:120 ccm fir
a=rtcp-fb:120 goog-remb
a=rtcp-fb:120 transport-cc
a=rtcp-fb:121 nack
a=rtcp-fb:121 nack pli
a=rtcp-fb:121 ccm fir
a=rtcp-fb:121 goog-remb
a=rtcp-fb:121 transport-cc
a=rtcp-fb:126 nack
a=rtcp-fb:126 nack pli
a=rtcp-fb:126 ccm fir
a=rtcp-fb:126 goog-remb
a=rtcp-fb:126 transport-cc
a=rtcp-fb:97 nack
a=rtcp-fb:97 nack pli
a=rtcp-fb:97 ccm fir
a=rtcp-fb:97 goog-remb
a=rtcp-fb:97 transport-cc
a=extmap:3 urn:ietf:params:rtp-hdrext:sdes:mid
a=extmap:4 http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time
a=extmap:5 urn:ietf:params:rtp-hdrext:toffset
a=extmap:6/recvonly http://www.webrtc.org/experiments/rtp-hdrext/playout-delay
a=extmap:7 http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01
a=setup:actpass
a=mid:1
a=msid:{6be459cd-3239-4c32-ab57-f5e39b8744e7} {305565e3-e580-490e-85b1-f1076dabccc8}
a=sendonly
a=ice-ufrag:60679741
a=ice-pwd:f74dcba74e253618ae0d856076abf1e4
a=ssrc:3703733639 cname:{9ef793e2-26b3-49e4-a736-c1f90cdc54df}
a=ssrc:2939640839 cname:{9ef793e2-26b3-49e4-a736-c1f90cdc54df}
a=ssrc-group:FID 3703733639 2939640839
a=rtcp-mux
a=rtcp-rsize
`
	assert.NoError(t, pc.SetRemoteDescription(SessionDescription{SDP: sdp, Type: SDPTypeOffer}))
	answer, err := pc.CreateAnswer(nil)
	assert.NoError(t, err)

	assert.Equal(t, 1, strings.Count(answer.SDP, "extmap:7"))
}
