package autonatv2

import "time"

// autoNATSettings is used to configure AutoNAT
type autoNATSettings struct {
	allowPrivateAddrs bool
	serverRPM         int
	serverPerPeerRPM  int
	serverDialDataRPM int
	dataRequestPolicy dataRequestPolicyFunc
	now               func() time.Time
}

func defaultSettings() *autoNATSettings {
	return &autoNATSettings{
		allowPrivateAddrs: false,
		// TODO: confirm rate limiting defaults
		serverRPM:         20,
		serverPerPeerRPM:  2,
		serverDialDataRPM: 5,
		dataRequestPolicy: amplificationAttackPrevention,
		now:               time.Now,
	}
}

type AutoNATOption func(s *autoNATSettings) error

func WithServerRateLimit(rpm, perPeerRPM, dialDataRPM int) AutoNATOption {
	return func(s *autoNATSettings) error {
		s.serverRPM = rpm
		s.serverPerPeerRPM = perPeerRPM
		s.serverDialDataRPM = dialDataRPM
		return nil
	}
}

func withDataRequestPolicy(drp dataRequestPolicyFunc) AutoNATOption {
	return func(s *autoNATSettings) error {
		s.dataRequestPolicy = drp
		return nil
	}
}

func allowPrivateAddrs(s *autoNATSettings) error {
	s.allowPrivateAddrs = true
	return nil
}
